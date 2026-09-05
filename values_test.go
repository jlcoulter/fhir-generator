package fhirgen

import (
	"errors"
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
// element path (e.g. address.city) overrides that field while other fields of
// the same object are still generated.
func TestWithValuesNestedPath(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithValues(map[string]any{"address.city": "Sydney"}))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	addrs, ok := out["address"].([]any)
	if !ok || len(addrs) == 0 {
		t.Fatalf("address = %#v, want non-empty array", out["address"])
	}
	addr, ok := addrs[0].(map[string]any)
	if !ok {
		t.Fatalf("address[0] = %#v, want map", addrs[0])
	}
	if addr["city"] != "Sydney" {
		t.Errorf("address.city = %#v, want Sydney", addr["city"])
	}
	// Other fields of the address object should still be generated (country is
	// fixed "AU" and always present).
	if addr["country"] != "AU" {
		t.Errorf("address.country should still be generated, got %#v", addr)
	}
}

// TestWithValuesWholeObject verifies that a caller-supplied map for an object
// path replaces the entire generated object.
func TestWithValuesWholeObject(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithValues(map[string]any{
		"address": map[string]any{"city": "Sydney", "state": "NSW"},
	}))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	addrs, ok := out["address"].([]any)
	if !ok || len(addrs) == 0 {
		t.Fatalf("address = %#v, want non-empty array", out["address"])
	}
	addr, ok := addrs[0].(map[string]any)
	if !ok {
		t.Fatalf("address[0] = %#v, want map", addrs[0])
	}
	if addr["city"] != "Sydney" {
		t.Errorf("address.city = %#v, want Sydney", addr["city"])
	}
	if addr["state"] != "NSW" {
		t.Errorf("address.state = %#v, want NSW", addr["state"])
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
		"address.city":    "Sydney",
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

// TestWithValuesInvalidPath verifies that a WithValues path that does not
// resolve against the registry returns an error rather than being silently
// ignored.
func TestWithValuesInvalidPath(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithValues(map[string]any{"nonexistent": "x"}))

	_, err := g.Generate("Patient")
	if err == nil {
		t.Fatal("expected error for invalid WithValues path")
	}
	if !errors.Is(err, ErrInvalidPath) {
		t.Errorf("err = %v, want ErrInvalidPath", err)
	}
}
