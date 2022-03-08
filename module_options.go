package gogh

import "github.com/blang/semver/v4"

// ModuleOption module option
type ModuleOption[T Importer] func(_ hiddenType, m *Module[T])

// WithAliasCorrector sets alias corrector for all GoRenderers to be created
func WithAliasCorrector[T Importer](corrector AliasCorrector) ModuleOption[T] {
	return func(_ hiddenType, m *Module[T]) {
		m.aliasCorrector = corrector
	}
}

// WithFixedDeps sets dependencies whose versions must have a specific version
func WithFixedDeps[T Importer](deps map[string]semver.Version) ModuleOption[T] {
	return func(_ hiddenType, m *Module[T]) {
		m.fixedDeps = make(map[string]semver.Version, len(deps))
		for modpath, version := range deps {
			m.fixedDeps[modpath] = version
		}
	}
}
