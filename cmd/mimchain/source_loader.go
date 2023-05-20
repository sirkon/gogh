package main

import (
	"go/token"
	"sync"

	"github.com/sirkon/errors"
	"github.com/sirkon/jsonexec"
	"golang.org/x/tools/go/packages"
)

func newSouceLoader(fset *token.FileSet) *souceLoader {
	return &souceLoader{
		fset:     fset,
		pkgCache: map[string]*packages.Package{},
	}
}

type souceLoader struct {
	fset *token.FileSet
	lock sync.Mutex

	pkgCache map[string]*packages.Package
}

func (l *souceLoader) loadPkg(pkgpath string) (pkg *packages.Package, err error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if v, ok := l.pkgCache[pkgpath]; ok {
		return v, nil
	}

	defer func() {
		if err != nil {
			return
		}

		l.pkgCache[pkgpath] = pkg
	}()

	var dst struct {
		ImportPath string
	}
	if err := jsonexec.Run(&dst, "go", "list", "--json", pkgpath); err != nil {
		return nil, errors.Wrap(err, "get package meta info").Str("pkg-path", pkgpath)
	}

	defer func() {
		if err == nil {
			return
		}

		err = errors.Just(err).Str("package-path", dst.ImportPath)
	}()

	mode := packages.NeedImports | packages.NeedTypes | packages.NeedName |
		packages.NeedDeps | packages.NeedSyntax | packages.NeedFiles | packages.NeedModule

	pkgs, err := packages.Load(
		&packages.Config{
			Mode:  mode,
			Fset:  l.fset,
			Tests: false,
		},
		dst.ImportPath,
	)
	if err != nil {
		return nil, errors.Wrap(err, "load package source")
	}

	for _, pkg := range pkgs {
		if pkg.PkgPath == dst.ImportPath {
			return pkg, nil
		}
	}

	return nil, errors.New("failed to load package source")
}
