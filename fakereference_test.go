package fhirgen

import (
	"strings"
	"testing"

	fhir "github.com/jlcoulter/fhir-registry"
)

// TestFakeReferenceResolvesBaseResourceType verifies that a generated Reference
// uses the target profile's base FHIR resource type (e.g. "Practitioner"),
// not the profile's canonical-URL last segment (e.g. "special-practitioner").
// A reference typed with a profile id is rejected by servers (HAPI-0931), so
// the generator must resolve the base type through the registry, even when the
// targetProfile carries a "|version" suffix.
func TestFakeReferenceResolvesBaseResourceType(t *testing.T) {
	reg := fhir.NewRegistry()
	reg.AddStructureDefinition(fhir.NewStructureDefinition(
		"http://example.org/fhir/StructureDefinition/special-practitioner",
		"SpecialPractitioner", "Practitioner", "resource",
		"http://hl7.org/fhir/StructureDefinition/Practitioner", "constraint", nil))

	g := New(reg, WithSeed(1))
	elem := &fhir.ElementDefinition{
		Path: "Thing.author",
		Types: []fhir.ElementType{{
			Code:          "Reference",
			TargetProfile: []string{"http://example.org/fhir/StructureDefinition/special-practitioner|26.0.0"},
		}},
	}

	ref := g.fakeReference(elem)
	got, _ := ref["reference"].(string)
	if !strings.HasPrefix(got, "Practitioner/") {
		t.Fatalf("fakeReference reference = %q, want prefix %q", got, "Practitioner/")
	}
}

// TestFakeReferenceFallsBackToURLSegment verifies that when the target profile
// is not indexed in the registry, the generator still produces a reference by
// falling back to the URL's last path segment (version stripped).
func TestFakeReferenceFallsBackToURLSegment(t *testing.T) {
	reg := fhir.NewRegistry()
	g := New(reg, WithSeed(1))
	elem := &fhir.ElementDefinition{
		Path: "Thing.subject",
		Types: []fhir.ElementType{{
			Code:          "Reference",
			TargetProfile: []string{"http://hl7.org/fhir/StructureDefinition/Patient|4.0.1"},
		}},
	}

	ref := g.fakeReference(elem)
	got, _ := ref["reference"].(string)
	if !strings.HasPrefix(got, "Patient/") {
		t.Fatalf("fakeReference reference = %q, want prefix %q", got, "Patient/")
	}
}
