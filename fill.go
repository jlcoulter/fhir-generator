package fhirgen

import (
	"fmt"
	"slices"
	"strings"

	fhir "github.com/jlcoulter/fhir-registry"
)

// Generate produces a conformant FHIR resource instance for the given base
// type name (e.g. "Patient", "Organization"). The returned map includes the
// "resourceType" key and is suitable for json.Marshal. If the type name is not
// defined in the registry, the returned error wraps ErrDefinitionNotFound.
func (g *Generator) Generate(typeName string) (map[string]any, error) {
	tree, err := g.reg.TreeForType(typeName)
	if err != nil {
		return nil, fmt.Errorf("%w: type %s", ErrDefinitionNotFound, typeName)
	}
	return g.generateFromTree(tree)
}

// GenerateForURL produces a conformant FHIR resource instance for a specific
// StructureDefinition canonical URL (e.g. an AU profile). Unlike Generate,
// which takes a base type name, GenerateForURL honors the full profile
// referenced by the URL. If the URL is not defined, the returned error wraps
// ErrDefinitionNotFound.
func (g *Generator) GenerateForURL(url string) (map[string]any, error) {
	tree, err := g.reg.Tree(url)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDefinitionNotFound, url)
	}
	return g.generateFromTree(tree)
}

func (g *Generator) generateFromTree(tree *fhir.ElementTree) (map[string]any, error) {
	if err := g.validateValues(tree); err != nil {
		return nil, err
	}
	obj, err := g.fillObject(tree.Root, tree)
	if err != nil {
		return nil, err
	}
	obj["resourceType"] = tree.Root.Path
	if g.id != "" {
		obj["id"] = g.id
	}
	if len(g.metaProfiles) > 0 {
		profiles := make([]any, 0, len(g.metaProfiles))
		for _, p := range g.metaProfiles {
			profiles = append(profiles, p)
		}
		obj["meta"] = map[string]any{"profile": profiles}
	}
	g.applyValues(obj, tree)
	g.normaliseIdentifiers(obj)
	if g.normaliser != nil {
		g.normaliser(obj)
	}
	if g.stripEmptyExtensions {
		stripEmptyExtensions(obj)
	}
	enforceExt1(obj)
	return obj, nil
}

// validateValues verifies that every WithValues path resolves against the
// element tree, so a typo fails fast rather than silently producing a resource
// missing the intended data.
func (g *Generator) validateValues(tree *fhir.ElementTree) error {
	for path := range g.values {
		if _, err := tree.LookupPath(path); err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidPath, path)
		}
	}
	return nil
}

// applyValues injects caller-supplied values (from WithValues) into the
// generated resource at their element paths. Paths are relative to the
// resource root (e.g. "name.family", "birthDate"). Values override the
// fake-generated data at those locations.
func (g *Generator) applyValues(obj map[string]any, tree *fhir.ElementTree) {
	if len(g.values) == 0 {
		return
	}
	for path, val := range g.values {
		g.setPath(obj, tree, path, val)
	}
}

// setPath navigates the generated object to the given relative element path and
// sets the value there. Repeating elements are descended into via their first
// instance; a whole-object value for a repeating element is wrapped in an array
// when it is not already a slice.
func (g *Generator) setPath(obj map[string]any, tree *fhir.ElementTree, path string, val any) {
	segs := strings.Split(path, ".")
	cur := any(obj)
	for i := 0; i < len(segs); i++ {
		seg := segs[i]
		last := i == len(segs)-1
		switch node := cur.(type) {
		case map[string]any:
			if last {
				if isMultiPath(tree, path) {
					if _, isSlice := val.([]any); !isSlice {
						val = []any{val}
					}
				}
				node[seg] = val
				return
			}
			next, ok := node[seg]
			if !ok {
				return
			}
			cur = next
		case []any:
			if len(node) == 0 {
				return
			}
			cur = node[0]
			i--
		default:
			return
		}
	}
}

// cloneValue deep-copies a JSON-like value (map[string]any, []any, or scalar)
// so the result never aliases the source. It is used to emit Fixed/Pattern
// values without sharing the registry's element-definition data, which callers
// may mutate.
func cloneValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = cloneValue(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = cloneValue(val)
		}
		return out
	default:
		return v
	}
}

