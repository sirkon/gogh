package gogh

import (
	"go/ast"
	"go/parser"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sirkon/errors"
	"github.com/sirkon/jsonexec"
)

// Formatter is a signature of source code formatting function. GoFmt and FancyFmt are provided by this package.
type Formatter func([]byte) ([]byte, error)

// New computes GoFile module data and creates an instance of Module
func New[T Importer](
	formatter Formatter,
	importer func(r *Imports) T,
	opts ...ModuleOption[T],
) (*Module[T], error) {
	var envData struct {
		GOMOD string
	}
	if err := jsonexec.Run(&envData, "go", "env", "--json"); err != nil {
		return nil, errors.Wrap(err, "get module environment data")
	}
	if envData.GOMOD == "" {
		return nil, errors.New("missing go.mod — only module projects are supported")
	}

	var moduleData struct {
		Module struct {
			Path string
		}
	}
	if err := jsonexec.Run(&moduleData, "go", "mod", "edit", "--json"); err != nil {
		return nil, errors.Wrap(err, "get current module info")
	}

	res := &Module[T]{
		name:     moduleData.Module.Path,
		root:     filepath.Dir(envData.GOMOD),
		fmt:      formatter,
		importer: importer,
		pkgs:     map[string]*Package[T]{},
		pkgcache: map[string]string{},
	}

	for _, opt := range opts {
		opt(hiddenType{}, res)
	}

	return res, nil
}

// Module all code generation is done within a module.
type Module[T Importer] struct {
	name           string
	root           string
	fmt            func([]byte) ([]byte, error)
	importer       func(imports *Imports) T
	aliasCorrector AliasCorrector

	pkgs     map[string]*Package[T]
	pkgcache map[string]string
}

// Root create if needed and returns a package placed right in the project root
func (m *Module[T]) Root(name string) (*Package[T], error) {
	return m.getPackage(name, "")
}

// Package creates if needed and returns a subpackage of the project root
func (m *Module[T]) Package(name, pkgpath string) (*Package[T], error) {
	return m.getPackage(name, pkgpath)
}

// Name returns module name
func (m *Module[T]) Name() string {
	return m.name
}

// Render renders generated data
func (m *Module[T]) Render() error {
	for pkgpath, pkg := range m.pkgs {
		for name, r := range pkg.rs {
			fullname := filepath.Join(m.root, pkgpath, name)
			localname := filepath.Join(m.name, pkgpath, name)

			if err := os.MkdirAll(filepath.Dir(fullname), 0755); err != nil {
				return errors.Wrap(err, "create a directory for "+localname)
			}

			if err := r.render(); err != nil {
				return errors.Wrap(err, "renders "+localname)
			}
		}
	}

	return nil
}

func (m *Module[T]) getPackage(name, pkgpath string) (*Package[T], error) {
	if err := validatePackageName(name); err != nil {
		return nil, errors.Wrap(err, "validate package name")
	}

	if err := validatePackagePath(pkgpath); err != nil {
		return nil, errors.Wrap(err, "validate package path")
	}

	if v, ok := m.pkgs[pkgpath]; ok {
		if v.name != name {
			return nil, errors.Newf("package has already been taken under the different name %s", v.name)
		}
	}

	return &Package[T]{
		mod:  m,
		rel:  pkgpath,
		name: name,
	}, nil
}

// validatePackagePath pkgpath must no be absolute nor must not have . or .. as its components
func validatePackagePath(pkgpath string) error {
	if filepath.IsAbs(pkgpath) {
		return errors.New("path must not be absolute")
	}
	if strings.HasSuffix(pkgpath, "/") {
		return errors.New("path must not ends with /")
	}

	dir, base := path.Split(pkgpath)
	switch dir {
	case "./", "../", ".", "..":
		return errors.New("there must be no . or .. as a path components")
	}
	switch base {
	case ".", "..":
		return errors.New("there must be no . or .. as a path components")
	}

	dir = path.Clean(dir)
	if dir != pkgpath {
		return validatePackagePath(dir)
	}

	return nil
}

func validatePackageName(name string) error {
	if name == "" {
		return errors.New("must not be empty")
	}

	expr, err := parser.ParseExpr(name)
	if err != nil {
		return errors.Wrap(err, "parse value")
	}

	if _, ok := expr.(*ast.Ident); !ok {
		return errors.Newf("parsed expression must be %T, got %T", ast.NewIdent(""), expr)
	}

	return nil
}
