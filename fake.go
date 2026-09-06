package fhirgen

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	fhir "github.com/jlcoulter/fhir-registry"
)

// randN returns a non-negative pseudo-random number in [0, n).
func (g *Generator) randN(n int) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.rng.Intn(n)
}

// randFloat returns a pseudo-random float64 in [0, 1).
func (g *Generator) randFloat() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.rng.Float64()
}

func (g *Generator) fakeBool() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.rng.Intn(2) == 0
}

func (g *Generator) fakeInt() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.rng.Intn(1000)
}

func (g *Generator) fakeDecimal() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return float64(g.rng.Intn(10000)) / 100
}

func (g *Generator) fakeDate() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	start := time.Date(1950, 1, 1, 0, 0, 0, 0, time.UTC)
	days := g.rng.Intn(70*365 + 1)
	return start.AddDate(0, 0, days).Format("2006-01-02")
}

func (g *Generator) fakeDateTime() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	start := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	days := g.rng.Intn(25*365 + 1)
	return start.AddDate(0, 0, days).Format(time.RFC3339)
}

func (g *Generator) fakeTime() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fmt.Sprintf("%02d:%02d:%02d", g.rng.Intn(24), g.rng.Intn(60), g.rng.Intn(60))
}

// fakeStringFor produces a plausible string for an element, using the element's
// path and binding to synthesize a sensible value where possible.
func (g *Generator) fakeStringFor(elem *fhir.ElementDefinition, tree *fhir.ElementTree) string {
	seg := lastSegment(elem.Path)
	lower := strings.ToLower(seg)

	// A configured BindingResolver takes precedence for any element with a
	// value set binding, so callers can override even well-known codes.
	if elem.Binding != nil && elem.Binding.ValueSet != "" && g.bindingResolver != nil {
		if rc, ok := g.bindingResolver.ResolveBinding(elem, tree); ok {
			return rc.Code
		}
	}

	// Coded elements: synthesize a plausible code from the path or binding.
	if fhir.PrimaryTypeCode(elem) == "code" {
		if c, ok := synthesizeCode(lower); ok {
			return c
		}
	}
	// Elements with a value set binding (code or string typed) produce a
	// plausible bound code.
	if elem.Binding != nil && elem.Binding.ValueSet != "" {
		if _, code, ok := knownBinding(elem.Binding.ValueSet); ok {
			return code
		}
	}

	// Common named fields.
	switch lower {
	case "id":
		return g.fakeID()
	case "url":
		return "http://example.org/fhir/StructureDefinition/" + g.fakeID()
	case "system":
		return "urn:oid:1.2.3.4.5"
	case "value":
		return g.fakeID()
	case "family":
		return g.fakeFamilyName()
	case "given":
		return g.fakeGivenName()
	case "text":
		return g.fakeSentence()
	case "display":
		return g.fakeSentence()
	case "version":
		return "1.0.0"
	case "language":
		return "en"
	case "status":
		return "active"
	case "name":
		return g.fakeCompanyName()
	case "title":
		return g.fakeSentence()
	case "email":
		return g.fakeEmail()
	case "phone":
		return g.fakePhone()
	case "gender":
		return g.fakeGender()
	case "birthDate":
		return g.fakeDate()
	}
	return g.fakeWord()
}

func (g *Generator) fakeID() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[g.rng.Intn(len(chars))]
	}
	return string(b)
}

func (g *Generator) fakeWord() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return pickWord(g.rng)
}

func pickWord(r *rand.Rand) string {
	words := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta", "iota", "kappa"}
	return words[r.Intn(len(words))]
}

func (g *Generator) fakeSentence() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	words := []string{"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing", "elit"}
	n := 3 + g.rng.Intn(5)
	parts := make([]string, n)
	for i := range parts {
		parts[i] = words[g.rng.Intn(len(words))]
	}
	return strings.Join(parts, " ")
}

func (g *Generator) fakeGivenName() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	names := []string{"James", "Mary", "John", "Patricia", "Robert", "Jennifer", "Michael", "Linda"}
	return names[g.rng.Intn(len(names))]
}

func (g *Generator) fakeFamilyName() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	names := []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis"}
	return names[g.rng.Intn(len(names))]
}

func (g *Generator) fakeCompanyName() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	names := []string{"Acme", "Globex", "Initech", "Umbrella", "Stark", "Wayne", "Hooli", "Vandelay"}
	return names[g.rng.Intn(len(names))] + " " + pickWord(g.rng)
}

func (g *Generator) fakeEmail() string {
	return strings.ToLower(g.fakeGivenName()) + "." + strings.ToLower(g.fakeFamilyName()) + "@example.org"
}

