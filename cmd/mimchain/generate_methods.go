package main

import (
	"go/types"
	"strconv"
)

// generateMethods will groups
func (g *generator) generateMethods(r *goRenderer, methods []*types.Func) {
	for _, gr := range groupFuncs(methods) {
		g.generateMethodsGroup(r.Scope(), gr)
	}
}

func (g *generator) generateMethodsGroup(r *goRenderer, gr []*types.Func) {
	r = r.Scope()

	if len(gr) == 0 {
		return
	}

	sig := gr[0].Type().(*types.Signature)
	variadic := sig.Variadic()

	supp := "method" + strconv.Itoa(sig.Params().Len())
	if sig.Variadic() {
		supp += "variadic"
	}
	methodName := r.Uniq("methodName")
	r.Let("baseMethod", supp)
	r.Let("methodName", methodName)

	for _, m := range gr {
		sig := m.Type().(*types.Signature)
		g.generateMethod(r.Scope(), m, sig)
	}

	r = g.b.Scope()
	r.N()
	r.Imports().Add("fmt").Ref("fmt")
	r.Let("baseMethod", supp)
	r.Let("methodName", methodName)
	r.Let("dst", r.S("$x.b.buf"))
	r.Let("r", r.S("$x.b.r"))
	r.Let("posargs", r.S("$x.b.a"))
	params, args := baseArgs(r, gr)
	r.M("$x", "*$gattr")(supp)(params).Returns("*$gattr").Body(func(r *goRenderer) {
		r.L(`$dst.WriteByte('.')`)
		r.L(`$dst.WriteString($methodName)`)
		r.L(`$dst.WriteByte('(')`)

		for i, arg := range args {
			if variadic && i == len(args)-1 {
				break
			}

			r.N()
			r.L(`// render argument '$0' usage`, arg)
			if i > 0 {
				r.L(`$dst.WriteString(", ")`)
			}
			r.L(`switch v := $0.(type) {`, arg)
			r.L(`case string:`)
			r.L(`    $dst.WriteString($r.S(v, $posargs...))`)
			r.L(`default:`)
			r.L(`    $dst.WriteString($fmt.Sprint($0))`, arg)
			r.L(`}`)
		}

		if variadic {
			arg := args[len(args)-1]
			r.Let("iter", r.Uniq("val"))
			if len(args) != 1 {
				r.Let("i", "_")
			} else {
				r.Let("i", r.Uniq("i"))
			}

			r.N()
			r.L(`// render variadic arguments '$0' usage"`, arg)
			r.L(`for $i, $iter := range $0 {`, arg)
			if len(args) > 1 {
				r.L(`$dst.WriteString(", ")`)
			} else {
				r.L(`if $i > 0 {`)
				r.L(`    $dst.WriteString(", ")`)
				r.L(`}`)
			}

			r.L(`    switch v := $iter.(type) {`)
			r.L(`    case string:`)
			r.L(`        $dst.WriteString($r.S(v, $posargs...))`)
			r.L(`    default:`)
			r.L(`        $dst.WriteString($fmt.Sprint($0))`, arg)
			r.L(`    }`)

			r.L(`}`)
		}

		r.N()
		r.L(`$dst.WriteByte(')')`)
		r.L(`return $x`)
	})
}

func (g *generator) generateMethod(r *goRenderer, m *types.Func, sig *types.Signature) {
	params := methodParams(r, sig)

	r.N()
	r.L(`// $0 call support.`, m.Name())
	r.M("$x", "*$gattr")(m.Name())(params...).Returns("*$gattr").Body(func(r *goRenderer) {
		r.L(`return $x.$baseMethod("$0", $1)`, m.Name(), paramsUsage(r, params, sig.Variadic()))
	})
}
