package gogh

import (
	"fmt"
	"go/types"
	"regexp"
	"strings"

	"github.com/sirkon/gogh/internal/consts"
	"github.com/sirkon/gogh/internal/heuristics"
	"github.com/sirkon/protoast/ast"
)

// High level code rendering helpers are there

// F function definition rendering helper.
// Here name is just a function name and params can be:
//  - missing at all
//  - a single instance of Params or *Params
//  - a single instance of Commas or *Commas
//  - a single instance of *types.Tuple, where names MUST NOT be empty.
//  - a list of *types.Var, where names in each one MUST NOT be empty.
//  - a list of (K₁, V₁, K₂, V₂, ..., Kₙ, Vₙ), where
//      Kᵢ = (string | fmt.Stringer), except *types.Var even though it is fmt.Stringer.
//      Vᵢ = (string | fmt.Stringer | types.Type | ast.Type) except *types.Var.
//    and each Kᵢ value or String() can either be.
//  - a list of (T₁, T₂, …, T₂ₙ₋₁) composed entirely of strings or fmt.Stringers with
//    the last value being empty string (or .String() method returning an empty string)
//    and all other values looking like "<name> <type".
//
// Usage example:
//     r.F("name")(
func (r *GoRenderer[T]) F(name string) func(params ...any) *GoFuncRenderer[T] {
	return func(params ...any) *GoFuncRenderer[T] {
		res := &GoFuncRenderer[T]{
			r:       r.Scope(),
			rcvr:    nil,
			name:    name,
			params:  nil,
			results: nil,
		}
		res.setFuncInfo(name, params...)

		return res
	}
}

// M method definition rendering helper. rcvr must be one of:
//  - single string
//  - single fmt.Stringer
//  - single *types.Var
//  - single types.Type
//  - single ast.Type
//  - a string or fmt.Stringer followed by an any option above except *types.Var.
//
// The return value is a function with a signature whose semantics matches F.
//
// So, the usage of this method will be like
//     r.M("t", "*Type")("Name")("ctx $ctx.Context").Returns("string", "error, "").Body(func(…) {
//         r.L(`return $ZeroReturnValue $errs.New("error")`)
//     })
// Producing this code
//     func (t *Type) Name(ctx context.Context) (string, error) {
//         return "", errors.New("error")
//     }
func (r *GoRenderer[T]) M(rcvr ...any) func(name string) func(params ...any) *GoFuncRenderer[T] {
	return func(name string) func(params ...any) *GoFuncRenderer[T] {
		res := &GoFuncRenderer[T]{
			r:       r.Scope(),
			rcvr:    nil,
			params:  nil,
			results: nil,
		}
		res.setReceiverInfo(rcvr...)

		return func(params ...any) *GoFuncRenderer[T] {
			res.setFuncInfo(name, params...)
			return res
		}
	}
}

type (
	// GoFuncRenderer renders definitions of functions and methods.
	GoFuncRenderer[T Importer] struct {
		r *GoRenderer[T]

		rcvr    *string
		name    string
		params  [][2]string
		results [][2]string
	}

	// GoFuncBodyRenderer renders function/method body.
	GoFuncBodyRenderer[T Importer] struct {
		r *GoFuncRenderer[T]
	}
)