// isMultiPath reports whether the element at the given relative path is
// repeating (max > 1).
func isMultiPath(tree *fhir.ElementTree, path string) bool {
	full := tree.Root.Path + "." + path
	return slices.ContainsFunc(tree.ByPath[full], fhir.IsMulti)
}

// hasDescendantValue reports whether any caller-supplied value path targets the
// given element or one of its descendants. This ensures optional elements that
// carry a provided value are still generated so the value has a home.
func (g *Generator) hasDescendantValue(tree *fhir.ElementTree, elem *fhir.ElementDefinition) bool {
	if len(g.values) == 0 {
		return false
	}
	prefix := elem.Path + "."
	for path := range g.values {
		full := tree.Root.Path + "." + path
		if full == elem.Path || strings.HasPrefix(full, prefix) {
			return true
		}
	}
	return false
}

// baseChildren returns the direct child element definitions of a node, one per
// path (slices collapsed to their base element). Complex-type children with no
// in-tree children are resolved through the registry.
func (g *Generator) baseChildren(tree *fhir.ElementTree, elem *fhir.ElementDefinition) []*fhir.ElementDefinition {
	var children []*fhir.ElementDefinition
	if len(elem.Children) > 0 {
		children = elem.Children
	} else if t, ok := g.reg.ResolveType(fhir.PrimaryTypeCode(elem), profilesOf(elem)); ok {
		children = t.Root.Children
	} else {
		return nil
	}

	seen := make(map[string]bool)
	base := make([]*fhir.ElementDefinition, 0, len(children))
	for _, c := range children {
		if seen[c.Path] {
			continue
		}
		seen[c.Path] = true
		base = append(base, c)
	}
	return base
}

func profilesOf(elem *fhir.ElementDefinition) []string {
	if len(elem.Types) == 0 {
		return nil
	}
	return elem.Types[0].Profiles
}

// isExtensionElement reports whether an element is an extension container
// (extension or modifierExtension).
func isExtensionElement(elem *fhir.ElementDefinition) bool {
	seg := lastSegment(elem.Path)
	return seg == "extension" || seg == "modifierExtension"
}

// maxDepth caps the recursion depth when filling nested/recursive types (e.g.
// Extension.extension is itself an Extension). This prevents infinite
// recursion on self-referential definitions.
const maxDepth = 20

// fillObject fills a JSON object for an element definition.
func (g *Generator) fillObject(elem *fhir.ElementDefinition, tree *fhir.ElementTree) (map[string]any, error) {
	return g.fillObjectDepth(elem, tree, 0)
}

