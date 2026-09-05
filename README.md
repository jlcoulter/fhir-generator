# fhir-generator

A Go library that generates conformant fake/sample FHIR resource instances
from structure definitions. It builds on
[`fhir-registry`](https://github.com/jlcoulter/fhir-registry), which loads and
indexes FHIR packages, and walks each resource's element tree to produce
`map[string]any` instances that respect cardinality, choice elements, fixed and
pattern values, value set bindings, and type resolution.

## Features

- **Conformant output** — generated resources pass the registry's own
  `Marshal` normalizer without cardinality violations.
- **Cardinality-aware** — required (`min > 0`) elements are always filled;
  repeating (`max > 1`) elements become arrays; choice (`[x]`) elements use
  their concrete type-suffixed key.
- **Value-aware** — `fixed`/`pattern` values are emitted verbatim; coded
  elements synthesize plausible codes from their path.
- **Type resolution** — complex types (Identifier, Address, HumanName, ...) are
  resolved through the registry, and nested profile types are honored.
- **Deterministic** — `WithSeed` produces identical output for a given seed.
- **Recursion-safe** — self-referential types (e.g. `Extension.extension`) are
  depth-limited so generation cannot hang.
- **Concurrency-safe** — fake-data generation is guarded by a mutex.

## Usage

```go
import (
    "encoding/json"
    "log"

    fhir "github.com/jlcoulter/fhir-registry"
    "github.com/jlcoulter/fhir-generator"
)

reg := fhir.NewRegistry()
if err := reg.LoadPackage("package"); err != nil {
    log.Fatal(err)
}

g := fhirgen.New(reg, fhirgen.WithSeed(42))

// Minimal (default): only required elements.
patient, err := g.Generate("Patient")

// Full: fill every element.
g2 := fhirgen.New(reg, fhirgen.WithFullFillMode())
org, err := g2.Generate("Organization")

// Generate for a specific canonical URL.
auPatient, err := g2.GenerateForURL("http://hl7.org.au/fhir/StructureDefinition/au-patient")

b, _ := json.MarshalIndent(patient, "", "  ")
```

## Options

- `WithSeed(int64)` — set the random seed for deterministic output.
- `WithLocale(string)` — locale for fake data (default `"en"`).
- `WithMinFillMode()` — only required elements (default).
- `WithFullFillMode()` — fill all elements, optional ones included.

## API

- `New(reg *fhir.Registry, opts ...Option) *Generator`
- `(*Generator) Generate(typeName string) (map[string]any, error)`
- `(*Generator) GenerateForURL(url string) (map[string]any, error)`

## Errors

Sentinel errors (`ErrDefinitionNotFound`, `ErrUnsupportedType`) are returned
wrapped and usable with `errors.Is`.

## Development

The tests load the sample package `testdata/au-base.tgz` (copied from
`fhir-registry`). Run them with:

```
go test ./...
```

This project is developed with strict test-driven development: every behavior
change is preceded by a failing test.
