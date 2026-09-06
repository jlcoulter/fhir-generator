package fhirgen

import (
	"strings"
	"testing"
)

func TestLuhnCheckDigit(t *testing.T) {
	tests := []struct {
		base string
		want string
	}{
		{"7992739871", "79927398713"},
		{"800361123456789", "8003611234567893"},
		{"800362123456789", "8003621234567892"},
	}
	for _, tc := range tests {
		got := luhnCheckDigit(tc.base)
		if got != tc.want {
			t.Errorf("luhnCheckDigit(%q) = %q, want %q", tc.base, got, tc.want)
		}
	}
}

func TestIsValidLuhn(t *testing.T) {
	valid := []string{
		"79927398713",
		"8003611234567893",
		"8003621234567892",
	}
	for _, v := range valid {
		if !isValidLuhn(v) {
			t.Errorf("isValidLuhn(%q) = false, want true", v)
		}
	}
	invalid := []string{
		"",
		"79927398710",
		"abc",
		"1234",
	}
	for _, v := range invalid {
		if isValidLuhn(v) {
			t.Errorf("isValidLuhn(%q) = true, want false", v)
		}
	}
}

func TestIsMod89Valid(t *testing.T) {
	abn := func(n string) bool { return isMod89Valid(n, abnWeights, true) }
	acn := func(n string) bool { return isMod89Valid(n, acnWeights, false) }

	if !abn("51824753556") {
		t.Error("expected known valid ABN to pass")
	}
	if abn("51824753557") {
		t.Error("expected modified ABN to fail")
	}
	if !acn("123456783") {
		t.Error("expected known valid ACN to pass")
	}
	if acn("123456780") {
		t.Error("expected modified ACN to fail")
	}
	if abn("") {
		t.Error("expected empty string to fail")
	}
	if abn("abc") {
		t.Error("expected non-digit string to fail")
	}
	if acn("") {
		t.Error("expected empty string to fail")
	}
	if acn("abc") {
		t.Error("expected non-digit string to fail")
	}
}

func TestMod89CheckDigit(t *testing.T) {
	prefix := "5182475355"
	full, ok := mod89CheckDigit(prefix, abnWeights, true)
	if !ok {
		t.Fatal("mod89CheckDigit failed to find valid check digit")
	}
	if full != "51824753556" {
		t.Errorf("mod89CheckDigit = %q, want %q", full, "51824753556")
	}
}

func TestFakeABN(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))
	for i := 0; i < 50; i++ {
		abn := g.fakeABN()
		if len(abn) != 11 {
			t.Fatalf("fakeABN() = %q, length %d, want 11", abn, len(abn))
		}
		if abn[0] == '0' {
			t.Fatalf("fakeABN() = %q, first digit is 0", abn)
		}
		if !isMod89Valid(abn, abnWeights, true) {
			t.Fatalf("fakeABN() = %q, failed mod-89 validation", abn)
		}
	}
}

func TestFakeABNDeterministic(t *testing.T) {
	reg := loadTestRegistry(t)
	g1 := New(reg, WithSeed(42))
	g2 := New(reg, WithSeed(42))
	for i := 0; i < 10; i++ {
		a, b := g1.fakeABN(), g2.fakeABN()
		if a != b {
			t.Fatalf("seed=42 produced different ABNs on iteration %d: %q vs %q", i, a, b)
		}
	}
}

func TestFakeACN(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))
	for i := 0; i < 50; i++ {
		acn := g.fakeACN()
		if len(acn) != 9 {
			t.Fatalf("fakeACN() = %q, length %d, want 9", acn, len(acn))
		}
		if !isMod89Valid(acn, acnWeights, false) {
			t.Fatalf("fakeACN() = %q, failed mod-89 validation", acn)
		}
	}
}

func TestFakeACNDeterministic(t *testing.T) {
	reg := loadTestRegistry(t)
	g1 := New(reg, WithSeed(42))
	g2 := New(reg, WithSeed(42))
	for i := 0; i < 10; i++ {
		a, b := g1.fakeACN(), g2.fakeACN()
		if a != b {
			t.Fatalf("seed=42 produced different ACNs on iteration %d: %q vs %q", i, a, b)
		}
	}
}

func TestFakeHPII(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))
	for i := 0; i < 50; i++ {
		hpi := g.fakeHPII()
		if len(hpi) != 16 {
			t.Fatalf("fakeHPII() = %q, length %d, want 16", hpi, len(hpi))
		}
		if !strings.HasPrefix(hpi, "800361") {
			t.Fatalf("fakeHPII() = %q, prefix != 800361", hpi)
		}
		if !isValidLuhn(hpi) {
			t.Fatalf("fakeHPII() = %q, failed Luhn check", hpi)
		}
	}
}

