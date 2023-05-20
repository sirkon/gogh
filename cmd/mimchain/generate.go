package main

import (
	"bytes"
	"fmt"
	"go/types"
	"sort"

	"github.com/sirkon/gogh"
	"golang.org/x/exp/maps"
)

type goRenderer = gogh.GoRenderer[*gogh.Imports]

type generator struct {
	src sourcePoint
	dst sourcePoint

	typ *types.Named
	v   *goRenderer
	r   *goRenderer
	b   *goRenderer
}

func (g *generator) generate(typ *types.Named, constrs []*types.Func, methods []*types.Func) {
	g.r.Let("gtype", g.dst.ID)
	g.r.Let("gattr", gogh.Public(g.dst.ID, "attr"))
	g.r.Let("orig", g.v.Type(g.typ))
	g.r.Let("x", g.r.Uniq("x"))
	g.b = g.r
	g.r = g.r.Z()

	g.generateType(g.r)
	g.generateConstructors(g.r, constrs)
	g.generateMethods(g.r, methods)
}

func (g *generator) generateType(r *goRenderer) {
	r = r.Scope()

	r.Imports().Add("bytes").Ref("bytes")
	r.L(`// $gtype is a dedicated code renderer for chaining calls of $orig.`)
	r.L(`// The type provides constructor calls for the original type.`)
	r.L(`type $gtype struct{`)
	r.L(`    r *xxxGoRenderer // change this to the real *gogh.GoRenderer[T] you will use`)
	r.L(`    buf *$bytes.Buffer`)
	r.L(`    a []any`)
	r.L(`}`)
	r.N()
	r.L(`// $gattr is a dedicated code renderer for chaining calls of $orig.`)
	r.L(`// The type provides chaining calls of the original type.`)
	r.L(`type $gattr struct{`)
	r.L(`    b *$gtype`)
	r.L(`}`)
	r.N()
	r.L(`func ($x *$gtype) String() string{`)
	r.L(`    return $x.buf.String()`)
	r.L(`}`)
	r.N()
	r.L(`func ($x *$gattr) String() string{`)
	r.L(`    return $x.b.buf.String()`)
	r.L(`}`)
}

func (g *generator) typeRcvr(r *goRenderer) gogh.Params {
	var params gogh.Params
	params.Add(r.S("$x"), r.S("$gtype"))

	return params
}

func (g *generator) attrRcvr(r *goRenderer) gogh.Params {
	var params gogh.Params
	params.Add(r.S("$x"), r.S("$gattr"))

	return params
}

func (g *generator) otype() string {
	return g.v.Type(g.typ)
}

// baseArgNames computes arg names for a base function of
// the given group.
//
//  - If all param[i] names are equal and not empty the arg[i] = param[i].Name.
//  - arg${i} otherwise.
func baseArgs(r *goRenderer, gr []*types.Func) (params gogh.Params, args []string) {
	variadic := isVariadic(gr[0])
	params.Add(r.S("$methodName"), "string")
	for _, f := range gr {
		s := f.Type().(*types.Signature)
		if len(args) == 0 {
			args = make([]string, s.Params().Len())
		}

		for i := 0; i < s.Params().Len(); i++ {
			p := s.Params().At(i)
			switch p.Name() {
			case "":
				args[i] = "*"
			case args[i]:
			default:
				if args[i] == "" {
					args[i] = p.Name()
					break
				}
				args[i] = "*"
			}
		}
	}

	for i, arg := range args {
		if arg != "*" {
			args[i] = r.Uniq(args[i])
		} else {
			args[i] = r.Uniq(fmt.Sprintf("arg%d", i+1))
		}

		argType := "any"
		if variadic && i == len(args)-1 {
			argType = "...any"
		}
		params.Add(args[i], argType)
	}

	return params, args
}

func methodParams(r *goRenderer, f *types.Signature) []any {
	var res []any
	for i := 0; i < f.Params().Len(); i++ {
		p := f.Params().At(i)
		tname := "any"
		if f.Variadic() && i == f.Params().Len()-1 {
			tname = "...any"
		}
		res = append(res, r.Uniq(p.Name()), tname)
	}

	return res
}

func paramsUsage(r *goRenderer, params []any, variadic bool) string {
	var buf bytes.Buffer
	for i := 0; i < len(params); i += 2 {
		if i != 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(params[i].(string))
		if variadic && i == len(params)-2 {
			buf.WriteString("...")
		}
	}

	return buf.String()
}

// groupFuncs groups funcs with the same combined weight of arguments together.
//
// weight of an argument:
//  - 3 for a variadic.
//  - 2 for non-variadic.
//
// This means combined weight will be an odd number if a method has
// variadic arguments, because only the last argument can be variadic.
// And an even number if no variadic args in a method definition.
func groupFuncs(methods []*types.Func) [][]*types.Func {
	groups := map[int][]*types.Func{}

	for _, m := range methods {
		w := weight(m)
		groups[w] = append(groups[w], m)
	}

	keys := maps.Keys(groups)
	sort.Ints(keys)
	var res [][]*types.Func
	for _, funcs := range groups {
		res = append(res, funcs)
	}

	return res
}

func weight(m *types.Func) int {
	s := m.Type().(*types.Signature)
	res := s.Params().Len() * 2

	if s.Variadic() {
		return res - 1
	}
	return res
}

func isVariadic(f *types.Func) bool {
	return f.Type().(*types.Signature).Variadic()
}
