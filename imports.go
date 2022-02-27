package gogh

import (
	"math"
	"strconv"

	"github.com/sirkon/errors"
	"github.com/sirkon/jsonexec"
	"github.com/sirkon/message"
)

// Importer an abstraction for Imports extensions
type Importer interface {
	Add(pkgpath string) *ImportAliasControl
	Module(relpath string) *ImportAliasControl
	Imports() *Imports
}

// AliasCorrector this function may alter alias for a package with the given name and path in case of name conflict.
// Empty string means no correction for the given pkgpath.
type AliasCorrector func(name, pkgpath string) string

// Imports a facility to add imports in the GoFile source file
type Imports struct {
	pkgs      map[string]string
	varcapter func(name string, value string) string
	cached    func(pkgpath string) string
	cacher    func(alias, pkgpath string)
	inprocess func(pkgpath string) string
	namer     func(relpath string) string
	pending   []*ImportAliasControl
	corrector AliasCorrector
}

// Imports to satisfy Importer
func (i *Imports) Imports() *Imports {
	return i
}

// Add registers new import if it wasn't before.
func (i *Imports) Add(pkgpath string) *ImportAliasControl {
	if err := validatePackagePath(pkgpath); err != nil {
		panic(errors.Wrapf(err, "validate package path '%s'", pkgpath))
	}

	i.pushImports()

	var alias string
	if v := i.cached(pkgpath); v != "" {
		alias = v
	} else {
		alias = i.getPkgName(pkgpath)
	}

	res := &ImportAliasControl{
		i:       i,
		pkgpath: pkgpath,
		alias:   alias,
	}
	i.pending = append(i.pending, res)
	return res
}

// Module to import a package placed with the current module
func (i *Imports) Module(relpath string) *ImportAliasControl {
	return i.Add(i.namer(relpath))
}

func (i *Imports) getPkgName(pkgpath string) string {
	// this can be a package which is under rendering currently, check it
	if v := i.inprocess(pkgpath); v != "" {
		return v
	}

	// it can be only outer package if we reach here, use go list
	var pkginfo struct {
		Name string
	}
	if err := jsonexec.Run(&pkginfo, "go", "list", "--json", pkgpath); err != nil {
		panic(errors.Wrapf(err, "get package %s info", pkgpath))
	}

	return pkginfo.Name
}

func (i *Imports) pushImports() {
	for _, a := range i.pending {
		message.Debugf("push import %s of %s", a.alias, a.pkgpath)
		a.push()
	}
	i.pending = i.pending[:0]
}

// ImportAliasControl allows to assign an alias for package import
type ImportAliasControl struct {
	i       *Imports
	pkgpath string
	alias   string
}

// As assign given alias for the import. Conflicting one may cause a panic.
func (a *ImportAliasControl) As(alias string) *ImportReferenceControl {
	if err := validatePackageName(alias); err != nil {
		panic(errors.Wrapf(err, "validate alias name '%s'", alias))
	}

	for pkgpath, pkgalias := range a.i.pkgs {
		if pkgalias == alias && pkgpath != a.pkgpath {
			panic(
				errors.Newf("name or alias '%s' has been taken before for package %s", alias, pkgpath),
			)
		}

		if pkgpath == a.pkgpath && pkgalias != a.alias {
			panic(
				errors.Newf(
					"package %s has been imported before with the different name or alias %s != %s",
					pkgpath,
					pkgalias,
					alias,
				),
			)
		}
	}

	a.alias = alias
	return &ImportReferenceControl{
		a: a,
	}
}

// Ref adds a package name or alias into the renderer's context under the given name ref
func (a *ImportAliasControl) Ref(ref string) {
	a.push()

	if prev := a.i.varcapter(ref, a.alias); prev != "" {
		panic(
			errors.Newf(
				"value '%s' has been set before with the different content '%s' != '%s'",
				ref,
				prev,
				a.alias,
			),
		)
	}
}

func (a *ImportAliasControl) push() string {
	if v, ok := a.i.pkgs[a.pkgpath]; ok {
		return v
	}

	defer func() {
		a.i.cacher(a.pkgpath, a.alias)
	}()

	// look for alias conflicts
	var conflict bool
	for pkgpath, pkgalias := range a.i.pkgs {
		if pkgalias == a.alias {
			if pkgpath == a.pkgpath {
				return a.alias
			}

			conflict = true
			break
		}
	}

	// there is the conflict, try alias corrector first
	if !conflict {
		a.i.pkgs[a.pkgpath] = a.alias
		return a.alias
	}

	conflict = false
	var alias string
	if a.i.corrector != nil {
		alias = a.i.corrector(a.alias, a.pkgpath)
		if alias != "" {
			for pkgpath, pkgalias := range a.i.pkgs {
				if pkgalias == alias {
					if pkgpath == a.pkgpath {
						return alias
					}

					conflict = true
					break
				}
			}
		}
	}

	// there's a the conflict even with alias corrector, will look for first free <alias>N
outer:
	for i := 2; i < math.MaxInt; i++ {
		alias = a.alias + strconv.Itoa(i)
		for pkgpath, pkgalias := range a.i.pkgs {
			if pkgalias == alias {
				if pkgpath == a.pkgpath {
					return alias
				}

				continue outer
			}
		}

		// found no conflict if in here
		a.alias = alias
		break
	}

	a.i.pkgs[a.pkgpath] = a.alias
	return a.alias
}

// ImportReferenceControl allows to add a variable having package name (or alias) in the renderer scope
type ImportReferenceControl struct {
	a *ImportAliasControl
}

// Ref adds a package name or alias into the renderering context under the given name ref
func (r *ImportReferenceControl) Ref(ref string) {
	r.a.Ref(ref)
}