// fillObjectDepth is fillObject with an explicit recursion depth.
func (g *Generator) fillObjectDepth(elem *fhir.ElementDefinition, tree *fhir.ElementTree, depth int) (map[string]any, error) {
	if depth > maxDepth {
		return nil, nil
	}
	out := make(map[string]any)
	children := g.baseChildren(tree, elem)
	for _, child := range children {
		if err := g.fillChild(child, tree, out, depth); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// fillChild fills a single child element into the output object.
func (g *Generator) fillChild(child *fhir.ElementDefinition, tree *fhir.ElementTree, out map[string]any, depth int) error {
	// Choice elements: pick one concrete type and use its suffixed key.
	if fhir.IsChoice(child) {
		return g.fillChoice(child, tree, out, depth)
	}

	key := lastSegment(child.Path)

	// Fixed or pattern values are emitted verbatim. They are deep-cloned so the
	// generated output never aliases the registry's shared element definition
	// data, which callers may mutate (e.g. normalisers that strip display/text).
	if child.Fixed != nil {
		out[key] = cloneValue(child.Fixed)
		return nil
	}
	if child.Pattern != nil {
		out[key] = cloneValue(child.Pattern)
		return nil
	}

	// Sliced elements: emit one instance per slice, using each slice's
	// profile URL for the extension url.
	if len(child.Slices) > 0 {
		return g.fillSlices(child, tree, out, depth)
	}

	// Unsliced extension/modifierExtension elements with no profile hint
	// cannot produce a valid extension URL, so they are skipped.
	if isExtensionElement(child) && len(profilesOf(child)) == 0 {
		return nil
	}

	// Decide whether to fill based on cardinality and fill mode. An element is
	// also filled when a caller-supplied value targets it or one of its
	// descendants, so that provided values always have a home in the output.
	if !g.shouldFill(child) && !g.hasDescendantValue(tree, child) {
		return nil
	}

	val, err := g.fillValue(child, tree, depth)
	if err != nil {
		return err
	}
	if val == nil {
		return nil
	}
	out[key] = val
	return nil
}

// fillSlices emits one value per slice group of a sliced element. Each slice
// is filled using its own definition (which carries the profile URL for
// extensions) and appended to the element's array.
func (g *Generator) fillSlices(elem *fhir.ElementDefinition, tree *fhir.ElementTree, out map[string]any, depth int) error {
	key := lastSegment(elem.Path)
	arr := make([]any, 0, len(elem.Slices))
	for _, sl := range elem.Slices {
		if !g.shouldFill(sl.Definition) {
			continue
		}
		v, err := g.fillSingle(sl.Definition, tree, depth)
		if err != nil {
			return err
		}
		if v != nil {
			// Overlay the slice's own Fixed/Pattern so the generated value
			// matches the slice's discriminator.
			if m, ok := v.(map[string]any); ok {
				applySlicePattern(m, sl.Definition)
				// Apply child Fixed/Pattern values too (e.g. a complex extension
				// slice whose value[x].coding fixes a code, carried a level down).
				applySliceChildPatterns(m, sl.Definition)
			}
			arr = append(arr, v)
		}
	}
	if len(arr) > 0 {
		out[key] = arr
	}
	return nil
}

// applySlicePattern overlays a slice element's Fixed/Pattern value onto a
// generated map. FHIR slices commonly carry their discriminating values as a
// pattern/fixed on the slice element itself, so without applying it a generated
// value does not match the slice.
func applySlicePattern(value map[string]any, def *fhir.ElementDefinition) {
	if value == nil || def == nil {
		return
	}
	var overlay any
	if def.Fixed != nil {
		overlay = def.Fixed
	} else if def.Pattern != nil {
		overlay = def.Pattern
	} else {
		return
	}
	if m, ok := overlay.(map[string]any); ok {
		for k, v := range m {
			if sub, ok := v.(map[string]any); ok {
				mergeSlicePattern(value, k, sub)
			} else {
				value[k] = v
			}
		}
	}
}

// applySliceChildPatterns applies each of the slice's children's Fixed/Pattern
// values onto the generated map, recursing into nested slices. A child that
// itself is sliced (e.g. a complex extension's nested "extension") is handled
// by matching each generated array element to its slice via the fixed "url".
func applySliceChildPatterns(value map[string]any, def *fhir.ElementDefinition) {
	if value == nil || def == nil {
		return
	}
	for _, child := range def.Children {
		if child != nil {
			applySliceChildPattern(value, child)
		}
	}
	for _, sl := range def.Slices {
		if sl != nil && sl.Definition != nil {
			applySliceChildPattern(value, sl.Definition)
		}
	}
}

// applySliceChildPattern applies one child's Fixed/Pattern onto the value at the
// child's JSON key, recursing into the child's own slices or its children when it
// has no direct Fixed/Pattern of its own.
func applySliceChildPattern(value map[string]any, child *fhir.ElementDefinition) {
	if value == nil || child == nil {
		return
	}
	key := childJSONKey(child)
	var overlay any
	switch {
	case child.Fixed != nil:
		overlay = child.Fixed
	case child.Pattern != nil:
		overlay = child.Pattern
	default:
		if len(child.Slices) > 0 {
			applyNestedChildPatterns(value, child, key)
			return
		}
		// No direct fixed/pattern or slices: recurse into the child's own
		// children (e.g. value[x].coding fixing a code) at the generated value.
		raw, ok := value[key]
		if !ok {
			return
		}
		switch t := raw.(type) {
		case map[string]any:
			applySliceChildPatterns(t, child)
		case []any:
			for _, item := range t {
				if m, ok := item.(map[string]any); ok {
					applySliceChildPatterns(m, child)
				}
			}
		}
		return
	}
	if m, ok := overlay.(map[string]any); ok {
		// A Fixed/Pattern value is emitted verbatim with only the fields it
		// defines. A fixed Coding carries system+code and must not gain display
		// or text (servers reject a display/text on a fixed value that defines
		// only system+code).
		m = stripFixedDisplayText(m)
		// If the target value is already an array (e.g. a CodeableConcept's
		// repeating "coding" element), preserve the array shape by wrapping the
		// fixed/pattern value in a single-element array rather than replacing it
		// with a bare object.
		if _, isArr := value[key].([]any); isArr {
			value[key] = []any{cloneValue(m)}
			// The parent CodeableConcept's text (added by fakeCodeableConcept) is
			// stale when its coding is fixed; drop it.
			delete(value, "text")
			return
		}
		mergeSlicePattern(value, key, m)
	} else {
		value[key] = overlay
	}
}

// stripFixedDisplayText removes display/text from a fixed/pattern coding map when
// the fixed value defines only system+code. A CodeableConcept whose coding is
// fixed by the profile may not carry extra display or text.
func stripFixedDisplayText(m map[string]any) map[string]any {
	// Coding: drop display, drop any top-level text.
	delete(m, "display")
	delete(m, "text")
	// CodeableConcept: drop its text and each coding's display.
	if codings, ok := m["coding"].([]any); ok {
		for _, c := range codings {
			if cm, ok := c.(map[string]any); ok {
				delete(cm, "display")
			}
		}
		delete(m, "text")
	}
	return m
}

// applyNestedChildPatterns descends into a child that carries no direct
// Fixed/Pattern but has nested slices (e.g. a complex extension's nested
// "extension" array), applying each matching slice's constraints onto the
// corresponding generated array element.
func applyNestedChildPatterns(value map[string]any, child *fhir.ElementDefinition, key string) {
	if len(child.Slices) == 0 {
		return
	}
	raw, ok := value[key]
	if !ok {
		return
	}
	arr, ok := raw.([]any)
	if !ok {
		return
	}
	for _, sl := range child.Slices {
		if sl == nil || sl.Definition == nil {
			continue
		}
		url := fixedSliceURL(sl.Definition)
		if url == "" {
			continue
		}
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if u, _ := m["url"].(string); u != url {
				continue
			}
			applySlicePattern(m, sl.Definition)
			applySliceChildPatterns(m, sl.Definition)
		}
	}
}

// fixedSliceURL returns the fixed "url" value of a slice element, the
// discriminator matching a generated extension array element to its slice.
func fixedSliceURL(sl *fhir.ElementDefinition) string {
	if sl == nil {
		return ""
	}
	for _, child := range sl.Children {
		if child == nil || lastSegment(child.Path) != "url" {
			continue
		}
		if u, ok := child.Fixed.(string); ok && u != "" {
			return u
		}
	}
	return ""
}

// childJSONKey returns the JSON object key for a child element, mapping a choice
// element (path "value[x]") to its concrete suffixed key (e.g. "valueCodeableConcept")
// when a single concrete type is fixed by a Fixed/Pattern value.
func childJSONKey(child *fhir.ElementDefinition) string {
	key := lastSegment(child.Path)
	if !strings.HasSuffix(key, "[x]") {
		return key
	}
	base := strings.TrimSuffix(key, "[x]")
	// Determine the concrete type: from the Fixed/Pattern value when present,
	// otherwise from the first type.
	var ty string
	switch {
	case child.Fixed != nil:
		ty = fhirTypeOfValue(child.Fixed)
	case child.Pattern != nil:
		ty = fhirTypeOfValue(child.Pattern)
	}
	if ty != "" {
		return base + capitalize(ty)
	}
	if len(child.Types) > 0 {
		return base + capitalize(child.Types[0].Code)
	}
	return base
}

// fhirTypeOfValue returns the FHIR complex type code implied by a Fixed/Pattern
// value's JSON shape (e.g. a map with "coding" → "CodeableConcept"; a map with
// "system" → "Coding").
func fhirTypeOfValue(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	if _, has := m["coding"]; has {
		return "CodeableConcept"
	}
	if _, has := m["system"]; has {
		return "Coding"
	}
	if _, has := m["period"]; has {
		return "Period"
	}
	if _, has := m["reference"]; has {
		return "Reference"
	}
	return ""
}

// mergeSlicePattern deep-merges a nested pattern map into the value at the given
// key, preserving any sibling keys already present.
func mergeSlicePattern(value map[string]any, key string, sub map[string]any) {
	existing, ok := value[key].(map[string]any)
	if !ok {
		existing = make(map[string]any)
		value[key] = existing
	}
	for k, v := range sub {
		if nested, ok := v.(map[string]any); ok {
			mergeSlicePattern(existing, k, nested)
		} else {
			existing[k] = v
		}
	}
}

// shouldFill reports whether an element should be generated.
func (g *Generator) shouldFill(elem *fhir.ElementDefinition) bool {
	if elem.Min > 0 {
		return true
	}
	// An optional element that carries a contract signal (fixed/pattern value,
	// examples, a value set binding, or type profiles) is generated even in
	// minimal mode, so contract-driven elements are always present.
	if hasContractSignal(elem) {
		return true
	}
	switch g.fill {
	case fillFull:
		return true
	case fillProbability:
		return g.randFloat() < g.fillProb
	default:
		return false
	}
}

// hasContractSignal reports whether an element carries a contract signal that
// should force its generation even when optional: a fixed or pattern value,
// example values, a value set binding, or type profiles.
func hasContractSignal(elem *fhir.ElementDefinition) bool {
	if elem == nil {
		return false
	}
	if elem.Fixed != nil || elem.Pattern != nil || len(elem.Examples) > 0 || elem.Binding != nil {
		return true
	}
	for _, et := range elem.Types {
		if len(et.Profiles) > 0 {
			return true
		}
	}
	return false
}

// fillChoice fills a choice ([x]) element by picking one of its allowed types.
func (g *Generator) fillChoice(elem *fhir.ElementDefinition, tree *fhir.ElementTree, out map[string]any, depth int) error {
	if len(elem.Types) == 0 {
		return nil
	}
	// If the caller supplied a value for one of the concrete suffixed keys,
	// prefer that type and let applyValues inject the value, so we do not emit
	// a conflicting concrete value for the same choice element.
	if key, ok := g.choiceValueKey(elem, tree); ok {
		_ = key
		return nil
	}
	ty := elem.Types[g.randN(len(elem.Types))]
	key := fhir.ChoiceName(elem, ty.Code)
	val, err := g.fillTypedValue(elem, ty.Code, tree, depth)
	if err != nil {
		return err
	}
	if val != nil {
		out[key] = val
	}
	return nil
}

// choiceValueKey returns the concrete suffixed key of a choice element for
// which the caller supplied a value, if any. Keys are relative to the resource
// root (e.g. "deceasedBoolean" for Patient.deceased[x]).
func (g *Generator) choiceValueKey(elem *fhir.ElementDefinition, tree *fhir.ElementTree) (string, bool) {
	if len(g.values) == 0 {
		return "", false
	}
	rel := strings.TrimPrefix(elem.Path, tree.Root.Path+".")
	base := strings.TrimSuffix(rel, "[x]")
	for _, ty := range elem.Types {
		key := base + capitalize(ty.Code)
		if _, ok := g.values[key]; ok {
			return key, true
		}
	}
	return "", false
}

// fillValue produces a value for an element, wrapping repeating elements in
// arrays.
func (g *Generator) fillValue(elem *fhir.ElementDefinition, tree *fhir.ElementTree, depth int) (any, error) {
	if fhir.IsMulti(elem) {
		n := 1 + g.randN(3)
		arr := make([]any, 0, n)
		for range n {
			v, err := g.fillSingle(elem, tree, depth)
			if err != nil {
				return nil, err
			}
			if v != nil {
				arr = append(arr, v)
			}
		}
		if len(arr) == 0 {
			return nil, nil
		}
		return arr, nil
	}
	return g.fillSingle(elem, tree, depth)
}

// fillSingle produces a single (non-array) value for an element.
func (g *Generator) fillSingle(elem *fhir.ElementDefinition, tree *fhir.ElementTree, depth int) (any, error) {
	code := fhir.PrimaryTypeCode(elem)
	return g.fillTypedValue(elem, code, tree, depth)
}

// fillTypedValue produces a value for an element of the given type code.
func (g *Generator) fillTypedValue(elem *fhir.ElementDefinition, typeCode string, tree *fhir.ElementTree, depth int) (any, error) {
	// FHIRPath system types (e.g. http://hl7.org/fhirpath/System.String) map
	// to their primitive equivalent by suffix.
	if after, ok := strings.CutPrefix(typeCode, "http://hl7.org/fhirpath/System."); ok {
		typeCode = strings.ToLower(after)
	}
	switch typeCode {
	case "":
		return nil, nil
	case "string", "markdown", "id", "code", "oid", "uri", "url", "canonical", "uuid", "base64Binary":
		return g.fakeStringFor(elem, tree), nil
	case "boolean":
		return g.fakeBool(), nil
	case "integer", "positiveInt", "unsignedInt":
		return g.fakeInt(), nil
	case "decimal":
		return g.fakeDecimal(), nil
	case "date":
		return g.fakeDate(), nil
	case "dateTime", "instant":
		return g.fakeDateTime(), nil
	case "time":
		return g.fakeTime(), nil
	case "Reference":
		return g.fakeReference(elem), nil
	case "Resource":
		// "Resource" is an abstract base type that cannot be instantiated
		// in FHIR. Contained resources with resourceType "Resource" are
		// rejected by servers (HAPI-1684). Skip rather than emit an
		// invalid payload.
		return nil, nil
	case "HumanName":
		return g.fakeComplex(elem, tree, depth, g.fakeHumanName)
	case "Address":
		return g.fakeComplex(elem, tree, depth, g.fakeAddress)
	case "Identifier":
		return g.fakeComplex(elem, tree, depth, g.fakeIdentifier)
	case "ContactPoint":
		return g.fakeComplex(elem, tree, depth, g.fakeContactPoint)
	case "Coding":
		return g.fakeCoding(elem, tree), nil
	case "CodeableConcept":
		return g.fakeCodeableConcept(elem, tree), nil
	case "Extension":
		return g.fakeComplex(elem, tree, depth, func() map[string]any {
			return g.fakeExtension(elem, tree)
		})
	case "Narrative":
		return g.fakeNarrative(), nil
	case "Meta":
		return g.fakeMeta(), nil
	case "Quantity":
		return g.fakeQuantity(), nil
	case "Period":
		return g.fakePeriod(), nil
	case "Range":
		return g.fakeRange(), nil
	case "Ratio":
		return g.fakeRatio(), nil
	case "Attachment":
		return g.fakeAttachment(), nil
	case "Annotation":
		return g.fakeAnnotation(), nil
	case "Timing":
		return g.fakeTiming(), nil
	case "Signature":
		return g.fakeSignature(), nil
	case "Dosage":
		return g.fakeDosage(), nil
	case "BackboneElement", "Element":
		return g.fillObjectDepth(elem, tree, depth+1)
	default:
		// Complex type: resolve through the registry and recurse.
		if t, ok := g.reg.ResolveType(typeCode, profilesOf(elem)); ok {
			return g.fillObjectDepth(t.Root, t, depth+1)
		}
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedType, typeCode)
	}
}

