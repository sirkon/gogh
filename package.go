package gogh

import (
	"fmt"
	"path"
)

// Package represents a package of the current module
type Package[T Importer] struct {
	mod  *Module[T]
	rel  string
	name string

	rs map[string]*GoRenderer[T]
}

// Package creates "subpackage" of the current package
func (p *Package[T]) Package(name, pkgpath string) (*Package[T], error) {
	return p.mod.Package(name, path.Join(p.mod.name, p.rel, pkgpath))
}

// Go creates new or reuse existing Go source file renderingOptionsHandler, options may alter code generation.
func (p *Package[T]) Go(name string, opts ...RendererOption) *GoRenderer[T] {
	if v, ok := p.rs[name]; ok {
		return v
	}

	res := &GoRenderer[T]{
		name:    name,
		pkg:     p,
		options: opts,
		vals:    map[string]interface{}{},
	}

	imports := &Imports{
		pkgs: map[string]string{},
		varcapter: func(name string, value string) string {
			if v, ok := res.vals[name]; ok {
				if vv := fmt.Sprint(v); v != value {
					return vv
				}
			}

			res.vals[name] = value

			return ""
		},
		cached: func(pkgpath string) string {
			return p.mod.pkgcache[pkgpath]
		},
		cacher: func(alias, pkgpath string) {
			p.mod.pkgcache[pkgpath] = alias
		},
		inprocess: func(pkgpath string) string {
			for _, pkg := range p.mod.pkgs {
				if pkg.Path() == pkgpath {
					return pkg.name
				}
			}

			return ""
		},
		namer: func(relpath string) string {
			return path.Join(p.mod.name, relpath)
		},
	}
	res.imports = p.mod.importer(imports)
	p.rs[name] = res

	return res
}

// Path returns package path
func (p *Package[T]) Path() string {
	return path.Join(p.mod.name, p.rel)
}
