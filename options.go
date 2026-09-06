package fhirgen

// Option configures a Generator. Options are applied in the order they are
// passed to New, and later options override earlier ones.
type Option func(*Generator)

// WithSeed sets the random seed for deterministic output. Two generators with
// the same seed produce identical resources.
func WithSeed(seed int64) Option {
	return func(g *Generator) {
		g.seed = seed
	}
}

// WithLocale sets the locale used for fake data. Defaults to "en".
func WithLocale(locale string) Option {
	return func(g *Generator) {
		g.locale = locale
	}
}

// WithMinFillMode generates only required (min > 0) elements. This is the
// default.
func WithMinFillMode() Option {
	return func(g *Generator) {
		g.fill = fillMinimal
	}
}

// WithFullFillMode generates all elements, including optional ones.
func WithFullFillMode() Option {
	return func(g *Generator) {
		g.fill = fillFull
	}
}

// WithProbabilityFillMode generates each optional (min == 0) element with the
// given probability in [0, 1]. Required elements are always generated. A
// probability of 1.0 behaves like full fill mode; 0.0 like minimal fill mode.
func WithProbabilityFillMode(prob float64) Option {
	return func(g *Generator) {
		g.fill = fillProbability
		g.fillProb = prob
	}
}

// WithID sets the logical id of the generated resource. When set, the
// generated resource carries an "id" key with this value.
func WithID(id string) Option {
	return func(g *Generator) {
		g.id = id
	}
}

// WithMetaProfiles sets the meta.profile array of the generated resource. When
// non-empty, the generated resource carries a "meta" object with a "profile"
// array listing the given canonical URLs.
func WithMetaProfiles(profiles []string) Option {
	return func(g *Generator) {
		g.metaProfiles = profiles
	}
}

// WithBindingResolver sets the resolver used to turn value set bindings into
// concrete codings for coded elements. When nil (the default), the generator
// uses its built-in heuristics.
func WithBindingResolver(r BindingResolver) Option {
	return func(g *Generator) {
		g.bindingResolver = r
	}
}

// WithCodingDisplayResolver sets the resolver used to populate the display
// field of generated codings. When nil (the default), the generator omits the
// display field.
func WithCodingDisplayResolver(r CodingDisplayResolver) Option {
	return func(g *Generator) {
		g.codingDisplayResolver = r
	}
}

// Withnormaliser sets a function applied to the generated resource after
// filling and value injection, mutating it in place. It is the extension point
// for domain-specific post-processing (e.g. normalizing identifiers, resolving
// displays, resource-specific patches). When nil (the default), no
// post-processing is applied.
func WithNormaliser(n func(map[string]any)) Option {
	return func(g *Generator) {
		g.normaliser = n
	}
}

// WithStripEmptyExtensions enables removal of extension and modifierExtension
// entries that have neither a value[x] nor a nested extension array. Such
// extensions violate the FHIR ext-1 invariant and are rejected by servers.
func WithStripEmptyExtensions() Option {
	return func(g *Generator) {
		g.stripEmptyExtensions = true
	}
}

// WithValues supplies caller-specified values keyed by FHIR element path
// (relative to the resource root, e.g. "name.family", "birthDate"). These
// values override the fake-generated data for matching elements; all other
// elements are still generated as usual. A value may be a scalar, a
// map[string]any for an object, or a []any for a repeating element; a non-slice
// value for a repeating element is wrapped in an array automatically.
//
// Paths are validated eagerly on each Generate or GenerateForURL call against
// the registry's element tree, so an unknown path returns ErrInvalidPath rather
// than silently producing a resource missing the intended data.
func WithValues(values map[string]any) Option {
	return func(g *Generator) {
		g.values = values
	}
}
