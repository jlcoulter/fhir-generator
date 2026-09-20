package fhirgen

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

const (
	SystemABN   = "http://hl7.org.au/id/abn"
	SystemACN   = "http://hl7.org.au/id/acn"
	SystemHPII  = "http://ns.electronichealth.net.au/id/hi/hpii/1.0"
	SystemHPIO  = "http://ns.electronichealth.net.au/id/hi/hpio/1.0"
	SystemAHPRA = "http://hl7.org.au/id/ahpra-registration-number"
)

var abnWeights = []int{10, 1, 3, 5, 7, 9, 11, 13, 15, 17, 19}
var acnWeights = []int{10, 1, 3, 5, 7, 9, 11, 13, 15}

func luhnCheckDigit(base string) string {
	if len(base) == 0 {
		return ""
	}
	sum := 0
	parity := (len(base) + 1) % 2
	for idx, r := range base {
		d := int(r - '0')
		if d < 0 || d > 9 {
			return base
		}
		if idx%2 == parity {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	checkDigit := (10 - (sum % 10)) % 10
	return base + strconv.Itoa(checkDigit)
}

func isValidLuhn(number string) bool {
	if number == "" {
		return false
	}
	sum := 0
	double := false
	for i := len(number) - 1; i >= 0; i-- {
		r := number[i]
		if r < '0' || r > '9' {
			return false
		}
		digit := int(r - '0')
		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		double = !double
	}
	return sum%10 == 0
}

func mod89CheckDigit(prefix string, weights []int, subtractFirst bool) (string, bool) {
	for c := 0; c <= 9; c++ {
		full := prefix + strconv.Itoa(c)
		if isMod89Valid(full, weights, subtractFirst) {
			return full, true
		}
	}
	return "", false
}

func isMod89Valid(number string, weights []int, subtractFirst bool) bool {
	if len(number) != len(weights) {
		return false
	}
	sum := 0
	for i := 0; i < len(number); i++ {
		d := int(number[i] - '0')
		if d < 0 || d > 9 {
			return false
		}
		if i == 0 && subtractFirst {
			d -= 1
		}
		sum += d * weights[i]
	}
	return sum%89 == 0
}

func (g *Generator) fakeABN() string {
	g.mu.Lock()
	base := 1000000000 + g.rng.Intn(9000000000)
	g.mu.Unlock()
	for i := range 100000 {
		prefix := fmt.Sprintf("%010d", base+i)
		if full, ok := mod89CheckDigit(prefix, abnWeights, true); ok {
			return full
		}
	}
	return "51824753556"
}

func (g *Generator) fakeACN() string {
	g.mu.Lock()
	base := 10000000 + g.rng.Intn(90000000)
	g.mu.Unlock()
	for i := range 100000 {
		prefix := fmt.Sprintf("%08d", base+i)
		if full, ok := mod89CheckDigit(prefix, acnWeights, false); ok {
			return full
		}
	}
	return "123456783"
}

func (g *Generator) fakeHPII() string {
	g.mu.Lock()
	suffix := fmt.Sprintf("%09d", g.rng.Intn(1000000000))
	g.mu.Unlock()
	return luhnCheckDigit("800361" + suffix)
}

func (g *Generator) fakeHPIO() string {
	g.mu.Lock()
	suffix := fmt.Sprintf("%09d", g.rng.Intn(1000000000))
	g.mu.Unlock()
	return luhnCheckDigit("800362" + suffix)
}

// FakeHPII returns a valid 16-digit Australian Healthcare Provider
// Identifier - Individual (HPI-I): prefix 800361, Luhn check digit.
func FakeHPII() string { return luhnCheckDigit("800361" + standaloneRandomDigits(9)) }

// FakeHPIO returns a valid 16-digit Australian Healthcare Provider
// Identifier - Organisation (HPI-O): prefix 800362, Luhn check digit.
func FakeHPIO() string { return luhnCheckDigit("800362" + standaloneRandomDigits(9)) }

// FakeABN returns a valid 11-digit Australian Business Number (mod-89).
func FakeABN() string {
	for i := 0; i < 100000; i++ {
		base := 1000000000 + (standaloneRand.Intn(9000000000)+i)%9000000000
		prefix := fmt.Sprintf("%010d", base)
		if full, ok := mod89CheckDigit(prefix, abnWeights, true); ok {
			return full
		}
	}
	return "51824753556"
}

// FakeACN returns a valid 9-digit Australian Company Number (mod-89).
func FakeACN() string {
	for i := 0; i < 100000; i++ {
		base := 10000000 + (standaloneRand.Intn(90000000)+i)%90000000
		prefix := fmt.Sprintf("%08d", base)
		if full, ok := mod89CheckDigit(prefix, acnWeights, false); ok {
			return full
		}
	}
	return "123456783"
}

// FakeAHPRA returns a syntactically valid Ahpra registration number: three
// uppercase letters followed by ten digits.
func FakeAHPRA() string { return "MED" + standaloneRandomDigits(10) }

// deterministicDigits returns n decimal digits derived from a hash of the seed
// string, so the same seed always yields the same digits regardless of call
// order (unlike the process-wide standaloneRand used by the unseeded Fake*
// constructors).
func deterministicDigits(seed string, n int) string {
	h := fnv32(seed)
	r := rand.New(rand.NewSource(int64(h)))
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteByte(byte('0' + r.Intn(10)))
	}
	return sb.String()
}

func fnv32(s string) uint32 {
	const (
		offset = 2166136261
		prime  = 16777619
	)
	h := uint32(offset)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime
	}
	return h
}

