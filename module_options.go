package gogh

// ModuleOption module option
type ModuleOption[T Importer] func(_ hiddenType, m *Module[T])

// WithAliasCorrector sets alias corrector for all GoRenderers to be created
func WithAliasCorrector[T Importer](corrector AliasCorrector) ModuleOption[T] {
	return func(_ hiddenType, m *Module[T]) {
		m.aliasCorrector = corrector
	}
}
