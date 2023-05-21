package testexample

import "github.com/sirkon/gogh"

// R creates `return ....` renderer with an error expression in it.
//
// Usage example:
//     wrp.R(r).Wrapf("err", "do something").Int("count", 13)
// Will be rendered into
//     return "", errors.Wrapf(err, "do something").Int("count", 13)
// In a function returning (string, error).
//
// Remember though, you need to have $ReturnZeroValues in the
// renderer context. It can be set by [M] or [F] method calls
// or directly using [LetReturnZeroValues] method.
// The availability of this context constant is not guaranteed
// for both [M] and [F] in a case the heuristics failed,
// so be careful with it.
//
// [M]: https://pkg.go.dev/github.com/sirkon/gogh#GoRenderer.M
// [F]: https://pkg.go.dev/github.com/sirkon/gogh#GoRenderer.M
// [SetReturnZeroValues]: https://pkg.go.dev/github.com/sirkon/gogh#GoRenderer.LetReturnZeroValues
func R[T gogh.Importer](r *gogh.GoRenderer[T], a ...any) *Error[T] {
	r = r.Scope()
	buffer := gogh.GoRendererBuffer(r)
	buffer.WriteString("return ")
	buffer.WriteString(r.S("$" + gogh.ReturnZeroValues))
	return &Error[T]{
		r:   r,
		buf: buffer,
		a:   a,
	}
}