func (g *Generator) fakePhone() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fmt.Sprintf("+1-%03d-%03d-%04d", g.rng.Intn(1000), g.rng.Intn(1000), g.rng.Intn(10000))
}

func (g *Generator) fakeGender() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	genders := []string{"male", "female", "other", "unknown"}
	return genders[g.rng.Intn(len(genders))]
}

// synthesizeCode returns a plausible code for a coded element based on its
// path segment.
func synthesizeCode(seg string) (string, bool) {
	switch seg {
	case "use":
		return "home", true
	case "status":
		return "active", true
	case "gender":
		return "male", true
	case "language":
		return "en", true
	case "maritalstatus":
		return "M", true
	case "relationship":
		return "C", true
	case "system":
		return "urn:oid:1.2.3.4.5", true
	case "code":
		return "active", true
	case "type":
		return "official", true
	case "priority":
		return "routine", true
	case "intent":
		return "order", true
	case "category":
		return "exam", true
	case "mode":
		return "send", true
	case "direction":
		return "inbound", true
	case "severity":
		return "moderate", true
	case "stage":
		return "1", true
	case "unit":
		return "mg", true
	}
	return "", false
}

func (g *Generator) fakeReference(elem *fhir.ElementDefinition) map[string]any {
	// Derive the resource type from the target profile URL (last path
	// segment, version stripped), producing a realistic "ResourceType/id".
	resourceType := "Organization"
	if len(elem.Types) > 0 && len(elem.Types[0].TargetProfile) > 0 {
		rt := resourceTypeFromURL(elem.Types[0].TargetProfile[0])
		if rt != "Resource" && rt != "DomainResource" {
			resourceType = rt
		}
	}
	return map[string]any{
		"reference": resourceType + "/" + g.fakeID(),
	}
}

// resourceTypeFromURL extracts the resource type name from a canonical
// StructureDefinition URL, stripping any "|version" suffix.
func resourceTypeFromURL(url string) string {
	if i := strings.LastIndex(url, "|"); i >= 0 {
		url = url[:i]
	}
	if i := strings.LastIndex(url, "/"); i >= 0 {
		return url[i+1:]
	}
	return url
}

func (g *Generator) fakeHumanName() map[string]any {
	return map[string]any{
		"use":    "official",
		"family": g.fakeFamilyName(),
		"given":  []any{g.fakeGivenName()},
	}
}

func (g *Generator) fakeAddress() map[string]any {
	return map[string]any{
		"use":        "home",
		"line":       []any{"123 Main St"},
		"city":       "Springfield",
		"state":      "NSW",
		"postalCode": "2000",
		"country":    "AU",
	}
}

func (g *Generator) fakeIdentifier() map[string]any {
	return map[string]any{
		"system": "urn:oid:1.2.3.4.5",
		"value":  g.fakeID(),
	}
}

func (g *Generator) fakeContactPoint() map[string]any {
	return map[string]any{
		"system": "phone",
		"value":  g.fakePhone(),
		"use":    "home",
	}
}

func (g *Generator) fakeCoding(elem *fhir.ElementDefinition, tree *fhir.ElementTree) map[string]any {
	system, code, display := g.fakeBoundCode(elem, tree)
	if display == "" && g.codingDisplayResolver != nil {
		if d, ok := g.codingDisplayResolver.ResolveDisplay(system, code); ok {
			display = d
		}
	}
	coding := map[string]any{
		"system": system,
		"code":   code,
	}
	if display != "" {
		coding["display"] = display
	}
	return coding
}

func (g *Generator) fakeCodeableConcept(elem *fhir.ElementDefinition, tree *fhir.ElementTree) map[string]any {
	return map[string]any{
		"coding": []any{g.fakeCoding(elem, tree)},
		"text":   g.fakeSentence(),
	}
}

// fakeBoundCode returns a plausible (system, code, display) triple for a coded
// element, using the element's value set binding when known. It consults the
// configured BindingResolver first, then falls back to the built-in heuristics.
// For unknown bindings it falls back to the binding's ValueSet URL as the
// system and a synthesized code.
func (g *Generator) fakeBoundCode(elem *fhir.ElementDefinition, tree *fhir.ElementTree) (string, string, string) {
	if elem.Binding != nil && elem.Binding.ValueSet != "" {
		if g.bindingResolver != nil {
			if rc, ok := g.bindingResolver.ResolveBinding(elem, tree); ok {
				return rc.System, rc.Code, rc.Display
			}
		}
		if sys, code, ok := knownBinding(elem.Binding.ValueSet); ok {
			return sys, code, ""
		}
		// Unknown binding: use the ValueSet URL as the system.
		return elem.Binding.ValueSet, g.fakeStringFor(elem, tree), ""
	}
	return "http://example.org/codes", g.fakeStringFor(elem, tree), ""
}