// fakeComplex fills a complex type, preferring resolution through the registry
// (which carries profile fixed/pattern values) when the element has profile
// hints, and falling back to the given hardcoded fake otherwise.
func (g *Generator) fakeComplex(elem *fhir.ElementDefinition, tree *fhir.ElementTree, depth int, fallback func() map[string]any) (any, error) {
	if len(profilesOf(elem)) > 0 {
		if t, ok := g.reg.ResolveType(fhir.PrimaryTypeCode(elem), profilesOf(elem)); ok {
			return g.fillObjectDepth(t.Root, t, depth+1)
		}
	}
	// An element without a profile hint may still carry its own resolved
	// children (e.g. a sliced Extension whose sub-extension slices define url
	// and value[x] directly, like the HCPD deactivatedBy/suppressedBy slices).
	// Filling from those children produces a conformant structure; falling back
	// to the hardcoded fake here would emit a generic placeholder extension.
	if len(elem.Children) > 0 {
		if t, ok := g.reg.ResolveType(fhir.PrimaryTypeCode(elem), nil); ok {
			return g.fillObjectDepth(elem, t, depth+1)
		}
		return g.fillObjectDepth(elem, tree, depth+1)
	}
	return fallback(), nil
}

// stripEmptyExtensions recursively removes extension and modifierExtension
// entries that have neither a value[x] nor a nested extension array. Such
// extensions violate the FHIR ext-1 invariant. It mutates the given object in
// place.
func stripEmptyExtensions(obj map[string]any) {
	for _, key := range []string{"extension", "modifierExtension"} {
		raw, ok := obj[key]
		if !ok {
			continue
		}
		arr, ok := raw.([]any)
		if !ok {
			continue
		}
		kept := arr[:0]
		for _, item := range arr {
			ext, ok := item.(map[string]any)
			if !ok {
				kept = append(kept, item)
				continue
			}
			if extensionHasValue(ext) {
				kept = append(kept, item)
			}
		}
		if len(kept) == 0 {
			delete(obj, key)
		} else {
			obj[key] = kept
		}
	}
	// Recurse into remaining children.
	for _, v := range obj {
		switch val := v.(type) {
		case map[string]any:
			stripEmptyExtensions(val)
		case []any:
			for _, item := range val {
				if m, ok := item.(map[string]any); ok {
					stripEmptyExtensions(m)
				}
			}
		}
	}
}

