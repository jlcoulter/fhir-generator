package fhirgen

import (
	"testing"
)

// TestFullModeHandlesFHIRPathSystemTypes verifies that full fill mode does not
// error when an element is typed as a FHIRPath System.* type (e.g.
// http://hl7.org/fhirpath/System.String), which appears in some snapshots.
func TestFullModeHandlesFHIRPathSystemTypes(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())
	if _, err := g.Generate("Patient"); err != nil {
		t.Fatalf("Generate in full mode: %v", err)
	}
}

// TestFullModeDoesNotRecurseForever verifies that recursive types such as
// Extension (Extension.extension is itself an Extension) do not cause infinite
// recursion in full fill mode, and that fake-data generation does not deadlock.
func TestFullModeDoesNotRecurseForever(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())
	for _, typ := range []string{"Patient", "Organization", "Practitioner"} {
		if _, err := g.Generate(typ); err != nil {
			t.Fatalf("Generate(%s) in full mode: %v", typ, err)
		}
	}
}

// TestFillChoiceProducesSuffixedKey verifies that a choice element produces a
// key with the concrete type suffix (e.g. deceasedBoolean, deceasedDateTime).
func TestFillChoiceProducesSuffixedKey(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(1))
	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	found := false
	for _, key := range []string{"deceasedBoolean", "deceasedDateTime", "multipleBirthBoolean", "multipleBirthInteger"} {
		if _, ok := out[key]; ok {
			found = true
		}
	}
	if !found {
		t.Errorf("no choice key produced, got %v", out)
	}
}
