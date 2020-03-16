package gogh

import (
	"golang.org/x/tools/go/packages"
)

// Weighter weights import path. Imports are to be splitted into groups of the same weight. The less the weight the
// higher the group to be placed in the import statement
type Weighter interface {
	Weight(importPath string) int
}

// GenericWeighter constructs a weighter what gives weight 0 to import of C, 1 to standard library imports and 2 for
// others. The constructor may panic and this is intentional as it is supposed to be run at the developer's machine
func GenericWeighter() Weighter {
	return genericWeighter{}
}

// GenericWeighter
type genericWeighter struct{}

// Weight for Weighter implementation
func (g genericWeighter) Weight(importPath string) int {
	if importPath == "C" {
		return 0
	}
	_, ok := stdlibPackages[importPath]
	if ok {
		return 1
	}

	return 2
}

var stdlibPackages map[string]struct{}

func init() {
	stdlibPackages = map[string]struct{}{}
	pkgs, err := packages.Load(nil, "std")
	if err != nil {
		panic(err)
	}

	for _, p := range pkgs {
		stdlibPackages[p.PkgPath] = struct{}{}
	}
}