// extensionHasValue reports whether an extension map carries a value[x] key or
// a non-empty nested extension array.
func extensionHasValue(ext map[string]any) bool {
	for k := range ext {
		if k == "url" || k == "id" {
			continue
		}
		if k == "extension" || k == "modifierExtension" {
			if arr, ok := ext[k].([]any); ok && len(arr) > 0 {
				return true
			}
			continue
		}
		// Any other key is a value[x] (e.g. valueString, valueCoding).
		return true
	}
	return false
}

// enforceExt1 removes value[x] keys from extensions that carry non-empty
// sub-extensions. FHIR's ext-1 invariant requires that an Extension has
// either a value (value[x]) or contained extensions, but never both.
func enforceExt1(obj map[string]any) {
	for _, key := range []string{"extension", "modifierExtension"} {
		raw, ok := obj[key]
		if !ok {
			continue
		}
		arr, ok := raw.([]any)
		if !ok {
			continue
		}
		for _, item := range arr {
			ext, ok := item.(map[string]any)
			if !ok {
				continue
			}
			// If this extension has non-empty sub-extensions, drop any value[x].
			if hasNonEmptyExtensions(ext) {
				removeValueKeys(ext)
			}
			enforceExt1(ext)
		}
	}
	// Recurse into remaining children.
	for _, v := range obj {
		switch val := v.(type) {
		case map[string]any:
			enforceExt1(val)
		case []any:
			for _, item := range val {
				if m, ok := item.(map[string]any); ok {
					enforceExt1(m)
				}
			}
		}
	}
}

// hasNonEmptyExtensions returns true if the map has a non-empty "extension"
// or "modifierExtension" array.
func hasNonEmptyExtensions(m map[string]any) bool {
	for _, key := range []string{"extension", "modifierExtension"} {
		if arr, ok := m[key].([]any); ok && len(arr) > 0 {
			return true
		}
	}
	return false
}

// removeValueKeys deletes any key that is a value[x] (i.e. not "url", "id",
// "extension", or "modifierExtension") from the map.
func removeValueKeys(m map[string]any) {
	for k := range m {
		if k == "url" || k == "id" || k == "extension" || k == "modifierExtension" {
			continue
		}
		delete(m, k)
	}
}
