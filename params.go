package gogh

import (
	"bytes"
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

// Append appends new <name> <type> pair
func (p *Params) Append(name, value string) {
	p.params = append(p.params, param{
		name:  name,
		value: value,
	})
}
