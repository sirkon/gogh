package testexample

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/sirkon/gogh"
)

// Error is a dedicated code renderer for chaining calls of errors.Error.
// The type provides constructor calls for the original type.
type Error[T gogh.Importer] struct {
	r   *gogh.GoRenderer[T]
	buf *bytes.Buffer
	a   []any
}

// ErrorAttr is a dedicated code renderer for chaining calls of errors.Error.
// The type provides chaining calls of the original type.
type ErrorAttr[T gogh.Importer] struct {
	b *Error[T]
}

func (x *Error[T]) String() string {
	return x.buf.String()
}

func (x *ErrorAttr[T]) String() string {
	return x.b.buf.String()
}

// Just call support.
func (x *Error[T]) Just(err any) *ErrorAttr[T] {
	r := x.r.Scope()
	r.Imports().Add("github.com/sirkon/errors").Ref("iy_XIVFZjnaQkfEXOVKVdvOMrPUEXsuq")
	return x.constructor1("${iy_XIVFZjnaQkfEXOVKVdvOMrPUEXsuq}.Just", err)
}

// New call support.
func (x *Error[T]) New(msg any) *ErrorAttr[T] {
	r := x.r.Scope()
	r.Imports().Add("github.com/sirkon/errors").Ref("iy_XIVFZjnaQkfEXOVKVdvOMrPUEXsuq")
	return x.constructor1("${iy_XIVFZjnaQkfEXOVKVdvOMrPUEXsuq}.New", gogh.QuoteBias(msg))
}

// Newf call support.
func (x *Error[T]) Newf(format any, a ...any) *ErrorAttr[T] {
	r := x.r.Scope()
	r.Imports().Add("github.com/sirkon/errors").Ref("iy_XIVFZjnaQkfEXOVKVdvOMrPUEXsuq")
	return x.constructor2variadic("${iy_XIVFZjnaQkfEXOVKVdvOMrPUEXsuq}.Newf", format, a...)
}

// Wrap call support.
func (x *Error[T]) Wrap(err any, msg any) *ErrorAttr[T] {
	r := x.r.Scope()
	r.Imports().Add("github.com/sirkon/errors").Ref("iy_XIVFZjnaQkfEXOVKVdvOMrPUEXsuq")
	return x.constructor2("${iy_XIVFZjnaQkfEXOVKVdvOMrPUEXsuq}.Wrap", err, msg)
}

// Wrapf call support.
func (x *Error[T]) Wrapf(err any, format any, a ...any) *ErrorAttr[T] {
	r := x.r.Scope()
	r.Imports().Add("github.com/sirkon/errors").Ref("iy_XIVFZjnaQkfEXOVKVdvOMrPUEXsuq")
	return x.constructor3variadic("${iy_XIVFZjnaQkfEXOVKVdvOMrPUEXsuq}.Wrapf", err, format, a...)
}

// Pfx call support.
func (x *ErrorAttr[T]) Pfx(prefix any) *ErrorAttr[T] {
	return x.method1("Pfx", prefix)
}

// Loc call support.
func (x *ErrorAttr[T]) Loc(depth any) *ErrorAttr[T] {
	return x.method1("Loc", depth)
}

// Bool call support.
func (x *ErrorAttr[T]) Bool(name any, value any) *ErrorAttr[T] {
	return x.method2("Bool", name, value)
}

// Int call support.
func (x *ErrorAttr[T]) Int(name any, value any) *ErrorAttr[T] {
	return x.method2("Int", name, value)
}

// Int8 call support.
func (x *ErrorAttr[T]) Int8(name any, value any) *ErrorAttr[T] {
	return x.method2("Int8", name, value)
}

// Int16 call support.
func (x *ErrorAttr[T]) Int16(name any, value any) *ErrorAttr[T] {
	return x.method2("Int16", name, value)
}

// Int32 call support.
func (x *ErrorAttr[T]) Int32(name any, value any) *ErrorAttr[T] {
	return x.method2("Int32", name, value)
}

