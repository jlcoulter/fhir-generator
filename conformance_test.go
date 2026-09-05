package fhirgen

import (
	"strings"
	"testing"
)

// TestPatternValueEmitted verifies that an element with a Pattern value emits
// that pattern verbatim (e.g. Identifier.type on au-ihi has a pattern with a
// fixed coding).
func TestPatternValueEmitted(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())

	// au-ihi Identifier.type has pattern {coding:[{code:NI, system:...}]}.
	tree, ok := reg.ResolveType("Identifier", []string{"http://hl7.org.au/fhir/StructureDefinition/au-ihi"})
	if !ok {
		t.Fatal("could not resolve au-ihi")
	}
	typeElems := tree.ByPath["Identifier.type"]
	if len(typeElems) == 0 {
		t.Fatal("no Identifier.type element")
	}
	if typeElems[0].Pattern == nil {
		t.Skip("au-ihi Identifier.type has no pattern in this package")
	}

	out := map[string]any{}
	if err := g.fillChild(typeElems[0], tree, out, 0); err != nil {
		t.Fatalf("fillChild: %v", err)
	}
	got, ok := out["type"].(map[string]any)
	if !ok {
		t.Fatalf("type = %#v, want map", out["type"])
	}
	coding, ok := got["coding"].([]any)
	if !ok || len(coding) == 0 {
		t.Fatalf("type.coding = %#v, want non-empty", got["coding"])
	}
	first, ok := coding[0].(map[string]any)
	if !ok {
		t.Fatalf("coding[0] = %#v, want map", coding[0])
	}
	if first["code"] != "NI" {
		t.Errorf("pattern code = %#v, want NI", first["code"])
	}
	if !strings.Contains(first["system"].(string), "v2-0203") {
		t.Errorf("pattern system = %#v, want v2-0203", first["system"])
	}
}

// TestFixedValueThroughTypeResolution verifies that a fixed value on a
// resolved complex type flows through (e.g. au-address country is fixed "AU").
func TestFixedValueThroughTypeResolution(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// Patient.address resolves to au-address, whose country is fixed "AU".
	addrs, ok := out["address"].([]any)
	if !ok || len(addrs) == 0 {
		t.Skip("no address generated")
	}
	addr, ok := addrs[0].(map[string]any)
	if !ok {
		t.Fatalf("address[0] = %#v", addrs[0])
	}
	if addr["country"] != "AU" {
		t.Errorf("address.country = %#v, want AU (fixed value)", addr["country"])
	}
}

// TestReferenceUsesTargetProfile verifies that a Reference element produces a
// realistic reference (ResourceType/id) derived from its target profile, not a
// raw profile URL.
func TestReferenceUsesTargetProfile(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())

	out, err := g.Generate("Organization")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	partOf, ok := out["partOf"].(map[string]any)
	if !ok {
		t.Fatalf("partOf = %#v, want map", out["partOf"])
	}
	ref, ok := partOf["reference"].(string)
	if !ok {
		t.Fatalf("partOf.reference = %#v, want string", partOf["reference"])
	}
	// Organization.partOf targets Organization, so reference must be
	// "Organization/<id>", not a URL.
	if !strings.HasPrefix(ref, "Organization/") {
		t.Errorf("partOf.reference = %q, want Organization/<id>", ref)
	}
	if strings.HasPrefix(ref, "http") {
		t.Errorf("partOf.reference = %q, should not be a raw URL", ref)
	}
}

// TestBindingAwareCode verifies that a coded element with a known binding
// produces a plausible code rather than a placeholder word.
func TestBindingAwareCode(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// Patient.gender is bound to administrative-gender.
	gender, ok := out["gender"].(string)
	if !ok {
		t.Skip("no gender generated")
	}
	switch gender {
	case "male", "female", "other", "unknown":
		// valid
	default:
		t.Errorf("gender = %q, want a plausible administrative-gender code", gender)
	}
}