func TestFakeHPIODeterministic(t *testing.T) {
	reg := loadTestRegistry(t)
	g1 := New(reg, WithSeed(42))
	g2 := New(reg, WithSeed(42))
	for i := 0; i < 10; i++ {
		a, b := g1.fakeHPIO(), g2.fakeHPIO()
		if a != b {
			t.Fatalf("seed=42 produced different HPI-Os on iteration %d: %q vs %q", i, a, b)
		}
	}
}

func TestFakeHPIO(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))
	for i := 0; i < 50; i++ {
		hpi := g.fakeHPIO()
		if len(hpi) != 16 {
			t.Fatalf("fakeHPIO() = %q, length %d, want 16", hpi, len(hpi))
		}
		if !strings.HasPrefix(hpi, "800362") {
			t.Fatalf("fakeHPIO() = %q, prefix != 800362", hpi)
		}
		if !isValidLuhn(hpi) {
			t.Fatalf("fakeHPIO() = %q, failed Luhn check", hpi)
		}
	}
}

func TestFakeHPIIDeterministic(t *testing.T) {
	reg := loadTestRegistry(t)
	g1 := New(reg, WithSeed(42))
	g2 := New(reg, WithSeed(42))
	for i := 0; i < 10; i++ {
		a, b := g1.fakeHPII(), g2.fakeHPII()
		if a != b {
			t.Fatalf("seed=42 produced different HPI-Is on iteration %d: %q vs %q", i, a, b)
		}
	}
}

func TestFakeAHPRA(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))
	for i := 0; i < 50; i++ {
		ahpra := g.fakeAHPRA()
		if len(ahpra) != 13 {
			t.Fatalf("fakeAHPRA() = %q, length %d, want 13", ahpra, len(ahpra))
		}
		if !strings.HasPrefix(ahpra, "MED") {
			t.Fatalf("fakeAHPRA() = %q, prefix != MED", ahpra)
		}
		tail := ahpra[3:]
		for _, r := range tail {
			if r < '0' || r > '9' {
				t.Fatalf("fakeAHPRA() = %q, non-digit in tail %q", ahpra, tail)
			}
		}
	}
}

func TestFakeAHPRADeterministic(t *testing.T) {
	reg := loadTestRegistry(t)
	g1 := New(reg, WithSeed(42))
	g2 := New(reg, WithSeed(42))
	for i := 0; i < 10; i++ {
		a, b := g1.fakeAHPRA(), g2.fakeAHPRA()
		if a != b {
			t.Fatalf("seed=42 produced different AHPRA on iteration %d: %q vs %q", i, a, b)
		}
	}
}

func TestFakeIdentifierProducesValidAUIdentifiers(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))
	seen := map[string]int{}
	for i := 0; i < 100; i++ {
		id := g.fakeIdentifier()
		sys, _ := id["system"].(string)
		val, _ := id["value"].(string)
		if sys == "" || val == "" {
			t.Fatalf("fakeIdentifier() missing system or value: %+v", id)
		}
		seen[sys]++
		switch sys {
		case SystemABN:
			if len(val) != 11 || !isMod89Valid(val, abnWeights, true) {
				t.Fatalf("fakeIdentifier() ABN %q invalid", val)
			}
		case SystemACN:
			if len(val) != 9 || !isMod89Valid(val, acnWeights, false) {
				t.Fatalf("fakeIdentifier() ACN %q invalid", val)
			}
		case SystemHPII:
			if len(val) != 16 || !isValidLuhn(val) {
				t.Fatalf("fakeIdentifier() HPI-I %q invalid", val)
			}
		case SystemHPIO:
			if len(val) != 16 || !isValidLuhn(val) {
				t.Fatalf("fakeIdentifier() HPI-O %q invalid", val)
			}
		case SystemAHPRA:
			if len(val) != 13 || !strings.HasPrefix(val, "MED") {
				t.Fatalf("fakeIdentifier() AHPRA %q invalid", val)
			}
		case "urn:oid:1.2.3.4.5":
			if len(val) != 8 {
				t.Fatalf("fakeIdentifier() generic %q invalid length", val)
			}
		default:
			t.Fatalf("fakeIdentifier() unexpected system %q", sys)
		}
	}
	if len(seen) < 5 {
		t.Fatalf("fakeIdentifier() only produced %d distinct systems in 100 iterations, expected at least 5: %v", len(seen), seen)
	}
}

func TestNormaliseIdentifiersFixesHPIO(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))
	id := map[string]any{
		"system": SystemHPIO,
		"value":  "junk",
	}
	g.normaliseIdentifiers(id)
	val, _ := id["value"].(string)
	if len(val) != 16 {
		t.Fatalf("after normalize, value %q length %d, want 16", val, len(val))
	}
	if !isValidLuhn(val) {
		t.Fatalf("after normalize, value %q failed Luhn check", val)
	}
}

