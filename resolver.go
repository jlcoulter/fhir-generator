package fhirgen

import fhir "github.com/jlcoulter/fhir-registry"

// ResolvedCoding is a coding resolved from a value set binding by a
// BindingResolver. It carries the system, code, and optional display of a
// single concept.
type ResolvedCoding struct {
	System  string
	Code    string
	Display string
}

// BindingResolver resolves a value set binding to a concrete coding. It is
// consulted for coded elements (code, Coding, CodeableConcept) before the
// generator falls back to its built-in heuristics. Implementations may query a
// terminology server, expand a value set, look up a code system, or search
// example instances. The element and its tree are passed so implementations can
// use the element's path, binding, type profiles, and sibling/child elements for
// resolution.
type BindingResolver interface {
	// ResolveBinding returns the coding for the given element's value set
	// binding. The bool reports whether a coding was resolved; when false, the
	// generator falls back to its own heuristics.
	ResolveBinding(elem *fhir.ElementDefinition, tree *fhir.ElementTree) (ResolvedCoding, bool)
}

// CodingDisplayResolver resolves the canonical display for a coding's
// (system, code) pair. It is consulted when generating a Coding or
// CodeableConcept to populate the display field. Implementations may look up a
// code system definition or query a terminology server.
type CodingDisplayResolver interface {
	// ResolveDisplay returns the display for the given system and code. The
	// bool reports whether a display was resolved; when false, the generator
	// omits the display field.
	ResolveDisplay(system, code string) (string, bool)
}
