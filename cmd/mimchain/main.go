package main

import (
	"go/token"
	"os"
	"runtime/debug"

	"github.com/alecthomas/kong"
	"github.com/sirkon/errors"
	"github.com/sirkon/gogh"
	"github.com/sirkon/message"
)

func main() {
	argParams := os.Args[1:]
	if len(argParams) > 0 && argParams[0] == "--" {
		argParams = argParams[1:]
	}

	var args arguments
	cliParser := kong.Must(
		&args,
		kong.Name(appName),
		kong.Description("Generate codegen wrapper type for the given type with methods chaining."),
		kong.UsageOnError(),
	)

	if _, err := cliParser.Parse(argParams); err != nil {
		message.Warning(errors.Wrap(err, "parse command line arguments"))
		cliParser.FatalIfErrorf(err)
	}

	if args.Version {
		var version string
		info, ok := debug.ReadBuildInfo()
		if !ok {
			version = "(devel)"
		} else {
			version = info.Main.Version
		}

		message.Info(appName, "version", version)
		return
	}

	l := newSouceLoader(token.NewFileSet())
	typ, err := getSourceType(l, args.Type)
	if err != nil {
		message.Fatal(errors.Wrap(err, "get source type info"))
	}

	constrs := getChainableTypeConstructors(typ.Obj().Pkg(), typ)
	methods := getTypePublicChainableMethods(typ)

	prj, err := gogh.New(
		gogh.FancyFmt,
		func(r *gogh.Imports) *gogh.Imports {
			return r
		},
	)
	if err != nil {
		message.Fatal(errors.Wrap(err, "set up rendering project"))
	}

	p, err := prj.Package("", args.Dst.Path)
	if err != nil {
		message.Fatal(errors.Wrap(err, "set up rendering package").Str("pkg-path", args.Dst.Path))
	}

	fileName := gogh.Underscored(args.Dst.ID) + "_generated.go"
	r := p.Go(fileName, gogh.Shy)

	g := &generator{
		src:          args.Type,
		dst:          args.Dst,
		typ:          typ,
		v:            p.Void(),
		r:            r,
		quoteStrings: args.StringArgsQuoted,
	}
	g.generate(typ, constrs, methods)

	if err := prj.Render(); err != nil {
		message.Fatal(errors.Wrap(err, "render generated source code"))
	}
}