// knownBinding maps a value set URL to a plausible (system, code) pair for
// common FHIR and AU value sets. This is a small curated table; it does not
// resolve value sets from a server.
func knownBinding(valueSet string) (string, string, bool) {
	switch {
	case strings.Contains(valueSet, "administrative-gender"):
		return "http://hl7.org/fhir/administrative-gender", "male", true
	case strings.Contains(valueSet, "address-use"):
		return "http://hl7.org/fhir/address-use", "home", true
	case strings.Contains(valueSet, "address-type"):
		return "http://hl7.org/fhir/address-type", "physical", true
	case strings.Contains(valueSet, "identifier-use"):
		return "http://hl7.org/fhir/identifier-use", "official", true
	case strings.Contains(valueSet, "identifier-type"):
		return "http://terminology.hl7.org/CodeSystem/v2-0203", "NI", true
	case strings.Contains(valueSet, "contact-point-system"):
		return "http://hl7.org/fhir/contact-point-system", "phone", true
	case strings.Contains(valueSet, "contact-point-use"):
		return "http://hl7.org/fhir/contact-point-use", "home", true
	case strings.Contains(valueSet, "name-use"):
		return "http://hl7.org/fhir/name-use", "official", true
	case strings.Contains(valueSet, "languages"):
		return "urn:ietf:bcp:47", "en", true
	case strings.Contains(valueSet, "marital-status"):
		return "http://terminology.hl7.org/CodeSystem/v3-MaritalStatus", "M", true
	case strings.Contains(valueSet, "australian-states-territories"):
		return "https://healthterminologies.gov.au/fhir/CodeSystem/australian-states-territories-2", "NSW", true
	case strings.Contains(valueSet, "healthcare-organisation-role-type"):
		return "https://healthterminologies.gov.au/fhir/CodeSystem/healthcare-organisation-role-type-1", "prov", true
	}
	return "", "", false
}

func (g *Generator) fakeExtension(elem *fhir.ElementDefinition, tree *fhir.ElementTree) map[string]any {
	url := "http://example.org/fhir/StructureDefinition/extension"
	if len(elem.Types) > 0 && len(elem.Types[0].Profiles) > 0 {
		url = elem.Types[0].Profiles[0]
	}
	// A valid extension carries a url and a value[x]. When the profile is
	// unknown we still emit a value so the extension is structurally valid.
	return map[string]any{
		"url":         url,
		"valueString": g.fakeStringFor(elem, tree),
	}
}

func (g *Generator) fakeNarrative() map[string]any {
	return map[string]any{
		"status": "generated",
		"div":    "<div xmlns=\"http://www.w3.org/1999/xhtml\">" + g.fakeSentence() + "</div>",
	}
}

func (g *Generator) fakeMeta() map[string]any {
	return map[string]any{
		"versionId":   "1",
		"lastUpdated": g.fakeDateTime(),
	}
}

func (g *Generator) fakeQuantity() map[string]any {
	return map[string]any{
		"value": g.fakeDecimal(),
		"unit":  "mg",
	}
}

func (g *Generator) fakePeriod() map[string]any {
	return map[string]any{
		"start": g.fakeDateTime(),
		"end":   g.fakeDateTime(),
	}
}

func (g *Generator) fakeRange() map[string]any {
	return map[string]any{
		"low":  map[string]any{"value": g.fakeDecimal()},
		"high": map[string]any{"value": g.fakeDecimal()},
	}
}

func (g *Generator) fakeRatio() map[string]any {
	return map[string]any{
		"numerator":   map[string]any{"value": g.fakeDecimal()},
		"denominator": map[string]any{"value": g.fakeDecimal()},
	}
}

func (g *Generator) fakeAttachment() map[string]any {
	return map[string]any{
		"contentType": "text/plain",
		"data":        "aGVsbG8=",
	}
}

func (g *Generator) fakeAnnotation() map[string]any {
	return map[string]any{
		"text": g.fakeSentence(),
	}
}

func (g *Generator) fakeTiming() map[string]any {
	return map[string]any{
		"repeat": map[string]any{"frequency": 1, "period": 1, "periodUnit": "d"},
	}
}

func (g *Generator) fakeSignature() map[string]any {
	return map[string]any{
		"type": []any{map[string]any{"system": "urn:iso-astm:E1762-95:2013", "code": "1.2.840.10065.1.12.1.1"}},
		"when": g.fakeDateTime(),
		"who":  map[string]any{"reference": "Patient/" + g.fakeID()},
	}
}

func (g *Generator) fakeDosage() map[string]any {
	return map[string]any{
		"text": g.fakeSentence(),
	}
}

func lastSegment(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[i+1:]
	}
	return path
}

// capitalize uppercases the first rune of s.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
