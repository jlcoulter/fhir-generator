package fhirgen

import (
	"testing"

	fhir "github.com/jlcoulter/fhir-registry"
)

// TestWithValuesOverridesFakeData verifies that a caller-supplied value for a
// top-level element path overrides the fake-generated value.
func TestWithValuesOverridesFakeData(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithValues(map[string]any{"birthDate": "1990-01-01"}))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if out["birthDate"] != "1990-01-01" {
		t.Errorf("birthDate = %#v, want 1990-01-01", out["birthDate"])
	}
}

// TestWithValuesNestedPath verifies that a caller-supplied value for a nested
// element path (e.g. name.family) overrides that field while other fields of
// the same object are still generated.
func TestWithValuesNestedPath(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithValues(map[string]any{"name.family": "Smith"}))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	names, ok := out["name"].([]any)
	if !ok || len(names) == 0 {
		t.Fatalf("name = %#v, want non-empty array", out["name"])
	}
	name, ok := names[0].(map[string]any)
	if !ok {
		t.Fatalf("name[0] = %#v, want map", names[0])
	}
	if name["family"] != "Smith" {
		t.Errorf("name.family = %#v, want Smith", name["family"])
	}
	// Other fields of the name object should still be generated.
	if _, ok := name["given"]; !ok {
		t.Errorf("name.given should still be generated, got %#v", name)
	}
}

// TestWithValuesWholeObject verifies that a caller-supplied map for an object
// path replaces the entire generated object.
func TestWithValuesWholeObject(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithValues(map[string]any{
		"name": map[string]any{"family": "Smith", "given": []any{"John"}},
	}))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	names, ok := out["name"].([]any)
	if !ok || len(names) == 0 {
		t.Fatalf("name = %#v, want non-empty array", out["name"])
	}
	name, ok := names[0].(map[string]any)
	if !ok {
		t.Fatalf("name[0] = %#v, want map", names[0])
	}
	if name["family"] != "Smith" {
		t.Errorf("name.family = %#v, want Smith", name["family"])
	}
	if name["given"].([]any)[0] != "John" {
		t.Errorf("name.given = %#v, want [John]", name["given"])
	}
}

// TestWithValuesArrayElement verifies that a caller-supplied value for a field
// within a repeating element is applied to the generated instances.
func TestWithValuesArrayElement(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithValues(map[string]any{"identifier.value": "specific-id"}))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	ids, ok := out["identifier"].([]any)
	if !ok || len(ids) == 0 {
		t.Fatalf("identifier = %#v, want non-empty array", out["identifier"])
	}
	ident, ok := ids[0].(map[string]any)
	if !ok {
		t.Fatalf("identifier[0] = %#v, want map", ids[0])
	}
	if ident["value"] != "specific-id" {
		t.Errorf("identifier.value = %#v, want specific-id", ident["value"])
	}
}

// TestWithValuesChoiceElement verifies that a caller-supplied value for a
// choice element's concrete suffixed key is honored.
func TestWithValuesChoiceElement(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithValues(map[string]any{"deceasedBoolean": true}))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if out["deceasedBoolean"] != true {
		t.Errorf("deceasedBoolean = %#v, want true", out["deceasedBoolean"])
	}
}

// TestWithValuesDoesNotAffectUnsetFields verifies that fields not present in
// the values map are still generated with fake data.
func TestWithValuesDoesNotAffectUnsetFields(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode(), WithValues(map[string]any{"birthDate": "1990-01-01"}))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, ok := out["gender"]; !ok {
		t.Errorf("gender should still be generated, got %#v", out)
	}
}

// TestWithValuesConformance verifies that a resource generated with caller
// values still passes the registry's conformance checks.
func TestWithValuesConformance(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithValues(map[string]any{
		"birthDate":       "1990-01-01",
		"name.family":     "Smith",
		"gender":          "male",
		"deceasedBoolean": true,
	}))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	_, rep, err := reg.Marshal("Patient", out)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, it := range rep.Items {
		if it.Severity == fhir.SeverityViolation {
			t.Errorf("conformance violation: %+v", it)
		}
	}
}
