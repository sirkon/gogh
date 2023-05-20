package main

import (
	"go/types"
	"strconv"
)

func (g *generator) generateConstructors(r *goRenderer, constrs []*types.Func) {
	for _, gr := range groupFuncs(constrs) {
		g.generateConstructorsGroup(r, gr)
	}
}

func (g *generator) generateConstructorsGroup(r *goRenderer, gr []*types.Func) {
	r = r.Scope()
	sig := gr[0].Type().(*types.Signature)
	variadic := sig.Variadic()

	supp := "constructor" + strconv.Itoa(sig.Params().Len())
	if sig.Variadic() {
		supp += "variadic"
	}
	methodName := r.Uniq("funcName")
	r.Let("baseFunc", supp)
	r.Let("funcName", methodName)

	for _, m := range gr {
		sig := m.Type().(*types.Signature)
		g.generateConstructorCall(r.Scope(), m, sig)
	}

	r = g.b.Scope()
	r.N()
	r.Imports().Add("fmt").Ref("fmt")
	r.Let("baseMethod", supp)
	r.Let("methodName", methodName)
	r.Let("dst", r.S("$x.buf"))
	r.Let("r", r.S("$x.r"))
	r.Let("posargs", r.S("$x.a"))
	params, args := baseArgs(r, gr)
	r.M("$x", "*$gtype")(supp)(params).Returns("*$gattr").Body(func(r *goRenderer) {
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
			r.L(`// render variadic arguments '$0' usage`, arg)
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
		r.L(`return &$gattr{`)
		r.L(`    b: $x,`)
		r.L(`}`)
	})
}

func (g *generator) generateConstructorCall(r *goRenderer, m *types.Func, sig *types.Signature) {
	params := methodParams(r, sig)

	r.N()
	r.Let("renderer", r.Uniq("r"))
	r.L(`// $0 call support.`, m.Name())
	r.M("$x", "*$gtype")(m.Name())(params...).Returns("*$gattr").Body(func(r *goRenderer) {
		r.L(`$renderer := $x.r.Scope()`)
		r.L(`$renderer.Imports().Add("$0").Ref("iy_XIVFZjnaQkfEXOVKVdvOMrPUEXsuq")`, g.typ.Obj().Pkg().Path())
		r.L(`return $x.$baseFunc("$${iy_XIVFZjnaQkfEXOVKVdvOMrPUEXsuq}.$0", $1)`, m.Name(), paramsUsage(r, params, sig.Variadic()))
	})
}
