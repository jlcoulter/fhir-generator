package fhirgen

import (
	"strings"
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

// TestNoContainedAbstractResourceType verifies that generated resources never
// contain a "contained" entry with resourceType "Resource" or other abstract
// types. FHIR servers reject these (HAPI-1684).
func TestNoContainedAbstractResourceType(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())

	for _, typ := range []string{"Patient", "Organization", "Practitioner", "Observation", "Location", "HealthcareService", "PractitionerRole"} {
		out, err := g.Generate(typ)
		if err != nil {
			t.Fatalf("Generate(%s): %v", typ, err)
		}
		contained, ok := out["contained"].([]any)
		if !ok {
			continue
		}
		for i, item := range contained {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			rt, _ := m["resourceType"].(string)
			switch rt {
			case "Resource", "DomainResource":
				t.Errorf("%s: contained[%d] has abstract resourceType %q", typ, i, rt)
			}
		}
	}
}

// TestFakeReferenceNoAbstractResourceType verifies that generated references
// never produce "Resource" as the resource type when no target profile is
// available. "Resource" is an abstract base type.
func TestFakeReferenceNoAbstractResourceType(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())

	for _, typ := range []string{"Patient", "Organization", "Practitioner", "Observation", "Location", "HealthcareService"} {
		out, err := g.Generate(typ)
		if err != nil {
			t.Fatalf("Generate(%s): %v", typ, err)
		}
		var walk func(v any)
		walk = func(v any) {
			switch val := v.(type) {
			case map[string]any:
				if ref, ok := val["reference"].(string); ok && ref != "" {
					parts := strings.SplitN(ref, "/", 2)
					if len(parts) == 2 {
						rt := parts[0]
						if rt == "Resource" || rt == "DomainResource" {
							t.Errorf("%s: reference has abstract resourceType in %q", typ, ref)
						}
					}
				}
				for _, child := range val {
					walk(child)
				}
			case []any:
				for _, item := range val {
					walk(item)
				}
			}
		}
		walk(out)
	}
}

// TestEnforceExt1StripsValueFromComplexExtension verifies that when a
// generated extension has both sub-extensions and a value[x], the value[x]
// is removed (enforcing ext-1).
func TestEnforceExt1StripsValueFromComplexExtension(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())

	for _, typ := range []string{"Patient", "Organization", "Practitioner", "Observation"} {
		out, err := g.Generate(typ)
		if err != nil {
			t.Fatalf("Generate(%s): %v", typ, err)
		}
		var walk func(v any)
		walk = func(v any) {
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
								t.Errorf("%s: complex extension %s still has value[x] %s after enforceExt1", typ, url, k)
							}
						}
					}
				}
				for _, child := range val {
					walk(child)
				}
			case []any:
				for _, item := range val {
					walk(item)
				}
			}
		}
		walk(out)
	}
}

// TestSliceChildPatternFixesNestedCoding verifies that a slice whose child
// value[x].coding fixes a code applies that fixed value, preserving the array
// shape of the repeating "coding" element.
func TestSliceChildPatternFixesNestedCoding(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))
	_ = g

	codeFixed := map[string]any{"system": "http://cs", "code": "organisation-initiated"}
	vx := &fhir.ElementDefinition{
		ID:    "X.extension.extension:sub.value[x]",
		Path:  "X.extension.extension.value[x]",
		Types: []fhir.ElementType{{Code: "CodeableConcept"}},
		Children: []*fhir.ElementDefinition{
			{ID: "X.extension.extension:sub.value[x].coding", Path: "X.extension.extension.value[x].coding", Fixed: codeFixed},
		},
	}
	subSlice := &fhir.ElementDefinition{
		ID:        "X.extension.extension:sub",
		Path:      "X.extension.extension",
		SliceName: "suppressedBy",
		Children: []*fhir.ElementDefinition{
			{ID: "X.extension.extension:sub.url", Path: "X.extension.extension.url", Fixed: "suppressedBy"},
			vx,
		},
	}
	extChild := &fhir.ElementDefinition{
		ID:     "X.extension.extension",
		Path:   "X.extension.extension",
		Types:  []fhir.ElementType{{Code: "Extension"}},
		Slices: []*fhir.SliceGroup{{Name: "suppressedBy", Definition: subSlice}},
	}
	slice := &fhir.ElementDefinition{
		ID:        "X.extension:suppressed",
		Path:      "X.extension",
		SliceName: "suppressed",
		Types:     []fhir.ElementType{{Code: "Extension"}},
		Children: []*fhir.ElementDefinition{
			{ID: "X.extension:suppressed.url", Path: "X.extension.url", Fixed: "http://ext/suppressed"},
			extChild,
		},
	}

	value := map[string]any{
		"url": "http://ext/suppressed",
		"extension": []any{
			map[string]any{"url": "suppressedBy", "valueCodeableConcept": map[string]any{
				"coding": []any{map[string]any{"code": "practitioner-initiated", "system": "http://cs"}},
			}},
		},
	}
	applySliceChildPatterns(value, slice)

	exts := value["extension"].([]any)
	m := exts[0].(map[string]any)
	vcc := m["valueCodeableConcept"].(map[string]any)
	codings, ok := vcc["coding"].([]any)
	if !ok || len(codings) == 0 {
		t.Fatalf("coding should be an array, got %T", vcc["coding"])
	}
	first := codings[0].(map[string]any)
	if first["code"] != "organisation-initiated" {
		t.Errorf("coding code = %v, want organisation-initiated", first["code"])
	}
}

