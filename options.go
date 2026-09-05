package fhirgen

// Option configures a Generator.
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