func TestNormaliseIdentifiersFixesHPII(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))
	id := map[string]any{
		"system": SystemHPII,
		"value":  "junk",
	}
	g.normaliseIdentifiers(id)
	val, _ := id["value"].(string)
	if len(val) != 16 {
		t.Fatalf("after normalize, value %q length %d, want 16", val, len(val))
	}
	if !isValidLuhn(val) {
		t.Fatalf("after normalize, value %q failed Luhn check", val)
	}
}

func TestNormaliseIdentifiersFixesABN(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))
	id := map[string]any{
		"system": SystemABN,
		"value":  "junk",
	}
	g.normaliseIdentifiers(id)
	val, _ := id["value"].(string)
	if len(val) != 11 {
		t.Fatalf("after normalize, value %q length %d, want 11", val, len(val))
	}
	if !isMod89Valid(val, abnWeights, true) {
		t.Fatalf("after normalize, value %q failed mod-89", val)
	}
}

func TestNormaliseIdentifiersFixesACN(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))
	id := map[string]any{
		"system": SystemACN,
		"value":  "junk",
	}
	g.normaliseIdentifiers(id)
	val, _ := id["value"].(string)
	if len(val) != 9 {
		t.Fatalf("after normalize, value %q length %d, want 9", val, len(val))
	}
	if !isMod89Valid(val, acnWeights, false) {
		t.Fatalf("after normalize, value %q failed mod-89", val)
	}
}

func TestNormaliseIdentifiersFixesAHPRA(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))
	id := map[string]any{
		"system": SystemAHPRA,
		"value":  "junk",
	}
	g.normaliseIdentifiers(id)
	val, _ := id["value"].(string)
	if len(val) != 13 {
		t.Fatalf("after normalize, value %q length %d, want 13", val, len(val))
	}
	if !strings.HasPrefix(val, "MED") {
		t.Fatalf("after normalize, value %q prefix != MED", val)
	}
}

func TestNormaliseIdentifiersSkipsUnknown(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))
	id := map[string]any{
		"system": "urn:unknown:system",
		"value":  "keep-me",
	}
	g.normaliseIdentifiers(id)
	val, _ := id["value"].(string)
	if val != "keep-me" {
		t.Fatalf("unknown system changed value from 'keep-me' to %q", val)
	}
}

func TestNormaliseIdentifiersSkipsNoSystem(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))
	id := map[string]any{
		"value": "keep-me",
	}
	g.normaliseIdentifiers(id)
	val, _ := id["value"].(string)
	if val != "keep-me" {
		t.Fatalf("map without system changed value to %q", val)
	}
}

func TestNormaliseIdentifiersRecursive(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42))
	resource := map[string]any{
		"resourceType": "Patient",
		"identifier": []any{
			map[string]any{
				"system": SystemABN,
				"value":  "junk",
			},
			map[string]any{
				"system": SystemHPII,
				"value":  "junk",
			},
		},
	}
	g.normaliseIdentifiers(resource)
	idents, _ := resource["identifier"].([]any)
	if len(idents) != 2 {
		t.Fatalf("expected 2 identifiers, got %d", len(idents))
	}
	for i, raw := range idents {
		id, _ := raw.(map[string]any)
		val, _ := id["value"].(string)
		if len(val) < 9 {
			t.Fatalf("identifier[%d] value %q too short after normalize", i, val)
		}
	}
}

func TestNormaliseIdentifiersInGenerate(t *testing.T) {
	reg := loadTestRegistry(t)
	g := New(reg, WithSeed(42), WithFullFillMode())
	org, err := g.Generate("Organization")
	if err != nil {
		t.Fatalf("Generate(Organization): %v", err)
	}
	idents, _ := org["identifier"].([]any)
	for _, raw := range idents {
		id, _ := raw.(map[string]any)
		sys, _ := id["system"].(string)
		val, _ := id["value"].(string)
		switch sys {
		case SystemABN:
			if !isMod89Valid(val, abnWeights, true) {
				t.Errorf("generated Organization has invalid ABN %q", val)
			}
		case SystemACN:
			if !isMod89Valid(val, acnWeights, false) {
				t.Errorf("generated Organization has invalid ACN %q", val)
			}
		case SystemHPII:
			if !isValidLuhn(val) {
				t.Errorf("generated Organization has invalid HPI-I %q", val)
			}
		case SystemHPIO:
			if !isValidLuhn(val) {
				t.Errorf("generated Organization has invalid HPI-O %q", val)
			}
		case SystemAHPRA:
			if len(val) != 13 || !strings.HasPrefix(val, "MED") {
				t.Errorf("generated Organization has invalid AHPRA %q", val)
			}
		}
	}
}

func TestFakeABNFallback(t *testing.T) {
	if !isMod89Valid("51824753556", abnWeights, true) {
		t.Error("fallback ABN is not valid")
	}
}

func TestFakeACNFallback(t *testing.T) {
	if !isMod89Valid("123456783", acnWeights, false) {
		t.Error("fallback ACN is not valid")
	}
}