// FakeHPIIFromSeed returns a valid 16-digit HPI-I deterministically derived from
// seed, so callers generating reproducible corpora get the same value for the
// same seed regardless of generation order.
func FakeHPIIFromSeed(seed string) string { return luhnCheckDigit("800361" + deterministicDigits(seed+"|hpii", 9)) }

// FakeHPIOFromSeed returns a valid 16-digit HPI-O deterministically derived
// from seed.
func FakeHPIOFromSeed(seed string) string { return luhnCheckDigit("800362" + deterministicDigits(seed+"|hpio", 9)) }

// FakeAHPRAFromSeed returns a valid Ahpra registration number deterministically
// derived from seed.
func FakeAHPRAFromSeed(seed string) string { return "MED" + deterministicDigits(seed+"|ahpra", 10) }

// FakeABNFromSeed returns a valid 11-digit ABN deterministically derived from
// seed.
func FakeABNFromSeed(seed string) string {
	base := 1000000000 + int64(fnv32(seed+"|abn"))%9000000000
	for i := int64(0); i < 100000; i++ {
		prefix := fmt.Sprintf("%010d", base+i)
		if full, ok := mod89CheckDigit(prefix, abnWeights, true); ok {
			return full
		}
	}
	return "51824753556"
}

// FakeACNFromSeed returns a valid 9-digit ACN deterministically derived from
// seed.
func FakeACNFromSeed(seed string) string {
	base := 10000000 + int64(fnv32(seed+"|acn"))%90000000
	for i := int64(0); i < 100000; i++ {
		prefix := fmt.Sprintf("%08d", base+i)
		if full, ok := mod89CheckDigit(prefix, acnWeights, false); ok {
			return full
		}
	}
	return "123456783"
}

// FakeHPIIFromSeedOrValue returns a deterministic identifier value for a system
// URL, derived from seed, or "" when the system is not a supported AU system.
func FakeHPIIFromSeedOrValue(system, seed string) string {
	switch strings.TrimSpace(system) {
	case SystemHPIO:
		return FakeHPIOFromSeed(seed)
	case SystemHPII:
		return FakeHPIIFromSeed(seed)
	case SystemABN:
		return FakeABNFromSeed(seed)
	case SystemACN:
		return FakeACNFromSeed(seed)
	case SystemAHPRA:
		return FakeAHPRAFromSeed(seed)
	}
	return ""
}

// randomDigits returns n random decimal digits using the process-wide RNG.
func randomDigits(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteByte(byte('0' + rand.Intn(10)))
	}
	return sb.String()
}

// standaloneRand is a fixed-seed RNG backing the exported Fake* constructors so
// callers (e.g. momus) that generate reproducible corpora get deterministic,
// varied AU identifiers without tying them to a Generator's seeded RNG.
var standaloneRand = rand.New(rand.NewSource(0xF1A7))

// standaloneRandomDigits returns n random decimal digits from standaloneRand.
func standaloneRandomDigits(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteByte(byte('0' + standaloneRand.Intn(10)))
	}
	return sb.String()
}

func (g *Generator) fakeAHPRA() string {
	g.mu.Lock()
	d := g.rng.Intn(10000000000)
	g.mu.Unlock()
	return "MED" + fmt.Sprintf("%010d", d)
}

func (g *Generator) normaliseIdentifiers(v any) {
	switch typed := v.(type) {
	case map[string]any:
		if sys, ok := typed["system"].(string); ok {
			if _, ok := typed["value"]; ok {
				// Derive the identifier value deterministically from the system
				// (plus the generator's seed), never from the advancing g.rng:
				// normaliseIdentifiers iterates a Go map in nondeterministic
				// order, so consuming g.rng in map order would assign a
				// different value to the same identifier across runs. The RNG
				// draw is still consumed so the generator's downstream stream
				// position is unchanged for callers that reuse one generator
				// across resources.
				if v2 := g.seededIdentifierValue(sys); v2 != "" {
					typed["value"] = v2
				}
				g.rng.Intn(10000000000)
			}
		}
		for _, child := range typed {
			g.normaliseIdentifiers(child)
		}
	case []any:
		for _, child := range typed {
			g.normaliseIdentifiers(child)
		}
	}
}

// seededIdentifierValue returns a valid, deterministic AU identifier value for
// system, keyed by the generator's seed so different resources still vary but
// the same identifier is reproducible regardless of generation order.
func (g *Generator) seededIdentifierValue(system string) string {
	seed := fmt.Sprintf("%d|%s", g.seed, system)
	switch strings.TrimSpace(system) {
	case SystemHPIO:
		return FakeHPIOFromSeed(seed)
	case SystemHPII:
		return FakeHPIIFromSeed(seed)
	case SystemABN:
		return FakeABNFromSeed(seed)
	case SystemACN:
		return FakeACNFromSeed(seed)
	case SystemAHPRA:
		return FakeAHPRAFromSeed(seed)
	}
	return ""
}
