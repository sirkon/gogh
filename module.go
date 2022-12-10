package gogh

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/sirkon/errors"
	"github.com/sirkon/jsonexec"
	"github.com/sirkon/message"
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
		raws:     map[string]*RawRenderer{},
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
	fixedDeps      map[string]semver.Version

	pkgs map[string]*Package[T]
	raws map[string]*RawRenderer

	pkgcache map[string]string
}

// Root create if needed and returns a package placed right in the project root. The name parameter is rather
// optional and will be replaced if there are existing Go files in the package.
func (m *Module[T]) Root(name string) (*Package[T], error) {
	return m.getPackage(name, "")
}

// Current create a package places in the current directory. The name parameter is rather
// optional and will be replaced if there are existing Go files in the package.
func (m *Module[T]) Current(name string) (*Package[T], error) {
	curdir, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "get current directory")
	}

	rel, err := filepath.Rel(m.root, curdir)
	if err != nil {
		return nil, errors.Wrap(err, "compute relative path of the current directory against the module root")
	}

	res, err := m.getPackage(name, rel)
	if err != nil {
		return nil, errors.Wrapf(err, "get package '%s'", rel)
	}

	return res, nil
}

// Package creates if needed and returns a subpackage of the project root.
// The pkgpath parameter can be relative to the module or to be a full package path,
// including module name as well. This will be handled.
// The name parameter is rather optional and will be replaced if there are existing
// Go files in the package.
func (m *Module[T]) Package(name, pkgpath string) (*Package[T], error) {
	return m.getPackage(name, pkgpath)
}

// PackageName returns package name if it does exist, returns empty string otherwise. props parameter may
func (m *Module[T]) PackageName(pkgpath string) (string, error) {
	if err := validatePackagePath(pkgpath); err != nil {
		return "", errors.Wrap(err, "validate package path")
	}

	// the name can be also full path against the current module, strip module name in this case
	switch {
	case strings.HasPrefix(pkgpath, m.name+"/"):
		pkgpath = pkgpath[len(m.name)+1:]
	case pkgpath == m.name:
		pkgpath = ""
	}

	if v, ok := m.pkgs[pkgpath]; ok {
		return v.name, nil
	}

	// there can be some previous go files in the package directory, check if the name is not different
	files, err := os.ReadDir(filepath.Join(m.root, pkgpath))
	if err != nil {
		if !os.IsNotExist(err) {
			return "", errors.Wrap(err, "look for existing files of the package")
		}
	}

	var pkgname string
	for _, filename := range files {
		if !strings.HasSuffix(filename.Name(), ".go") || strings.HasSuffix(filename.Name(), "_test.go") {
			continue
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(
			fset,
			filepath.Join(m.root, pkgpath, filename.Name()),
			nil,
			parser.PackageClauseOnly,
		)
		if err != nil {
			return "", errors.Wrap(err, "parse "+path.Join(m.name, pkgpath, filename.Name()))
		}

		if pkgname == "" {
			pkgname = file.Name.Name
		} else {
			if pkgname != file.Name.Name {
				return "", errors.New("there is package name conflict in " + path.Join(m.name, pkgpath))
			}
		}
	}

	return pkgname, nil
}

// Raw creates a renderer for plain text file
func (m *Module[T]) Raw(relpath string, opts ...RendererOption) *RawRenderer {
	fullpath := filepath.Join(m.root, relpath)
	localpath := path.Join(m.name, relpath)
	if v, ok := m.raws[relpath]; ok {
		return v
	}

	res := &RawRenderer{
		localname: localpath,
		fullname:  fullpath,
		options:   opts,
		vals:      map[string]any{},
	}
	m.raws[relpath] = res
	return res
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

	for _, r := range m.raws {
		if err := os.MkdirAll(filepath.Dir(r.fullname), 0755); err != nil {
			return errors.Wrap(err, "create a directory for "+r.localname)
		}

		if err := r.render(); err != nil {
			return errors.Wrap(err, "renders "+r.localname)
		}
	}

	return nil
}

func (m *Module[T]) getPackage(name, pkgpath string) (*Package[T], error) {
	if err := validatePackagePath(pkgpath); err != nil {
		return nil, errors.Wrap(err, "validate package path")
	}

	// the name can be also full path against the current module, strip module name in this case
	switch {
	case strings.HasPrefix(pkgpath, m.name+"/"):
		pkgpath = pkgpath[len(m.name)+1:]
	case pkgpath == m.name:
		pkgpath = ""
	}

	prevname, err := m.PackageName(pkgpath)
	if err != nil {
		return nil, errors.Wrap(err, "get the existing name of the package")
	}

	if name == "" {
		name = prevname
	}

	if v, ok := m.pkgs[pkgpath]; ok {
		if v.name != name {
			return nil, errors.Newf("package has already been taken under the different name %s", v.name)
		}
	}

	if err := validatePackageName(name); err != nil {
		return nil, errors.Wrap(err, "validate package name")
	}

	if prevname != "" && prevname != name {
		message.Warningf(
			"package %s already exists under different name %s",
			path.Join(m.name, pkgpath),
			prevname,
		)
		name = prevname
	}

	res := &Package[T]{
		mod:  m,
		rel:  pkgpath,
		name: name,
		rs:   map[string]*GoRenderer[T]{},
	}
	m.pkgs[pkgpath] = res
	return res, nil
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
		return errors.Newf("there must be no . or .. as a path components")
	}
	switch base {
	case ".", "..":
		return errors.New("there must be no . or .. as a path components")
	}

	if dir == "" {
		return nil
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
