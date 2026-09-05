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
	if ErrDefinitionNotFound == ErrUnsupportedType {
		t.Error("sentinels should be distinct")
	}
}