// TestFixedChildReplaceNotMerge verifies that a Fixed value on a child replaces
// the generated target exactly (killing any synthesized display/text), whereas
// a Pattern value merges beneath the generated siblings.
func TestFixedChildReplaceNotMerge(t *testing.T) {
	fixed := map[string]any{"system": "http://cs", "code": "org-initiated"}
	pattern := map[string]any{"system": "http://cs2", "code": "pat"}

	// A generated CodeableConcept with a stale display/text and two codings.
	value := map[string]any{"coding": []any{
		map[string]any{"system": "http://cs", "code": "x", "display": "Stale"},
		map[string]any{"system": "http://cs", "code": "y"},
	}, "text": "stale text"}

	fixedChild := &fhir.ElementDefinition{
		ID:    "R.value[x].coding",
		Path:  "R.value[x].coding",
		Types: []fhir.ElementType{{Code: "Coding"}},
		Fixed: fixed,
	}
	// Fixed replaces the whole coding subtree.
	copy1 := cloneValue(value).(map[string]any)
	applySliceChildPattern(copy1, fixedChild)
	codings, ok := copy1["coding"].([]any)
	if !ok || len(codings) != 1 {
		t.Fatalf("fixed coding = %T, want single-element array", copy1["coding"])
	}
	c0 := codings[0].(map[string]any)
	if c0["code"] != "org-initiated" {
		t.Errorf("fixed coding code = %v, want org-initiated", c0["code"])
	}
	if _, hasDisplay := c0["display"]; hasDisplay {
		t.Error("fixed coding kept a stale display")
	}
	if _, hasText := copy1["text"]; hasText {
		t.Error("fixed CodeableConcept kept a stale text")
	}
	if _, marked := c0[FixedCodingKey]; !marked {
		t.Error("fixed coding not marked with FixedCodingKey")
	}

	// Pattern merges: the generated coding array is replaced, but the sibling
	// text is preserved (pattern allows extra properties).
	patternChild := &fhir.ElementDefinition{
		ID:    "R.value[x].coding",
		Path:  "R.value[x].coding",
		Types: []fhir.ElementType{{Code: "Coding"}},
		Pattern: pattern,
	}
	copy2 := cloneValue(value).(map[string]any)
	applySliceChildPattern(copy2, patternChild)
	if _, hasText := copy2["text"]; !hasText {
		t.Error("pattern overlay dropped sibling text")
	}
}

// TestFixedCodingMarkedAndStripped verifies that codings materialised from a
// Fixed value carry the FixedCodingKey marker, and that StripFixedCodingMarkers
// removes it recursively.
func TestFixedCodingMarkedAndStripped(t *testing.T) {
	coding := map[string]any{"system": "http://cs", "code": "c"}
	markFixedCodings(coding)
	if _, marked := coding[FixedCodingKey]; !marked {
		t.Fatal("markFixedCodings did not mark a bare coding")
	}

	cc := map[string]any{"coding": []any{map[string]any{"system": "s", "code": "c"}}, "text": "T"}
	markFixedCodings(cc)
	if _, hasText := cc["text"]; hasText {
		t.Fatal("markFixedCodings should strip CodeableConcept text")
	}
	inner := cc["coding"].([]any)[0].(map[string]any)
	if _, marked := inner[FixedCodingKey]; !marked {
		t.Fatal("CodeableConcept coding not marked")
	}

	payload := map[string]any{"coding": []any{map[string]any{FixedCodingKey: true}}, FixedCodingKey: true}
	StripFixedCodingMarkers(payload)
	if _, ok := payload[FixedCodingKey]; ok {
		t.Fatal("top-level marker not stripped")
	}
	if _, ok := payload["coding"].([]any)[0].(map[string]any)[FixedCodingKey]; ok {
		t.Fatal("nested marker not stripped")
	}
}

