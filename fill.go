package fhirgen

import (
	"fmt"
	"slices"
	"strings"

	fhir "github.com/jlcoulter/fhir-registry"
)

// Generate produces a conformant FHIR resource instance for the given base
// type name (e.g. "Patient", "Organization"). The result is a map suitable
// for json.Marshal.
func (g *Generator) Generate(typeName string) (map[string]any, error) {
	tree, err := g.reg.TreeForType(typeName)
	if err != nil {
		return nil, fmt.Errorf("%w: type %s", ErrDefinitionNotFound, typeName)
	}
	return g.generateFromTree(tree)
}

// GenerateForURL produces a conformant FHIR resource instance for a specific
// StructureDefinition canonical URL.
func (g *Generator) GenerateForURL(url string) (map[string]any, error) {
	tree, err := g.reg.Tree(url)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDefinitionNotFound, url)
	}
	return g.generateFromTree(tree)
}

func (g *Generator) generateFromTree(tree *fhir.ElementTree) (map[string]any, error) {
	obj, err := g.fillObject(tree.Root, tree)
	if err != nil {
		return nil, err
	}
	obj["resourceType"] = tree.Root.Path
	g.applyValues(obj, tree)
	return obj, nil
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

	// Fixed or pattern values are emitted verbatim.
	if child.Fixed != nil {
		out[key] = child.Fixed
		return nil
	}
	if child.Pattern != nil {
		out[key] = child.Pattern
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
			arr = append(arr, v)
		}
	}
	if len(arr) > 0 {
		out[key] = arr
	}
	return nil
}

// shouldFill reports whether an element should be generated.
func (g *Generator) shouldFill(elem *fhir.ElementDefinition) bool {
	if elem.Min > 0 {
		return true
	}
	return g.fill == fillFull
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
		return g.fakeStringFor(elem), nil
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
		return g.fakeCoding(elem), nil
	case "CodeableConcept":
		return g.fakeCodeableConcept(elem), nil
	case "Extension":
		return g.fakeComplex(elem, tree, depth, func() map[string]any {
			return g.fakeExtension(elem)
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
