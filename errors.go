package fhirgen

import "errors"

// Sentinel errors returned by the library.
var (
	// ErrDefinitionNotFound is returned when a structure definition cannot be
	// found for a type name or canonical URL. It is wrapped with the offending
	// type name or URL and matched with errors.Is.
	ErrDefinitionNotFound = errors.New("fhirgen: structure definition not found")

	// ErrUnsupportedType is returned when an element's type cannot be resolved
	// to a known FHIR type or a complex type in the registry, and the generator
	// is configured to fail rather than skip it.
	ErrUnsupportedType = errors.New("fhirgen: unsupported element type")

	// ErrInvalidPath is returned when a WithValues path does not resolve
	// against the registry's element tree. It is wrapped with the offending
	// path and matched with errors.Is.
	ErrInvalidPath = errors.New("fhirgen: invalid element path")
)
