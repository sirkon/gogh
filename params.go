package gogh

import (
	"bytes"
	"fmt"
)

type param struct {
	name  string
	value string
}

// Params represent comma-separated list of function arguments or result values
type Params struct {
	params []param
}

func (p Params) String() string {
	if len(p.params) == 0 {
		return ""
	}
	var buf bytes.Buffer
	var length int
	for _, p := range p.params {
		length += len(p.name) + len(p.value) + 1
	}
	length += 2 * (len(p.params) - 1)
	buf.Grow(length)
	for i, p := range p.params {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(p.name)
		buf.WriteByte(' ')
		buf.WriteString(p.value)
	}
	return buf.String()
}

// Multiline return a stringer rendering params in multiple lines in cases two or more were collected
func (p Params) Multiline() fmt.Stringer {
	return multilineParams{
		params: p,
	}
}

// Append appends new <name> <type> pair
func (p *Params) Append(name, value string) {
	p.params = append(p.params, param{
		name:  name,
		value: value,
	})
}

type multilineParams struct {
	params Params
}

func (m multilineParams) String() string {
	if len(m.params.params) < 2 {
		return m.params.String()
	}

	var buf bytes.Buffer
	length := 1
	for _, p := range m.params.params {
		length += len(p.name) + len(p.value) + 3
	}
	buf.Grow(length)

	buf.WriteByte('\n')
	for _, p := range m.params.params {
		buf.WriteString(p.name)
		buf.WriteByte(' ')
		buf.WriteString(p.value)
		buf.WriteString(",\n")
	}

	return buf.String()
}
