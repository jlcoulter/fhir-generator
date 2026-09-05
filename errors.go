package fhirgen

import "errors"

// Sentinel errors returned by the library.
var (
	// ErrDefinitionNotFound is returned when a structure definition cannot be
	// found for a type name or canonical URL.
	ErrDefinitionNotFound = errors.New("fhirgen: structure definition not found")

	// ErrUnsupportedType is returned when an element's type cannot be resolved
	// and the generator is configured to fail rather than skip it.
	ErrUnsupportedType = errors.New("fhirgen: unsupported element type")
)