// TestExtensionSliceUsesProfileURL verifies that a sliced extension element
// uses its profile URL (e.g. patient-birthPlace) rather than a generic
// placeholder URL.
func TestExtensionSliceUsesProfileURL(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	exts, ok := out["extension"].([]any)
	if !ok || len(exts) == 0 {
		t.Skip("no extension generated")
	}
	for _, e := range exts {
		ext, ok := e.(map[string]any)
		if !ok {
			continue
		}
		url, ok := ext["url"].(string)
		if !ok {
			continue
		}
		if strings.HasPrefix(url, "http://example.org") {
			t.Errorf("extension url = %q, should be a real profile URL", url)
		}
		if !strings.Contains(url, "StructureDefinition/") {
			t.Errorf("extension url = %q, want a StructureDefinition URL", url)
		}
	}
}

// TestBindingAwareScalarCode verifies that a scalar code element with a known
// binding (e.g. Address.state bound to australian-states-territories) produces
// a plausible code rather than a placeholder word.
func TestBindingAwareScalarCode(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	addrs, ok := out["address"].([]any)
	if !ok || len(addrs) == 0 {
		t.Skip("no address generated")
	}
	addr, ok := addrs[0].(map[string]any)
	if !ok {
		t.Fatalf("address[0] = %#v", addrs[0])
	}
	state, ok := addr["state"].(string)
	if !ok {
		t.Skip("no state generated")
	}
	// AU states are codes like NSW, VIC, QLD, SA, WA, TAS, NT, ACT.
	valid := map[string]bool{"NSW": true, "VIC": true, "QLD": true, "SA": true, "WA": true, "TAS": true, "NT": true, "ACT": true}
	if !valid[state] {
		t.Errorf("address.state = %q, want a valid AU state code", state)
	}
}

// TestExtensionHasValue verifies that a generated extension includes a value
// (value[x]) when the extension profile requires one, not just a url.
func TestExtensionHasValue(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	exts, ok := out["extension"].([]any)
	if !ok || len(exts) == 0 {
		t.Skip("no extension generated")
	}
	for _, e := range exts {
		ext, ok := e.(map[string]any)
		if !ok {
			continue
		}
		url, _ := ext["url"].(string)
		// AU extensions like indigenous-status require a value[x].
		if strings.Contains(url, "indigenous-status") {
			if _, hasValue := ext["valueCoding"]; !hasValue {
				t.Errorf("extension %s missing valueCoding: %#v", url, ext)
			}
		}
	}
}

// TestUnslicedExtensionSkipped verifies that extension and modifierExtension
// elements with no profile hint and no slices are not generated, since a valid
// extension URL cannot be derived without IG context.
func TestUnslicedExtensionSkipped(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// Patient.modifierExtension has no slices and no profile hint, so it must
	// not be emitted.
	if _, ok := out["modifierExtension"]; ok {
		t.Errorf("modifierExtension should be skipped (no profile/slices), got %#v", out["modifierExtension"])
	}
}

// TestNoPlaceholderExtensionURLs verifies that no generated extension or
// modifier extension uses a placeholder example.org URL.
func TestNoPlaceholderExtensionURLs(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())

	out, err := g.Generate("Patient")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var walk func(v any)
	walk = func(v any) {
		switch val := v.(type) {
		case map[string]any:
			if url, ok := val["url"].(string); ok && strings.HasPrefix(url, "http://example.org") {
				t.Errorf("placeholder extension url = %q", url)
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

// TestElementTypeProfileResolution verifies that an element with type profiles
// resolves through the first profile to produce richer output (e.g.
// Organization.identifier resolves through au-ihi or another AU profile).
func TestElementTypeProfileResolution(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())

	out, err := g.Generate("Organization")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	ids, ok := out["identifier"].([]any)
	if !ok || len(ids) == 0 {
		t.Skip("no identifier generated")
	}
	for _, id := range ids {
		ident, ok := id.(map[string]any)
		if !ok {
			continue
		}
		sys, ok := ident["system"].(string)
		if !ok {
			continue
		}
		if strings.HasPrefix(sys, "urn:oid:1.2.3.4.5") {
			t.Errorf("identifier.system = %q, should be a real AU system", sys)
		}
	}
}