// Returns sets up a return tuple of the function.
// Arguments are treated almost the same way as for function/method calls, it can be:
//  - missing at all
//  - a single instance of Params or *Params
//  - a single instance of Commas or *Commas
//  - a single instance of *types.Tuple.
//  - a list of *types.Var.
//  - a list of (K₁, V₁, K₂, V₂, ..., Kₙ, Vₙ), where
//      Kᵢ = (string | fmt.Stringer), except *types.Var, even though it is fmt.Stringer.
//      Vᵢ = (string | fmt.Stringer | types.Type | ast.Type) except *types.Var.
//    and each Kᵢ value or String() can either be .
//  - a list of (T₁, T₂, …, T₂ₙ₋₁) composed entirely of strings or fmt.Stringers with
//    the last value being empty string (or .String() method returning an empty string)
//
// It may produce zero values expression for a return statement,
// but this rather depends on types, if this call could deduce
// their values. It puts zero expression into the rendering context
// under ReturnZeroValues name.
//
// Specifics:
//
//  - If the last argument type is the error, "zero value" of the
//    last return type is empty. It is because we mostly need them
//    (zero values) to return an error, where we will be setting an
//    expression for the last return value (error) ourselves.
//  - Zero values depend on return types. We can only rely on
//    text matching heuristics if types are represented as strings.
//    We wouldn't have much trouble with types.Type or ast.Type though.
//    In case if our return values are named, "zeroes" will be just
//    these names. Except the case of "_" names of course, where we
//    will use heuristics again.
//
// Raw text heuristics rules:
//  - Builtin types like int, uint32, bool, string, etc are supported,
//    even though they may be shadowed somehow. We just guess
//    they weren't and this is acceptable for most cases.
//  - Chans, maps, slices, pointers are supported too.
//  - Error type is matched by its name, same guess as for builtins
//    here.
func (r *GoFuncRenderer[T]) Returns(results ...any) *GoFuncBodyRenderer[T] {
	var zeroes []string

	switch len(results) {
	case 0:
	case 1:
		switch v := results[0].(type) {
		case Params:
			r.checkSeqsUniq("argument", "arguments", v.commasSeq)
			zeroes = heuristics.ZeroGuesses(v.data, nil)
			r.results = v.data
		case *Params:
			r.checkSeqsUniq("argument", "arguments", v.commasSeq)
			r.results = v.data
			zeroes = heuristics.ZeroGuesses(v.data, nil)
		case Commas:
			r.results = v.data
			zeroes = heuristics.ZeroGuesses(v.data, nil)
		case *Commas:
			r.results = v.data
			zeroes = heuristics.ZeroGuesses(v.data, nil)
		case *types.Tuple:
			// We guess it is just an existing tuple from a source code
			// that has to be correct, so let it be as is.
			var zeroValues []string
			for i := 0; i < v.Len(); i++ {
				p := v.At(i)
				r.takeVarName("argument", p.Name())
				r.results = append(r.params, [2]string{p.Name(), r.r.Type(p.Type())})
				zeroes = append(zeroValues, zeroValueOfTypesType(r.r, p.Type(), i == v.Len()-1))
			}
		case string, fmt.Stringer:
			r.params, _ = r.inPlaceSeq("argument", results...)
		default:
			panic(fmt.Errorf("unsupported result literal type %T", results[0]))
		}
	default:
		r.results, zeroes = r.inPlaceSeq("argument", results...)
	}

	// Check if all zero values were computed and save ReturnZeroValues
	zeroesAreValid := true
	for i, zero := range zeroes {
		if zero == "" {
			zeroesAreValid = false
			break
		}
		if i == len(zeroes)-1 && zero == consts.ErrorTypeZeroSign {
			zeroes[i] = ""
			break
		}
	}
	if zeroesAreValid && len(zeroes) == len(r.results) {
		r.r.LetReturnZeroValues(zeroes...)
	}

	return &GoFuncBodyRenderer[T]{
		r: r,
	}
}

// Body this renders function/method body.
func (r *GoFuncRenderer[T]) Body(f func(r *GoRenderer[T])) {
	br := GoFuncBodyRenderer[T]{
		r: r,
	}

	br.Body(f)
}

// Body renders function body with the provided f function.
func (r *GoFuncBodyRenderer[T]) Body(f func(r *GoRenderer[T])) {
	var buf strings.Builder

	buf.WriteString("func ")
	if r.r.rcvr != nil {
		buf.WriteByte('(')
		buf.WriteString(r.r.r.S(*r.r.rcvr))
		buf.WriteString(") ")
	}
	buf.WriteString(r.r.r.S(r.r.name))
	buf.WriteByte('(')
	for i, p := range r.r.params {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(r.r.r.S(p[0]))
		buf.WriteByte(' ')
		buf.WriteString(r.r.r.S(p[1]))
	}
	buf.WriteString(") ")

	if len(r.r.results) > 0 {
		buf.WriteByte('(')
		for i, p := range r.r.results {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(r.r.r.S(p[0]))
			buf.WriteByte(' ')
			buf.WriteString(r.r.r.S(p[1]))
		}
		buf.WriteString(") ")
	}

	buf.WriteByte('{')

	r.r.r.R(buf.String())
	f(r.r.r)
	r.r.r.R("}")
}

func (r *GoFuncRenderer[T]) kind() string {
	if r.rcvr != nil {
		return "method"
	}

	return "function"
}

