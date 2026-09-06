package fhirgen

import (
	"testing"

	fhir "github.com/jlcoulter/fhir-registry"
)

// TestFullModeIncludesOptionalElements verifies that full fill mode generates
// optional elements that minimal mode omits. Patient.name is 0..1 (optional).
func TestFullModeIncludesOptionalElements(t *testing.T) {
	reg := loadTestRegistry(t)

	minG := New(reg, WithSeed(42))
	minOut, err := minG.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate(minimal): %v", err)
	}
	if _, ok := minOut["name"]; ok {
		t.Error("minimal mode should not include optional Patient.name")
	}

	fullG := New(reg, WithSeed(42), WithFullFillMode())
	fullOut, err := fullG.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate(full): %v", err)
	}
	if _, ok := fullOut["name"]; ok {
		t.Logf("full mode included optional name: %#v", fullOut["name"])
	} else {
		t.Error("full mode should include optional Patient.name (min 0..1 present in full mode)")
	}
}

// TestProbabilityFillModeAlways verifies that WithProbabilityFillMode(1.0)
// includes optional elements (like full fill mode).
func TestProbabilityFillModeAlways(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithProbabilityFillMode(1.0))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// Patient.name is optional (min 0) and has no contract signal. With
	// probability 1.0 it must be present.
	if _, ok := out["name"]; !ok {
		t.Errorf("name unexpectedly absent with probability 1.0")
	}
}

// TestProbabilityFillModeNever verifies that WithProbabilityFillMode(0.0)
// excludes optional elements (like minimal fill mode).
func TestProbabilityFillModeNever(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithProbabilityFillMode(0.0))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// Patient.name is optional (min 0) and has no contract signal. With
	// probability 0.0 it must be absent.
	if _, ok := out["name"]; ok {
		t.Errorf("name unexpectedly present with probability 0.0")
	}
}

// stubBindingResolver is a BindingResolver that returns a fixed coding for any
// element.
type stubBindingResolver struct {
	coding ResolvedCoding
}

func (s stubBindingResolver) ResolveBinding(elem *fhir.ElementDefinition, tree *fhir.ElementTree) (ResolvedCoding, bool) {
	return s.coding, true
}

// TestBindingResolver verifies that WithBindingResolver is consulted for coded
// elements and its result is used in place of the built-in heuristics.
func TestBindingResolver(t *testing.T) {
	reg := loadTestRegistry(t)
	resolver := stubBindingResolver{coding: ResolvedCoding{
		System:  "http://example.org/custom-system",
		Code:    "CUSTOM",
		Display: "Custom Code",
	}}
	g := New(reg, WithSeed(42), WithFullFillMode(), WithBindingResolver(resolver))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// Patient.gender is a scalar code bound to administrative-gender. The
	// resolver must override it.
	gender, ok := out["gender"].(string)
	if !ok {
		t.Fatalf("gender = %#v, want string", out["gender"])
	}
	if gender != "CUSTOM" {
		t.Errorf("gender = %q, want %q from resolver", gender, "CUSTOM")
	}
}

// TestBindingResolverCoding verifies the resolver result flows into a Coding
// object (system, code, display).
func TestBindingResolverCoding(t *testing.T) {
	reg := loadTestRegistry(t)
	resolver := stubBindingResolver{coding: ResolvedCoding{
		System:  "http://example.org/custom-system",
		Code:    "CUSTOM",
		Display: "Custom Code",
	}}
	g := New(reg, WithSeed(42), WithFullFillMode(), WithBindingResolver(resolver))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// Patient.maritalStatus is a CodeableConcept bound to marital-status.
	ms, ok := out["maritalStatus"].(map[string]any)
	if !ok {
		t.Skip("no maritalStatus generated")
	}
	codings, ok := ms["coding"].([]any)
	if !ok || len(codings) == 0 {
		t.Fatalf("maritalStatus.coding = %#v, want non-empty", ms["coding"])
	}
	first, ok := codings[0].(map[string]any)
	if !ok {
		t.Fatalf("coding[0] = %#v, want map", codings[0])
	}
	if first["system"] != "http://example.org/custom-system" {
		t.Errorf("coding.system = %#v, want resolver system", first["system"])
	}
	if first["code"] != "CUSTOM" {
		t.Errorf("coding.code = %#v, want resolver code", first["code"])
	}
}

// stubDisplayResolver is a CodingDisplayResolver that returns a fixed display
// for any (system, code) pair.
type stubDisplayResolver struct {
	display string
}

func (s stubDisplayResolver) ResolveDisplay(system, code string) (string, bool) {
	return s.display, true
}

