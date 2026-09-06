package fhirgen

import (
	"fmt"
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
				switch strings.TrimSpace(sys) {
				case SystemHPIO:
					typed["value"] = g.fakeHPIO()
				case SystemHPII:
					typed["value"] = g.fakeHPII()
				case SystemABN:
					typed["value"] = g.fakeABN()
				case SystemACN:
					typed["value"] = g.fakeACN()
				case SystemAHPRA:
					typed["value"] = g.fakeAHPRA()
				}
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
