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
//
// A Generator is safe for concurrent use: it guards its random source with an
// internal mutex. It is configured at construction time via New and the
// With* options, and is not meant to be mutated afterwards.
type Generator struct {
	reg *fhir.Registry

	mu   sync.Mutex
	rng  *rand.Rand
	seed int64

	locale string
	fill   fillMode

	// fillProb is the per-optional-element inclusion probability used when
	// fill == fillProbability.
	fillProb float64

	// id, when non-empty, is injected as the resource's "id" key.
	id string

	// metaProfiles, when non-empty, is injected as the resource's
	// "meta.profile" array.
	metaProfiles []string

	// bindingResolver, when non-nil, resolves value set bindings to codings.
	bindingResolver BindingResolver

	// codingDisplayResolver, when non-nil, resolves coding displays.
	codingDisplayResolver CodingDisplayResolver

	// normalizer, when non-nil, is applied to the generated resource after
	// filling and value injection, mutating it in place.
	normalizer func(map[string]any)

	// stripEmptyExtensions, when true, removes extension/modifierExtension
	// entries with neither a value[x] nor a nested extension array (ext-1).
	stripEmptyExtensions bool

	// values maps a FHIR element path (relative to the resource root, e.g.
	// "name.family") to a caller-supplied value that overrides fake data.
	values map[string]any
}

// fillMode controls how optional elements are handled during generation.
type fillMode int

const (
	fillMinimal     fillMode = iota // only required (min > 0) elements
	fillFull                        // all elements
	fillProbability                 // each optional element with probability fillProb
)

// New returns a generator backed by the given registry. By default the
// generator uses minimal fill mode (only required elements), a random seed, and
// the "en" locale; options are applied in order to override any of these.
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
