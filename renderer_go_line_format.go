package gogh

import (
	"bytes"
	"fmt"
	"go/types"
	"strconv"
	"strings"

	"github.com/sirkon/errors"
	"github.com/sirkon/go-format/v2"
	"github.com/sirkon/protoast/ast"
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
	bctx := r.renderCtx()

	if bctx == nil {
		bctx = format.NewContextBuilder()
	}

	for i, v := range a {
		if v == nil {
			continue
		}

		d := strconv.Itoa(i)
		bctx.Add(d, v)
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
	case ast.Type:
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
