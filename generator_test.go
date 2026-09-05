package fhirgen

import (
	"encoding/json"
	"testing"

	fhir "github.com/jlcoulter/fhir-registry"
)

func loadTestRegistry(t *testing.T) *fhir.Registry {
	t.Helper()
	reg := fhir.NewRegistry()
	if err := reg.LoadPackageTgz("testdata/au-base.tgz"); err != nil {
		t.Fatalf("LoadPackageTgz: %v", err)
	}
	return reg
}

// TestNew verifies a generator can be constructed from a registry.
func TestNew(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg)
	if g == nil {
		t.Fatal("New returned nil generator")
	}
}

// TestGeneratePatient verifies that generating a Patient produces a
// conformant resource: it has a resourceType, and passes the registry's own
// Marshal normalizer without cardinality violations.
func TestGeneratePatient(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if out["resourceType"] != "Patient" {
		t.Errorf("resourceType = %#v, want Patient", out["resourceType"])
	}

	// The generated resource must be conformant: Marshal should report no
	// cardinality violations.
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

// TestGenerateDeterministic verifies that the same seed produces identical
// output across two calls.
func TestGenerateDeterministic(t *testing.T) {
	reg := loadTestRegistry(t)
	g1 := New(reg, WithSeed(7))
	g2 := New(reg, WithSeed(7))

	a, err := g1.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	b, err := g2.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	if string(aj) != string(bj) {
		t.Errorf("same seed produced different output:\n%s\n%s", aj, bj)
	}
}

// TestGenerateForURL verifies generation for a specific StructureDefinition
// canonical URL.
func TestGenerateForURL(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(1))

	out, err := g.GenerateForURL("http://hl7.org.au/fhir/StructureDefinition/au-patient")
	if err != nil {
		t.Fatalf("GenerateForURL: %v", err)
	}
	if out["resourceType"] != "Patient" {
		t.Errorf("resourceType = %#v, want Patient", out["resourceType"])
	}
}

// TestGenerateUnknownType verifies an error for an unknown type name.
func TestGenerateUnknownType(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg)
	if _, err := g.Generate("NoSuchType"); err == nil {
		t.Fatal("expected error for unknown type")
	}
}

// TestGenerateJSONMarshalable verifies the output can be marshaled to JSON.
func TestGenerateJSONMarshalable(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(3))
	out, err := g.Generate("Organization")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, err := json.Marshal(out); err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
}

// TestWithID verifies that WithID injects the given id into the generated
// resource.
func TestWithID(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithID("momus-setup-patient-1"))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if out["id"] != "momus-setup-patient-1" {
		t.Errorf("id = %#v, want %q", out["id"], "momus-setup-patient-1")
	}
}

// TestWithoutID verifies that a generator without WithID does not inject an id.
func TestWithoutID(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, ok := out["id"]; ok {
		t.Errorf("id unexpectedly present: %#v", out["id"])
	}
}

// TestWithMetaProfiles verifies that WithMetaProfiles injects a meta.profile
// array into the generated resource.
func TestWithMetaProfiles(t *testing.T) {
	reg := loadTestRegistry(t)
	profiles := []string{
		"http://hl7.org.au/fhir/StructureDefinition/au-patient",
		"http://example.org/StructureDefinition/extra",
	}
	g := New(reg, WithSeed(42), WithMetaProfiles(profiles))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	meta, ok := out["meta"].(map[string]any)
	if !ok {
		t.Fatalf("meta = %#v, want map[string]any", out["meta"])
	}
	got, ok := meta["profile"].([]any)
	if !ok {
		t.Fatalf("meta.profile = %#v, want []any", meta["profile"])
	}
	if len(got) != len(profiles) {
		t.Fatalf("meta.profile length = %d, want %d", len(got), len(profiles))
	}
	for i, want := range profiles {
		if got[i] != want {
			t.Errorf("meta.profile[%d] = %#v, want %q", i, got[i], want)
		}
	}
}

// TestWithoutMetaProfiles verifies that a generator without WithMetaProfiles
// does not inject a meta.profile.
func TestWithoutMetaProfiles(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	meta, ok := out["meta"].(map[string]any)
	if !ok {
		return // no meta at all is fine
	}
	if _, ok := meta["profile"]; ok {
		t.Errorf("meta.profile unexpectedly present: %#v", meta["profile"])
	}
}
