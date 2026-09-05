// Package fhirgen generates conformant fake/sample FHIR resource instances
// from structure definitions. It builds on fhir-registry, which loads and
// indexes FHIR packages, and walks each resource's element tree to produce
// map[string]any instances that respect cardinality, choice elements, fixed
// and pattern values, value set bindings, and type resolution.
//
// The typical workflow is to construct a Generator from a fhir-registry
// Registry and then ask it to produce a resource for a type name or a
// StructureDefinition canonical URL:
//
//	reg := fhir.NewRegistry()
//	if err := reg.LoadPackage("package"); err != nil {
//	    log.Fatal(err)
//	}
//
//	g := fhirgen.New(reg, fhirgen.WithSeed(42))
//	patient, err := g.Generate("Patient")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// The returned map is suitable for json.Marshal and includes the resource's
// "resourceType". Generation is deterministic for a given seed and safe for
// concurrent use: a Generator guards its random source with an internal mutex.
//
// Options control which elements are filled and let callers override specific
// fields; see New and the With* functions. Errors are returned as sentinel
// values (ErrDefinitionNotFound, ErrUnsupportedType, ErrInvalidPath) usable
// with errors.Is.
//
// See the project README for extended usage, a full list of options, and
// development instructions.
package fhirgen