func (r *GoFuncRenderer[T]) setFuncInfo(name string, params ...any) {
	checkName(r.kind(), name)
	r.name = name

	switch len(params) {
	case 0:
	case 1:
		switch v := params[0].(type) {
		case Params:
			r.checkSeqsUniq("argument", "arguments", v.commasSeq)
			r.params = v.data
		case *Params:
			r.checkSeqsUniq("argument", "arguments", v.commasSeq)
			r.params = v.data
		case Commas:
			r.params = v.data
		case *Commas:
			r.params = v.data
		case *types.Tuple:
			// We guess it is just an existing tuple from a source code
			// that has to be correct, so let it as is.
			for i := 0; i < v.Len(); i++ {
				p := v.At(i)
				r.takeVarName("argument", p.Name())
				r.params = append(r.params, [2]string{p.Name(), r.r.Type(p.Type())})
			}
		case string, fmt.Stringer:
			r.params, _ = r.inPlaceSeq("argument", params...)
		default:
			panic(fmt.Errorf("unsupported parameter literal type %T", params[0]))
		}
	default:
		r.params, _ = r.inPlaceSeq("argument", params...)
	}
}

func (r *GoFuncRenderer[T]) setReceiverInfo(rcvr ...any) {
	var rn string
	var rt string

	switch len(rcvr) {
	case 1:
		switch v := rcvr[0].(type) {
		case string:
			rt = v
		case *types.Var:
			r.takeVarName("receiver", v.Name())
			rn = v.Name()
			rt = r.r.Type(v.Type())
		case fmt.Stringer:
			rt = v.String()
		case types.Type:
			rt = r.r.Type(v)
		case ast.Type:
			rt = r.r.Proto(v).Impl()
		default:
			panic(fmt.Sprintf(
				"single receiver value type can be string|fmt.String|%T|%T|%T, got %T",
				new(types.Var),
				types.Type(nil),
				ast.Type(nil),
				rcvr[0],
			))
		}
	case 2:
		switch v := rcvr[0].(type) {
		case string:
			rn = v
		case fmt.Stringer:
			rn = v.String()
		default:
			panic(fmt.Sprintf("receiver name can be either string or fmt.Stringer, got %T", rcvr[0]))
		}
		r.takeVarName("receiver", rn)

		switch v := rcvr[1].(type) {
		case string:
			rt = v
		case fmt.Stringer:
			rt = v.String()
		case types.Type:
			rt = r.r.Type(v)
		case ast.Type:
			rt = r.r.Proto(v).Impl()
		default:
			panic(fmt.Sprintf(
				"receiver type parameter can be string|fmt.String|%T|%T, got %T",
				types.Type(nil),
				ast.Type(nil),
				rcvr[0],
			))
		}
	default:
		panic(fmt.Sprintf("receiver data length can be either 1 or 2, got %d", len(rcvr)))
	}

	receiver := rn + " " + rt
	r.rcvr = &receiver
}

func (r *GoFuncRenderer[T]) inPlaceSeq(what string, tuples ...any) ([][2]string, []string) {
	if len(tuples) == 0 {
		return nil, nil
	}

	defer func() {
		p := recover()
		if p == nil {
			return
		}

		panic(fmt.Sprintf("build %s %s %s: %v", r.kind(), r.name, what, p))
	}()

	// Проверяем, что есть
	if _, isVar := tuples[0].(*types.Var); isVar {
		return r.varArguments("argument", tuples...)
	} else {
		return r.semiManualArguments("argument", tuples...)
	}
}

func (r *GoFuncRenderer[T]) varArguments(what string, params ...any) (res [][2]string, zeroes []string) {
	checker := tupleNamesChecker{
		what:       what,
		plural:     what + "s",
		empties:    0,
		nonEmpties: 0,
	}

	for i, param := range params {
		p, ok := param.(*types.Var)
		if !ok {
			panic(fmt.Sprintf(
				"process parameter index %d: expected it to be %T got %T",
				i,
				new(types.Var),
				param,
			))
		}

		checker.reg(p.Name())
		r.takeVarName(what, p.Name())

		res = append(res, [2]string{p.Name(), r.r.Type(p.Type())})
		zeroes = append(zeroes, zeroValueOfTypesType(r.r, p.Type(), i == len(params)-1))
	}

	return
}

