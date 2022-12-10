package gogh

import (
	"bytes"
	"io"
	"os"

	"github.com/sirkon/errors"
	"github.com/sirkon/go-format/v2"
)

// RawRenderer rendering of plain text files
type RawRenderer struct {
	localname string
	fullname  string
	options   []RendererOption

	vals   map[string]any
	blocks []*bytes.Buffer
}

// N puts new line character
func (r *RawRenderer) N() {
	r.last().WriteByte('\n')
}

// L puts a single formatted line and new line character
func (r *RawRenderer) L(line string, a ...any) {
	renderLine(r.last(), line, r.rendererCtx(), a...)
	r.last().WriteByte('\n')
}

// R puts unformatted line and new line character
func (r *RawRenderer) R(line string) {
	r.last().WriteString(line)
	r.last().WriteByte('\n')
}

// S same as L, just returns string insert of pushing it
func (r *RawRenderer) S(line string, a ...any) string {
	var dst bytes.Buffer
	renderLine(&dst, line, r.rendererCtx(), a...)

	return dst.String()
}

// Z returns extended RawRenderer which will write after the last written line of the current one
// and before any new line pushed after this call.
func (r *RawRenderer) Z() *RawRenderer {
	r.last()

	vals := make(map[string]any, len(r.vals))
	for name, value := range r.vals {
		r.vals[name] = value
	}

	res := &RawRenderer{
		fullname: r.fullname,
		vals:     vals,
		blocks:   r.blocks,
	}
	r.blocks = append(r.blocks, &bytes.Buffer{})
	return res
}

func (r *RawRenderer) path() string {
	return r.fullname
}

func (r *RawRenderer) localPath() string {
	return r.localname
}

func (r *RawRenderer) comment() *bytes.Buffer {
	panic("makes no sense for raw rendering")
}

func (r *RawRenderer) setVals(vals map[string]any) {
	for name, value := range vals {
		if v, ok := r.vals[name]; ok && value != v {
			panic(errors.Newf("attempt to set '%s into different value", name))
		}
	}
}

func (r *RawRenderer) last() *bytes.Buffer {
	if len(r.blocks) == 0 {
		r.blocks = append(r.blocks, &bytes.Buffer{})
	}

	return r.blocks[len(r.blocks)-1]
}

func (r *RawRenderer) rendererCtx() *format.ContextBuilder {
	res := format.NewContextBuilder()
	for name, value := range r.vals {
		res.Add(name, value)
	}

	return res
}

func (r *RawRenderer) render() error {
	for _, opt := range r.options {
		if !opt(r) {
			return nil
		}
	}

	var dest bytes.Buffer
	for _, block := range r.blocks {
		_, _ = io.Copy(&dest, block)
	}

	if err := os.WriteFile(r.fullname, dest.Bytes(), 0644); err != nil {
		return err
	}

	return nil
}