// Int64 call support.
func (x *ErrorAttr[T]) Int64(name any, value any) *ErrorAttr[T] {
	return x.method2("Int64", name, value)
}

// Uint call support.
func (x *ErrorAttr[T]) Uint(name any, value any) *ErrorAttr[T] {
	return x.method2("Uint", name, value)
}

// Uint8 call support.
func (x *ErrorAttr[T]) Uint8(name any, value any) *ErrorAttr[T] {
	return x.method2("Uint8", name, value)
}

// Uint16 call support.
func (x *ErrorAttr[T]) Uint16(name any, value any) *ErrorAttr[T] {
	return x.method2("Uint16", name, value)
}

// Uint32 call support.
func (x *ErrorAttr[T]) Uint32(name any, value any) *ErrorAttr[T] {
	return x.method2("Uint32", name, value)
}

// Uint64 call support.
func (x *ErrorAttr[T]) Uint64(name any, value any) *ErrorAttr[T] {
	return x.method2("Uint64", name, value)
}

// Float32 call support.
func (x *ErrorAttr[T]) Float32(name any, value any) *ErrorAttr[T] {
	return x.method2("Float32", name, value)
}

// Float64 call support.
func (x *ErrorAttr[T]) Float64(name any, value any) *ErrorAttr[T] {
	return x.method2("Float64", name, value)
}

// Str call support.
func (x *ErrorAttr[T]) Str(name any, value any) *ErrorAttr[T] {
	return x.method2("Str", name, value)
}

// Stg call support.
func (x *ErrorAttr[T]) Stg(name any, value any) *ErrorAttr[T] {
	return x.method2("Stg", name, value)
}

// Strs call support.
func (x *ErrorAttr[T]) Strs(name any, value any) *ErrorAttr[T] {
	return x.method2("Strs", name, value)
}

// Type call support.
func (x *ErrorAttr[T]) Type(name any, typ any) *ErrorAttr[T] {
	return x.method2("Type", name, typ)
}

// Any call support.
func (x *ErrorAttr[T]) Any(name any, value any) *ErrorAttr[T] {
	return x.method2("Any", name, value)
}

func (x *Error[T]) constructor1(funcName string, arg1 any) *ErrorAttr[T] {
	x.buf.WriteString(x.r.S(funcName, x.a...))
	x.buf.WriteByte('(')

	// render argument 'arg1' usage
	switch v := arg1.(type) {
	case string:
		x.buf.WriteString(x.r.S(v, x.a...))
	case fmt.Stringer:
		x.buf.WriteString(x.r.S(v.String(), x.a...))
	default:
		x.buf.WriteString(fmt.Sprint(arg1))
	}

	x.buf.WriteByte(')')
	return &ErrorAttr[T]{
		b: x,
	}
}

func (x *Error[T]) constructor2variadic(funcName string, format any, a ...any) *ErrorAttr[T] {
	x.buf.WriteString(x.r.S(funcName, x.a...))
	x.buf.WriteByte('(')

	// render argument 'format' usage
	switch v := format.(type) {
	case string:
		v = strconv.Quote(v)
		x.buf.WriteString(x.r.S(v, x.a...))
	case fmt.Stringer:
		x.buf.WriteString(x.r.S(v.String(), x.a...))
	default:
		x.buf.WriteString(fmt.Sprint(format))
	}

	// render variadic arguments 'a' usage"
	for _, val := range a {
		x.buf.WriteString(", ")
		switch v := val.(type) {
		case string:
			x.buf.WriteString(x.r.S(v, x.a...))
		case fmt.Stringer:
			x.buf.WriteString(x.r.S(v.String(), x.a...))
		default:
			x.buf.WriteString(fmt.Sprint(v))
		}
	}

	x.buf.WriteByte(')')
	return &ErrorAttr[T]{
		b: x,
	}
}

