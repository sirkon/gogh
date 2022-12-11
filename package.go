package gogh

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"

	"github.com/sirkon/errors"
	"github.com/sirkon/gogh/internal/blocks"
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

// Go creates new or reuse existing Go source file renderer, options may alter code generation.
func (p *Package[T]) Go(name string, opts ...RendererOption) *GoRenderer[T] {
	if v, ok := p.rs[name]; ok {
		return v
	}

	res := &GoRenderer[T]{
		name:    name,
		pkg:     p,
		options: opts,
		vals:    map[string]any{},
		blocks:  blocks.New(),
		uniqs:   map[string]struct{}{},
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
			res.uniqs[value] = struct{}{}

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

// Reuse creates a renderer over existing file if it exists.
// Works as Go without options otherwise.
func (p *Package[T]) Reuse(name string) (*GoRenderer[T], error) {
	fpath := filepath.Join(p.mod.root, p.rel, name)
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return p.Go(name), nil
		}

		return nil, errors.Wrap(err, "check if the file exists")
	}

	data, err := os.ReadFile(fpath)
	if err != nil {
		return nil, errors.Wrap(err, "read existing file")
	}

	var fset token.FileSet
	file, err := parser.ParseFile(&fset, fpath, data, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return nil, errors.Wrap(err, "parse existing file")
	}

	r := p.Go(name)
	r.reuse = true
	r.preImport = map[string]struct{}{}
	r.reuseFirstImportPos = -1

	for _, decl := range file.Decls {
		imp, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		if imp.Tok != token.IMPORT {
			continue
		}

		pos := fset.Position(decl.Pos()).Line - 1
		if r.reuseFirstImportPos > pos || r.reuseFirstImportPos < 0 {
			r.reuseFirstImportPos = pos
		}
	}

	r.last().Write(data)
	r.newline()

	return r, nil
}

// Raw creates new or reuse existing plain text file renderer.
func (p *Package[T]) Raw(name string, opts ...RendererOption) *RawRenderer {
	return p.mod.Raw(path.Join(p.rel, name))
}

// Path returns package path
func (p *Package[T]) Path() string {
	return path.Join(p.mod.name, p.rel)
}
