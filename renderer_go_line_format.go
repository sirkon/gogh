package gogh

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/types"
	"iter"
	"strconv"
	"strings"
	"unicode"

	"github.com/sirkon/errors"
	"github.com/sirkon/go-format/v2"
	"github.com/sirkon/protoast/v2/past"
)

var _ format.Formatter = casesFormatter{}

type casesFormatter struct {
	format byte
	value  string
}

// Clarify to implement format.Formatter
func (c casesFormatter) Clarify(s string) (format.Formatter, error) {
	f := strings.TrimSpace(s)
	switch f {
	case "P", "p", "R", "_", "-":
		return casesFormatter{format: f[0], value: c.value}, nil
	default:
		return nil, errors.New("unsupported format").Str("unsupported-format", f)
	}
}

// Format to implement format.Formatter
func (c casesFormatter) Format(s string) (string, error) {
	switch s {
	case "P":
		return Public(c.value), nil
	case "p":
		return Private(c.value), nil
	case "_":
		return Underscored(c.value), nil
	case "-":
		return Striked(c.value), nil
	case "R":
		return Proto(c.value), nil
	}

	return c.value, nil
}

type sequenceFormatter struct {
	multi bool
	value commasSeq
}

// Clarify to implement format.Formatter
func (c sequenceFormatter) Clarify(s string) (format.Formatter, error) {
	f := strings.TrimSpace(s)
	switch f {
	case `\n`:
		v := c
		v.multi = true
		return v, nil
	default:
		return nil, errors.New("unsupported format").Str("unsupportd-format", f)
	}
}

// Format to implement format.Formatter
func (c sequenceFormatter) Format(s string) (string, error) {
	if c.multi {
		return c.value.Multi(), nil
	}

	return c.value.String(), nil
}

func (r *GoRenderer[T]) renderLine(
	dst *bytes.Buffer,
	line string,
	a ...any,
) {
	r.linebuf.Reset()
	if r.linebuf.Cap() < len(line)+2 {
		r.linebuf.Grow(len(line))
	}
	for part := range inlineUniqueParts(line) {
		switch part.typ {
		case inlinePartTypeText:
			r.linebuf.WriteString(part.val)
		case inlinePartTypeUnique:
			val := r.Uniq(part.val)
			r.Let(part.val, val)
			r.linebuf.WriteString("${")
			r.linebuf.WriteString(part.val)
			r.linebuf.WriteByte('}')
		}
	}

	line = r.linebuf.String()
	bctx := r.renderCtx()

	if bctx == nil {
		bctx = format.NewContextBuilder()
	}

	for i, v := range a {
		if v == nil {
			continue
		}

		d := strconv.Itoa(i)
		if vv, ok := r.uniqTags[v]; ok {
			bctx.Add(d, vv)
			continue
		}
		switch vv := v.(type) {
		case past.Type:
			v = r.Proto(vv)
		case types.Type:
			v = r.Type(vv)
		}
		bctx.Add(d, r.ctxValue(v))
	}

	ctx, err := bctx.Build()
	if err != nil {
		panic(errors.Wrap(err, "build formatting context"))
	}
	res, err := format.Format(line, ctx)
	if err != nil {
		panic(errors.Wrap(err, "format with context"))
	}
	dst.WriteString(res)
}

func (r *GoRenderer[T]) ctxValue(value any) any {
	switch v := value.(type) {
	case types.Type:
		return casesFormatter{value: r.Type(v)}
	case past.Type:
		return casesFormatter{value: r.Proto(v).String()}
	case types.Object:
		return casesFormatter{value: r.Object(v)}
	case Commas:
		return sequenceFormatter{value: v.commasSeq}
	case *Commas:
		return sequenceFormatter{value: v.commasSeq}
	case Params:
		return sequenceFormatter{value: v.commasSeq}
	case *Params:
		return sequenceFormatter{value: v.commasSeq}
	case string:
		return casesFormatter{value: v}
	case fmt.Stringer:
		return casesFormatter{value: v.String()}
	default:
		return v
	}
}

func renderLine(
	dst *bytes.Buffer,
	line string,
	bctx *format.ContextBuilder,
	a ...any,
) {
	if bctx == nil {
		bctx = format.NewContextBuilder()
	}

	for i, v := range a {
		if v == nil {
			continue
		}

		d := strconv.Itoa(i)
		switch vv := v.(type) {
		case string:
			bctx.AddFormatter(d, casesFormatter{value: vv})
		case fmt.Stringer:
			bctx.AddFormatter(d, casesFormatter{value: vv.String()})
		default:
			bctx.Add(d, v)
		}
	}

	ctx, err := bctx.Build()
	if err != nil {
		panic(errors.Wrap(err, "build formatting context"))
	}
	res, err := format.Format(line, ctx)
	if err != nil {
		panic(errors.Wrap(err, "format with context"))
	}
	dst.WriteString(res)
}

func inlineUniqueParts(line string) iter.Seq[inlinePart] {
	return func(yield func(part inlinePart) bool) {
		for len(line) > 0 {
			pos := strings.IndexByte(line, '@')
			if pos < 0 {
				part := inlinePart{
					typ: inlinePartTypeText,
					val: line,
				}
				if !yield(part) {
					return
				}
				return
			}

			if pos == len(line)-1 {
				panic(errors.New(errorInvalidInlineUniqueSyntax))
			}

			if pos > 0 {
				part := inlinePart{
					typ: inlinePartTypeText,
					val: line[:pos],
				}
				if !yield(part) {
					return
				}
				line = line[pos:]
			}

			//
			switch line[1] {
			case '@':
				part := inlinePart{
					typ: inlinePartTypeText,
					val: "@",
				}
				if !yield(part) {
					return
				}
				line = line[2:]
			case '{':
				pos = strings.IndexByte(line, '}')
				if pos < 0 {
					panic(errors.Newf("missing '}' for inline unique after %s", trailer(line)))
				}
				expr, err := parser.ParseExpr(line[2:pos])
				if err != nil || !isIdent(expr) {
					panic(errors.Newf("invalid inline unique argument %s", line[:2]+"\033[1m"+line[2:pos]+"\033[0m"+line[pos:]))
				}
				node := inlinePart{
					typ: inlinePartTypeUnique,
					val: line[2:pos],
				}
				if !yield(node) {
					return
				}
				line = line[pos+1:]

			default:
				var ident string
				ident, line = scrapIdentifier(line[1:])
				if ident == "" {
					panic(errors.New(errorInvalidInlineUniqueSyntax))
				}
				node := inlinePart{
					typ: inlinePartTypeUnique,
					val: ident,
				}
				if !yield(node) {
					return
				}
			}
		}
	}
}

type inlinePart struct {
	typ inplinePartType
	val string
}

type inplinePartType int

const (
	inlinePartTypeText inplinePartType = iota
	inlinePartTypeUnique
)

const errorInvalidInlineUniqueSyntax = "symbol @ must be followed by '@' or '{' or correct Go identifier"

func trailer(line string) (res string) {
	defer func() {
		res = "\033[1m" + res + "\033[0m"
	}()

	if len(line) < 20 {
		return line
	}

	return line[:7] + "..."
}

func scrapIdentifier(line string) (ident string, tail string) {
	for i, c := range line {
		if unicode.IsLetter(c) || c == '_' {
			continue
		}
		if i > 0 && unicode.IsDigit(c) {
			continue
		}

		return line[:i], line[i:]
	}

	return line, ""
}

func isIdent(node ast.Expr) bool {
	_, ok := node.(*ast.Ident)
	return ok
}
