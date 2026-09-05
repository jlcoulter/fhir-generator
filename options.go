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
