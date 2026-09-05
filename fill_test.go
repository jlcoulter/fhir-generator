package fhirgen

import (
	"testing"

	fhir "github.com/jlcoulter/fhir-registry"
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

// TestComplexExtensionsHaveNoValue verifies that a complex extension (one that
// carries a nested extension array) does not also carry a value[x]. A complex
// extension's value[x] is Max 0, so emitting both is structurally invalid.
func TestComplexExtensionsHaveNoValue(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())

	for _, typ := range []string{"Patient", "Organization", "Practitioner", "Observation", "PractitionerRole", "HealthcareService"} {
		out, err := g.Generate(typ)
		if err != nil {
			t.Fatalf("Generate(%s): %v", typ, err)
		}
		var walk func(v any, path string)
		walk = func(v any, path string) {
			switch val := v.(type) {
			case map[string]any:
				if url, ok := val["url"].(string); ok && url != "" {
					hasNested := false
					for k, cv := range val {
						if k == "extension" || k == "modifierExtension" {
							if arr, ok := cv.([]any); ok && len(arr) > 0 {
								hasNested = true
							}
						}
					}
					if hasNested {
						for k := range val {
							if k != "url" && k != "id" && k != "extension" && k != "modifierExtension" {
								t.Errorf("%s: complex extension %s carries a value[x] (%s): %#v", typ, url, k, val)
							}
						}
					}
				}
				for k, child := range val {
					walk(child, path+"."+k)
				}
			case []any:
				for _, item := range val {
					walk(item, path)
				}
			}
		}
		walk(out, typ)
	}
}

// TestSlicePatternOverlay verifies that a slice's Pattern value is overlaid
// onto the generated slice instance, so the value matches the slice's
// discriminator.
func TestSlicePatternOverlay(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())

	// A sliced element with one slice whose definition carries a Pattern.
	sliceDef := &fhir.ElementDefinition{
		ID:        "R.telecom:phone",
		Path:      "R.telecom",
		SliceName: "phone",
		Min:       1,
		Max:       1,
		Types:     []fhir.ElementType{{Code: "ContactPoint"}},
		Pattern:   map[string]any{"system": "email"},
	}
	elem := &fhir.ElementDefinition{
		ID:     "R.telecom",
		Path:   "R.telecom",
		Min:    1,
		Max:    1,
		Types:  []fhir.ElementType{{Code: "ContactPoint"}},
		Slices: []*fhir.SliceGroup{{Definition: sliceDef}},
	}
	tree := &fhir.ElementTree{
		Root:   &fhir.ElementDefinition{ID: "R", Path: "R", Min: 1, Max: 1},
		ByPath: map[string][]*fhir.ElementDefinition{},
		ByID:   map[string]*fhir.ElementDefinition{},
	}
	out := map[string]any{}
	if err := g.fillChild(elem, tree, out, 0); err != nil {
		t.Fatalf("fillChild: %v", err)
	}
	arr, ok := out["telecom"].([]any)
	if !ok || len(arr) == 0 {
		t.Fatalf("telecom = %#v, want non-empty slice array", out["telecom"])
	}
	first, ok := arr[0].(map[string]any)
	if !ok {
		t.Fatalf("telecom[0] = %#v, want map", arr[0])
	}
	if first["system"] != "email" {
		t.Errorf("telecom[0].system = %#v, want %q (slice pattern)", first["system"], "email")
	}
}

// TestExtensionsSatisfyExt1 verifies that every generated extension satisfies
// the FHIR ext-1 invariant: it carries either a value[x] or a nested
// extension array. This guards against emitting structurally invalid
// extensions (e.g. a complex extension with a spurious value[x], or a simple
// extension with neither).
func TestExtensionsSatisfyExt1(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())

	for _, typ := range []string{"Patient", "Organization", "Practitioner", "Observation"} {
		out, err := g.Generate(typ)
		if err != nil {
			t.Fatalf("Generate(%s): %v", typ, err)
		}
		var walk func(v any, path string)
		walk = func(v any, path string) {
			switch val := v.(type) {
			case map[string]any:
				if url, ok := val["url"].(string); ok && url != "" {
					// This is an extension object. It must have a value[x] or
					// a nested extension array.
					hasValue := false
					hasNested := false
					for k, cv := range val {
						if k == "url" || k == "id" {
							continue
						}
						if k == "extension" || k == "modifierExtension" {
							if arr, ok := cv.([]any); ok && len(arr) > 0 {
								hasNested = true
							}
							continue
						}
						hasValue = true
					}
					if !hasValue && !hasNested {
						t.Errorf("%s: extension %s violates ext-1 (no value[x], no nested extension): %#v", typ, url, val)
					}
				}
				for k, child := range val {
					walk(child, path+"."+k)
				}
			case []any:
				for _, item := range val {
					walk(item, path)
				}
			}
		}
		walk(out, typ)
	}
}
