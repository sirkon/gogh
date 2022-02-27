package gogh

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/sirkon/errors"
	"github.com/sirkon/go-format/v2"
)

var _ format.Formatter = casesFormatter{}

type casesFormatter struct {
	format byte
	value  string
}

// Clarify для реализации format.Formatter
func (c casesFormatter) Clarify(s string) (format.Formatter, error) {
	f := strings.TrimSpace(s)
	switch f {
	case "P", "p", "R", "_", "-":
		return casesFormatter{format: f[0], value: c.value}, nil
	default:
		return nil, errors.Newf("format '%s' is not supported", f)
	}
}

// Format для реализации format.Formatter
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

func renderLine(dst *bytes.Buffer, line string, bctx *format.ContextBuilder, a ...interface{}) {
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
