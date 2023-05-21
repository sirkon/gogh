package main

import (
	"go/parser"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/sirkon/errors"
	"github.com/sirkon/gogh"
	"github.com/sirkon/jsonexec"
)

type arguments struct {
	Version kong.VersionFlag `help:"Print version and exit." short:"v"`

	StringArgsQuoted bool `help:"Quote string values for parameters having string type in an original method/function." default:"false" short:"q"`

	Type sourcePoint `arg:"" help:"Type with chaining methods. Referenced as <pkgpath>:<typename>."`
	Dst  sourcePoint `arg:"" help:"Codegen wrapper type to generate. Referenced as <pkgpath>:<typename>"`

	PackageName string `help:"Package name for the wrapper type. Will be ignored if the package exists already." short:"p"`
}

// sourcePoint represents a command line argument that looks like
// <path>:<identifier>.
type sourcePoint struct {
	Path string
	ID   string
}

// UnmarshalText to implement encoding.TextUnmarshaler.
func (s *sourcePoint) UnmarshalText(text []byte) error {
	v := string(text)
	parts := strings.Split(v, ":")
	switch len(parts) {
	case 1:
		return errors.Newf("missing ':' in '%s'", v)
	case 2:
	default:
		return errors.Newf("path and identifier separated with ':', got %d separated parts instead", len(parts))
	}

	pkg, err := checkPkg(parts[0])
	if err != nil {
		return errors.Wrap(err, "check path")
	}
	if _, err := parser.ParseExpr(parts[1]); err != nil {
		return errors.Newf("invalid identifier '%s'", parts[1])
	}
	if parts[1] != gogh.Public(parts[1]) {
		return errors.New("do not like the proposed type name").
			Str("proposed", parts[1]).
			Str("would-be-good", gogh.Public(parts[1]))
	}

	s.Path = pkg
	s.ID = parts[1]
	return nil
}

func (s sourcePoint) IsValid() bool {
	return s.Path != "" && s.ID != ""
}

func checkPkg(pkgpath string) (string, error) {
	var dst struct {
		ImportPath string
	}
	if err := jsonexec.Run(&dst, "go", "list", "--json", pkgpath); err != nil {
		return "", errors.Wrap(err, "get package meta info").Str("pkg-path", pkgpath)
	}

	return dst.ImportPath, nil
}
