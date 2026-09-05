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
	if g.normalizer != nil {
		g.normalizer(obj)
	}
	if g.stripEmptyExtensions {
		stripEmptyExtensions(obj)
	}
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
	// data, which callers may mutate (e.g. normalizers that strip display/text).
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
		return map[string]any{"resourceType": "Resource", "id": g.fakeID()}, nil
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