// TestFakePeriodOrdered verifies that a generated Period always has start < end
// (satisfying the per-1/base period invariants), across many seeds.
func TestFakePeriodOrdered(t *testing.T) {
	reg := loadTestRegistry(t)
	for seed := int64(0); seed < 50; seed++ {
		g := New(reg, WithSeed(seed), WithFullFillMode())
		period := g.fakePeriod()
		start := period["start"].(string)
		end := period["end"].(string)
		if start >= end {
			t.Fatalf("seed %d: period start %q not before end %q", seed, start, end)
		}
	}
}

// TestFakeUseIsNeverHome verifies generated ContactPoint/Address/synthesized
// "use" values are never 'home', so Organization telecom/address satisfy
// org-3/org-2.
func TestFakeUseIsNeverHome(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())
	for i := 0; i < 50; i++ {
		if g.fakeContactPoint()["use"] == "home" {
			t.Fatal("fakeContactPoint use must not be home")
		}
		if g.fakeAddress()["use"] == "home" {
			t.Fatal("fakeAddress use must not be home")
		}
	}
	if c, ok := synthesizeCode("use"); !ok || c == "home" {
		t.Fatalf("synthesizeCode(use) = %q, ok=%v; want non-home", c, ok)
	}
	if _, c, ok := knownBinding("http://hl7.org/fhir/ValueSet/contact-point-use"); !ok || c == "home" {
		t.Fatalf("knownBinding(contact-point-use) = %q, ok=%v; want non-home", c, ok)
	}
	if _, c, ok := knownBinding("http://hl7.org/fhir/ValueSet/address-use"); !ok || c == "home" {
		t.Fatalf("knownBinding(address-use) = %q, ok=%v; want non-home", c, ok)
	}
}

// TestFillSlicesRespectsParentMax verifies that a sliced element whose own Max
// is bounded emits at most Max slice instances, preferring required slices.
func TestFillSlicesRespectsParentMax(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())

	mkSlice := func(name string, min int) *fhir.SliceGroup {
		return &fhir.SliceGroup{Name: name, Definition: &fhir.ElementDefinition{
			ID:        "R.identifier:" + name,
			Path:      "R.identifier",
			SliceName: name,
			Min:       min,
			Max:       1,
			Types:     []fhir.ElementType{{Code: "Identifier"}},
		}}
	}
	// Parent identifier has Max 1 but two optional slices.
	elem := &fhir.ElementDefinition{
		ID:     "R.identifier",
		Path:   "R.identifier",
		Min:    0,
		Max:    1,
		Types:  []fhir.ElementType{{Code: "Identifier"}},
		Slices: []*fhir.SliceGroup{mkSlice("ahpra", 0), mkSlice("pbprn", 0)},
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
	arr, ok := out["identifier"].([]any)
	if !ok {
		t.Fatalf("identifier = %#v, want array", out["identifier"])
	}
	if len(arr) > 1 {
		t.Errorf("identifier has %d slices, want at most 1 (parent max)", len(arr))
	}

	// With an unbounded parent both optional slices are emitted.
	unbounded := &fhir.ElementDefinition{
		ID:     "R.identifier",
		Path:   "R.identifier",
		Min:    0,
		Max:    fhir.MaxUnbounded,
		Types:  []fhir.ElementType{{Code: "Identifier"}},
		Slices: []*fhir.SliceGroup{mkSlice("ahpra", 0), mkSlice("pbprn", 0)},
	}
	out2 := map[string]any{}
	if err := g.fillChild(unbounded, tree, out2, 0); err != nil {
		t.Fatalf("fillChild unbounded: %v", err)
	}
	if arr2, ok := out2["identifier"].([]any); !ok || len(arr2) < 2 {
		t.Errorf("unbounded identifier = %#v, want >= 2 slices", out2["identifier"])
	}
}

