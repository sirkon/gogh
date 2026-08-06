package gogh

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirkon/errors"
	"golang.org/x/mod/modfile"
)

// moduleResolver resolves package paths to the declared version and locality of
// the module they belong to, based on the parsed go.mod (and go.work, if any)
// of the current module.
//
// It exists to key the package-name cache so that a dependency bump invalidates
// exactly the affected entries, and to handle replaced, workspaced and local
// packages without going through `go list` (which can take seconds): for those,
// the package name is read straight from the on-disk source.
type moduleResolver struct {
	// modules maps a module path to its resolved info.
	modules map[string]moduleInfo
}

type moduleInfo struct {
	version string
	local   bool   // replaced with a local path or covered by a workspace
	dir     string // on-disk module root; set when local
}

// resolveResult is the outcome of resolving a package path against the module
// graph declared in go.mod / go.work.
type resolveResult struct {
	modpath string // matched module path
	version string // declared version (valid when known && !local)
	local   bool
	dir     string // on-disk module root (valid when local)
	known   bool   // whether any declared module owns this package path
}

// newModuleResolver builds a resolver from the given go.mod path and an
// optional go.work path (gowork may be "" or "off").
func newModuleResolver(gomodPath, goworkPath string) (*moduleResolver, error) {
	res := &moduleResolver{modules: map[string]moduleInfo{}}

	if err := res.loadGoMod(gomodPath); err != nil {
		return nil, err
	}

	if goworkPath != "" && goworkPath != "off" {
		if err := res.loadGoWork(goworkPath); err != nil {
			return nil, err
		}
	}

	return res, nil
}

// loadGoMod records require versions and replace directives from go.mod.
func (r *moduleResolver) loadGoMod(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return errors.Wrap(err, "read go.mod")
	}

	f, err := modfile.Parse(path, data, nil)
	if err != nil {
		return errors.Wrap(err, "parse go.mod")
	}

	for _, req := range f.Require {
		r.modules[req.Mod.Path] = moduleInfo{version: req.Mod.Version}
	}

	modDir := filepath.Dir(path)
	for _, rep := range f.Replace {
		r.applyReplace(rep, modDir)
	}

	return nil
}

// loadGoWork applies workspace use directives and replaces from go.work. A
// workspace use makes the referenced module local (its source lives on disk and
// is mutable); its on-disk root is recorded so package names can be read
// directly. go.work replaces override those of go.mod.
func (r *moduleResolver) loadGoWork(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrap(err, "read go.work")
	}

	f, err := modfile.ParseWork(path, data, nil)
	if err != nil {
		return errors.Wrap(err, "parse go.work")
	}

	workdir := filepath.Dir(path)
	for _, use := range f.Use {
		modpath := use.ModulePath
		if modpath == "" {
			useDir := use.Path
			if !filepath.IsAbs(useDir) {
				useDir = filepath.Join(workdir, useDir)
			}
			modpath, err = readModulePath(filepath.Join(useDir, "go.mod"))
			if err != nil {
				return errors.Wrap(err, "read module path of workspace use "+use.Path)
			}
		}

		useDir := use.Path
		if !filepath.IsAbs(useDir) {
			useDir = filepath.Join(workdir, useDir)
		}
		if modpath != "" {
			r.modules[modpath] = moduleInfo{local: true, dir: useDir}
		}
	}

	for _, rep := range f.Replace {
		r.applyReplace(rep, workdir)
	}

	return nil
}

// applyReplace records a single replace directive. A replace with no version
// points at a local directory and is therefore local; its resolved on-disk path
// is stored. A versioned replace overrides the declared version with the
// replacement's.
func (r *moduleResolver) applyReplace(rep *modfile.Replace, baseDir string) {
	if rep.New.Version == "" {
		dir := rep.New.Path
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(baseDir, dir)
		}
		r.modules[rep.Old.Path] = moduleInfo{local: true, dir: dir}
		return
	}

	r.modules[rep.Old.Path] = moduleInfo{version: rep.New.Version}
}

// resolve returns the info of the module owning pkgpath. Ownership is by
// longest-prefix match, so that major-version siblings (e.g. foo and foo/v2)
// resolve correctly. A package absent from go.mod (transitive, not declared)
// yields known=false.
func (r *moduleResolver) resolve(pkgpath string) resolveResult {
	var best string
	for modpath := range r.modules {
		if pkgpath != modpath && !strings.HasPrefix(pkgpath, modpath+"/") {
			continue
		}
		if best == "" || len(modpath) > len(best) {
			best = modpath
		}
	}
	if best == "" {
		return resolveResult{}
	}
	info := r.modules[best]
	return resolveResult{
		modpath: best,
		version: info.version,
		local:   info.local,
		dir:     info.dir,
		known:   true,
	}
}

// meta resolves a package path for caching purposes and returns the package
// name (when it could be obtained without `go list`) and the bolt cache key to
// use:
//
//   - Local packages (replaced/workspaced with a disk dir): the name is read
//     straight from the on-disk source and returned with an empty cacheKey, so
//     the result is always fresh and never bolt-cached.
//   - Versioned dependencies: cacheKey is pkgpath + "\x00" + version, so a
//     bump invalidates exactly this entry.
//   - Packages absent from go.mod (stdlib, transitive): cacheKey is pkgpath
//     alone, mirroring the original behaviour — their names change rarely.
//
// If a local package's name cannot be read from disk, it falls back to the
// pkgpath-keyed path (a single `go list`).
func (r *moduleResolver) meta(pkgpath string) (name string, cacheKey string) {
	res := r.resolve(pkgpath)

	if res.local && res.dir != "" {
		relpath := ""
		if pkgpath != res.modpath {
			relpath = strings.TrimPrefix(pkgpath, res.modpath+"/")
		}
		diskDir := filepath.Join(res.dir, filepath.FromSlash(relpath))
		if n, err := readPackageClauseName(diskDir); err == nil && n != "" {
			return n, ""
		}
		// disk read failed — fall back to `go list`, cached by pkgpath.
		return "", pkgpath
	}

	if !res.known {
		return "", pkgpath
	}

	return "", pkgpath + "\x00" + res.version
}

// readPackageClauseName reads the package name from the .go files (excluding
// tests) in the given directory by parsing only their package clauses. It is
// much cheaper than `go list` since it touches only the filesystem.
func readPackageClauseName(dir string) (string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", errors.Wrap(err, "read package directory")
	}

	var name string
	for _, f := range files {
		fn := f.Name()
		if !strings.HasSuffix(fn, ".go") || strings.HasSuffix(fn, "_test.go") {
			continue
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filepath.Join(dir, fn), nil, parser.PackageClauseOnly)
		if err != nil {
			return "", errors.Wrap(err, "parse "+filepath.Join(dir, fn))
		}

		if name == "" {
			name = file.Name.Name
		} else if name != file.Name.Name {
			return "", errors.New("package name conflict in " + dir)
		}
	}

	return name, nil
}

// readModulePath reads the module path from a go.mod file.
func readModulePath(gomodPath string) (string, error) {
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return "", errors.Wrap(err, "read go.mod")
	}

	f, err := modfile.Parse(gomodPath, data, nil)
	if err != nil {
		return "", errors.Wrap(err, "parse go.mod")
	}

	if f.Module == nil {
		return "", errors.New("missing module statement in " + gomodPath)
	}

	return f.Module.Mod.Path, nil
}
