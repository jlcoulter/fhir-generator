package fhirgen

import (
	"math/rand"
	"sync"

	fhir "github.com/jlcoulter/fhir-registry"
)

// Generator produces conformant FHIR resource instances from a registry of
// structure definitions. It walks the element tree for a resource type and
// fills in fake data that respects cardinality, choice elements, fixed and
// pattern values, and value set bindings.
type Generator struct {
	reg *fhir.Registry

	mu   sync.Mutex
	rng  *rand.Rand
	seed int64

	locale string
	fill   fillMode

	// values maps a FHIR element path (relative to the resource root, e.g.
	// "name.family") to a caller-supplied value that overrides fake data.
	values map[string]any
}

// fillMode controls how optional elements are handled during generation.
type fillMode int

const (
	fillMinimal fillMode = iota // only required (min > 0) elements
	fillFull                    // all elements
)

// New returns a generator backed by the given registry.
func New(reg *fhir.Registry, opts ...Option) *Generator {
	g := &Generator{
		reg:    reg,
		seed:   rand.Int63(),
		locale: "en",
		fill:   fillMinimal,
	}
	for _, opt := range opts {
		opt(g)
	}
	g.rng = rand.New(rand.NewSource(g.seed))
	return g
}
