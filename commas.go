package gogh

import "strings"

// A shortcut function to create comma separated list out of
// string list values quickly.
func A(a ...string) Commas {
	var res Commas
	for _, v := range a {
		res.Add(v)
	}

	return res
}

// Params parameters list
type Params struct {
	commasSeq
}

// Add adds new parameter
func (p *Params) Add(name, typ string) *Params {
	p.add(name, typ)
	return p
}

// Commas comma-separated list
type Commas struct {
	commasSeq
}

// Add adds new value
func (c *Commas) Add(value string) *Commas {
	c.add(value, "")
	return c
}

// commasSeq to represent a sequence of params
type commasSeq struct {
	data [][2]string
}

func (s *commasSeq) add(name, typ string) *commasSeq {
	s.data = append(s.data, [2]string{name, typ})
	return s
}

// String returns a line of comma-separated entities
func (s commasSeq) String() string {
	var buf strings.Builder
	for i, v := range s.data {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(v[0])
		buf.WriteByte(' ')
		buf.WriteString(v[1])
	}

	return buf.String()
}

// Multi returns a parameters text representation written in one per line
func (s *commasSeq) Multi() string {
	var buf strings.Builder
	for _, v := range s.data {
		buf.WriteString(v[0])
		buf.WriteByte(' ')
		buf.WriteString(v[1])
		buf.WriteString(",\n")
	}

	return buf.String()
}