func (x *Error[T]) constructor2(funcName string, err any, msg any) *ErrorAttr[T] {
	x.buf.WriteString(x.r.S(funcName, x.a...))
	x.buf.WriteByte('(')

	// render argument 'err' usage
	switch v := err.(type) {
	case string:
		x.buf.WriteString(x.r.S(v, x.a...))
	case fmt.Stringer:
		x.buf.WriteString(x.r.S(v.String(), x.a...))
	default:
		x.buf.WriteString(fmt.Sprint(err))
	}

	// render argument 'msg' usage
	x.buf.WriteString(", ")
	switch v := msg.(type) {
	case string:
		v = strconv.Quote(v)
		x.buf.WriteString(x.r.S(v, x.a...))
	case fmt.Stringer:
		x.buf.WriteString(x.r.S(v.String(), x.a...))
	default:
		x.buf.WriteString(fmt.Sprint(msg))
	}

	x.buf.WriteByte(')')
	return &ErrorAttr[T]{
		b: x,
	}
}

func (x *Error[T]) constructor3variadic(funcName string, err any, format any, a ...any) *ErrorAttr[T] {
	x.buf.WriteString(x.r.S(funcName, x.a...))
	x.buf.WriteByte('(')

	// render argument 'err' usage
	switch v := err.(type) {
	case string:
		x.buf.WriteString(x.r.S(v, x.a...))
	case fmt.Stringer:
		x.buf.WriteString(x.r.S(v.String(), x.a...))
	default:
		x.buf.WriteString(fmt.Sprint(err))
	}

	// render argument 'format' usage
	x.buf.WriteString(", ")
	switch v := format.(type) {
	case string:
		v = strconv.Quote(v)
		x.buf.WriteString(x.r.S(v, x.a...))
	case fmt.Stringer:
		x.buf.WriteString(x.r.S(v.String(), x.a...))
	default:
		x.buf.WriteString(fmt.Sprint(format))
	}

	// render variadic arguments 'a' usage"
	for _, val := range a {
		x.buf.WriteString(", ")
		switch v := val.(type) {
		case string:
			x.buf.WriteString(x.r.S(v, x.a...))
		case fmt.Stringer:
			x.buf.WriteString(x.r.S(v.String(), x.a...))
		default:
			x.buf.WriteString(fmt.Sprint(v))
		}
	}

	x.buf.WriteByte(')')
	return &ErrorAttr[T]{
		b: x,
	}
}

func (x *ErrorAttr[T]) method1(methodName string, arg1 any) *ErrorAttr[T] {
	x.b.buf.WriteByte('.')
	x.b.buf.WriteString(x.b.r.S(methodName, x.b.a...))
	x.b.buf.WriteByte('(')

	// render argument 'arg1' usage
	switch v := arg1.(type) {
	case string:
		x.b.buf.WriteString(x.b.r.S(v, x.b.a...))
	case fmt.Stringer:
		x.b.buf.WriteString(x.b.r.S(v.String(), x.b.a...))
	default:
		x.b.buf.WriteString(fmt.Sprint(arg1))
	}

	x.b.buf.WriteByte(')')
	return x
}

func (x *ErrorAttr[T]) method2(methodName string, name any, arg2 any) *ErrorAttr[T] {
	x.b.buf.WriteByte('.')
	x.b.buf.WriteString(x.b.r.S(methodName, x.b.a...))
	x.b.buf.WriteByte('(')

	// render argument 'name' usage
	switch v := name.(type) {
	case string:
		v = strconv.Quote(v)
		x.b.buf.WriteString(x.b.r.S(v, x.b.a...))
	case fmt.Stringer:
		x.b.buf.WriteString(x.b.r.S(v.String(), x.b.a...))
	default:
		x.b.buf.WriteString(fmt.Sprint(name))
	}

	// render argument 'arg2' usage
	x.b.buf.WriteString(", ")
	switch v := arg2.(type) {
	case string:
		x.b.buf.WriteString(x.b.r.S(v, x.b.a...))
	case fmt.Stringer:
		x.b.buf.WriteString(x.b.r.S(v.String(), x.b.a...))
	default:
		x.b.buf.WriteString(fmt.Sprint(arg2))
	}

	x.b.buf.WriteByte(')')
	return x
}