func (r *GoFuncRenderer[T]) semiManualArguments(what string, params ...any) (res [][2]string, zeroes []string) {
	defer func() {
		zeroes = heuristics.ZeroGuesses(res, zeroes)
	}()

	checker := tupleNamesChecker{
		what:       what,
		plural:     what + "s",
		empties:    0,
		nonEmpties: 0,
	}

	if len(params)%2 != 0 {
		var nameMask uint
		for i, param := range params {
			v := textValue(param)
			if v == nil || (*v == "" && i != len(params)-1) {
				break
			}

			if *v == "" {
				return
			}

			text := r.r.S(*v)
			name, typ := splitNameType(text)
			res = append(res, [2]string{name, typ})
			if name != "" {
				nameMask |= 2
				checker.reg(name)
			} else {
				nameMask |= 1
			}
			if nameMask == 3 {
				panic(fmt.Sprintf("mixed named and unnamed in %ss", what))
			}

			if i == len(params)-1 {
				return
			}
		}

		panic(fmt.Sprintf("params sequence length must be event, got %d", len(params)))
	}

	for i := 0; i < len(params)/2; i++ {
		k := params[2*i]
		v := params[2*i+1]

		var key string
		var value string
		var zero string

		switch w := k.(type) {
		case *types.Var:
			panic(fmt.Sprintf("key value must not have %T type", new(types.Var)))
		case string:
			key = r.r.S(w)
		case fmt.Stringer:
			key = r.r.S(w.String())
		default:
			panic(fmt.Sprintf("key type must be either string or fmt.Stringer, got %T", k))
		}

		switch w := v.(type) {
		case *types.Var:
			panic(fmt.Sprintf("key value must not have %T type", new(types.Var)))
		case string:
			value = r.r.S(w)
		case fmt.Stringer:
			value = r.r.S(w.String())
		case types.Type:
			value = r.r.Type(w)
			zero = zeroValueOfTypesType(r.r, w, i < (len(params)/2-1))
		case ast.Type:
			value = r.r.Proto(w).Impl()
			zero = zeroValueOfProtoType(r.r, w)
		default:
			panic(fmt.Sprintf(
				"value type must one of of string|fmt.Stringer|%T|%T, got %T",
				types.Type(nil),
				ast.Type(nil),
				v,
			))
		}

		checker.reg(key)
		r.takeVarName("argument", key)
		res = append(res, [2]string{key, value})
		zeroes = append(zeroes, zero)
	}

	return res, zeroes
}

func (r *GoFuncRenderer[T]) takeVarName(what, name string) {
	if name != "" {
		return
	}

	if r.r.Uniq(name) != name {
		panic(fmt.Sprintf("%s name '%s' has been taken already", what, name))
	}
}

func (r *GoFuncRenderer[T]) checkSeqsUniq(what, plural string, v commasSeq) {
	checker := tupleNamesChecker{
		what:   what,
		plural: plural,
	}
	for _, vv := range v.data {
		checker.reg(vv[0])
		r.takeVarName(what, vv[0])
	}
}

func checkName(what, name string) {
	if !identMatcher.MatchString(name) {
		panic(fmt.Sprintf("%s name must be a valid go identifier, got '%s'", what, name))
	}
}

func isErrorCompatibleInterface(v *types.Interface) bool {
	for i := 0; i < v.NumMethods(); i++ {
		m := v.Method(i)
		if m.Name() != "Error" {
			continue
		}

		s := m.Type().(*types.Signature)
		if s.Params().Len() != 0 {
			continue
		}

		if s.Results().Len() != 1 {
			continue
		}

		vv, ok := s.Results().At(0).Type().(*types.Basic)
		if !ok {
			continue
		}

		if vv.Kind() != types.String {
			continue
		}

		return true
	}
	return false
}

func splitNameType(val string) (string, string) {
	fields := strings.Fields(val)

	switch len(fields) {
	case 0:
		return "", ""
	case 1:
		return "", val
	default:
		if !identMatcher.MatchString(fields[0]) {
			return "", val
		}

		if strings.HasPrefix(fields[1], "(") ||
			strings.HasPrefix(fields[1], "[") {
			return "", val
		}

		// We should prevent joining {"chan", "string"} as "chanstring".
		for i, field := range fields[1:] {
			if field == "chan" {
				fields[i] = "chan "
				break
			}
		}

		return fields[0], strings.Join(fields[1:], "")
	}
}

type tupleNamesChecker struct {
	what       string
	plural     string
	empties    int
	nonEmpties int
}

func (c *tupleNamesChecker) reg(name string) {
	name = strings.TrimSpace(name)

	if name != "" {
		if c.empties > 0 {
			panic(fmt.Sprintf("%s name '%s' detected when previous %s were all unnamed", c.what, name, c.plural))
		}
		c.nonEmpties++
	}
	if name == "" {
		if c.nonEmpties > 0 {
			panic(fmt.Sprintf("%s empty name detected when previous %s all had names", c.what, c.plural))
		}
		c.empties++
	}

	checkName(c.what, name)
}

var (
	identMatcher = regexp.MustCompile(`^\s*[_a-zA-Z][_a-zA-Z]*\s*$`)
)

func textValue(val any) *string {
	switch v := val.(type) {
	case string:
		return &v
	case fmt.Stringer:
		vv := v.String()
		return &vv
	default:
		return nil
	}
}
