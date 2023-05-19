package gogh

import "strconv"

// Q is a shortcut to write string values as Go source code strings.
type Q string

func (q Q) String() string {
	return strconv.Quote(string(q))
}

// L means "literal" and is intended to be used when raw strings are to
// be represented as quoted string literals while fmt.Stringer-s are
// to keep their original values.
type L string

func (l L) String() string {
	return string(l)
}