// TestCodingDisplayResolver verifies that WithCodingDisplayResolver populates
// the display field of generated codings.
func TestCodingDisplayResolver(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode(), WithCodingDisplayResolver(stubDisplayResolver{display: "Resolved Display"}))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// Patient.maritalStatus is a CodeableConcept with a coding.
	ms, ok := out["maritalStatus"].(map[string]any)
	if !ok {
		t.Skip("no maritalStatus generated")
	}
	codings, ok := ms["coding"].([]any)
	if !ok || len(codings) == 0 {
		t.Fatalf("maritalStatus.coding = %#v, want non-empty", ms["coding"])
	}
	first, ok := codings[0].(map[string]any)
	if !ok {
		t.Fatalf("coding[0] = %#v, want map", codings[0])
	}
	if first["display"] != "Resolved Display" {
		t.Errorf("coding.display = %#v, want %q", first["display"], "Resolved Display")
	}
}

// TestNormaliser verifies that WithNormaliser is applied to the generated
// resource after filling, mutating it in place.
func TestNormaliser(t *testing.T) {
	reg := loadTestRegistry(t)
	called := false
	g := New(reg, WithSeed(42), WithNormaliser(func(body map[string]any) {
		called = true
		body["normalized"] = true
	}))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !called {
		t.Fatal("normaliser was not called")
	}
	if out["normalized"] != true {
		t.Errorf("normalized = %#v, want true", out["normalized"])
	}
}

// TestStripEmptyExtensions verifies that WithStripEmptyExtensions removes
// extension entries that have neither a value[x] nor a nested extension array.
func TestStripEmptyExtensions(t *testing.T) {
	reg := loadTestRegistry(t)
	// Inject an empty extension via the normaliser, then verify it is stripped.
	g := New(reg, WithSeed(42), WithStripEmptyExtensions(), WithNormaliser(func(body map[string]any) {
		body["extension"] = []any{
			map[string]any{"url": "http://example.org/empty"},                  // no value, no sub-extensions
			map[string]any{"url": "http://example.org/ok", "valueString": "x"}, // has value
		}
	}))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	exts, ok := out["extension"].([]any)
	if !ok {
		t.Fatalf("extension = %#v, want []any", out["extension"])
	}
	if len(exts) != 1 {
		t.Fatalf("extension length = %d, want 1 (empty stripped)", len(exts))
	}
	first, ok := exts[0].(map[string]any)
	if !ok {
		t.Fatalf("extension[0] = %#v, want map", exts[0])
	}
	if first["url"] != "http://example.org/ok" {
		t.Errorf("extension[0].url = %#v, want the non-empty extension", first["url"])
	}
}

// TestWithoutStripEmptyExtensions verifies that empty extensions are kept when
// WithStripEmptyExtensions is not set.
func TestWithoutStripEmptyExtensions(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithNormaliser(func(body map[string]any) {
		body["extension"] = []any{
			map[string]any{"url": "http://example.org/empty"},
		}
	}))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	exts, ok := out["extension"].([]any)
	if !ok {
		t.Fatalf("extension = %#v, want []any", out["extension"])
	}
	if len(exts) != 1 {
		t.Fatalf("extension length = %d, want 1 (kept)", len(exts))
	}
}

// TestContractSignalOptionalFilled verifies that an optional (min == 0) element
// carrying a contract signal (Fixed, Pattern, Examples, Binding, or type
// profiles) is generated even in minimal fill mode. This matches the behavior
// of consumers that need contract-driven elements present in non-exhaustive
// output.
func TestContractSignalOptionalFilled(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42)) // minimal mode

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// Patient.gender is optional (min 0) but bound to administrative-gender.
	// It carries a contract signal (binding) and must be present in minimal mode.
	if _, ok := out["gender"]; !ok {
		t.Errorf("gender (optional, bound) unexpectedly absent in minimal mode")
	}
}

// TestFillFixedValueEmitted verifies that an element with a Fixed value emits
// that value verbatim.
func TestFillFixedValueEmitted(t *testing.T) {
	reg := loadTestRegistry(t)
	elem := &fhir.ElementDefinition{ID: "R", Path: "R.resourceType", Min: 1, Max: 1, Fixed: "Patient"}
	out := map[string]any{}
	g := New(reg)
	tree := &fhir.ElementTree{Root: elem, ByPath: map[string][]*fhir.ElementDefinition{}, ByID: map[string]*fhir.ElementDefinition{}}
	if err := g.fillChild(elem, tree, out, 0); err != nil {
		t.Fatalf("fillChild: %v", err)
	}
	if out["resourceType"] != "Patient" {
		t.Errorf("fixed value not emitted: %#v", out)
	}
}

// TestErrorSentinels verifies exported sentinel errors are distinct.
func TestErrorSentinels(t *testing.T) {
	if ErrDefinitionNotFound == nil {
		t.Error("ErrDefinitionNotFound is nil")
	}
	if ErrUnsupportedType == nil {
		t.Error("ErrUnsupportedType is nil")
	}
	if ErrInvalidPath == nil {
		t.Error("ErrInvalidPath is nil")
	}
	if ErrDefinitionNotFound == ErrUnsupportedType {
		t.Error("sentinels should be distinct")
	}
	if ErrDefinitionNotFound == ErrInvalidPath {
		t.Error("sentinels should be distinct")
	}
	if ErrUnsupportedType == ErrInvalidPath {
		t.Error("sentinels should be distinct")
	}
}
