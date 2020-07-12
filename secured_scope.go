package gogh

import (
	"go/token"
	"strings"

	"github.com/chonla/roman-number-go"
)

var widelyUsed = []string{
	"chan",
	"string", "bool", "byte", "rune", "error",
	"int", "int8", "int16", "int32", "int64",
	"uint", "uint8", "uint16", "uint32", "uint64",
	"uintptr",
	"float32", "float64",
	"complex32", "complex64",
	"append", "len", "cap", "close", "make", "new", "print", "println", "recover", "real", "imag", "panic", "copy", "delete",
}

// NewScope конструктор Scope
func NewScope() *Scope {
	careful := map[string]int{}
	for _, w := range widelyUsed {
		careful[w] = 1
	}
	return &Scope{
		vars: careful,
		rmn:  roman.NewRoman(),
	}
}

// Scope объект генерирующий непересекающиеся названия переменных с данным префиксом
type Scope struct {
	vars map[string]int
	rmn  *roman.Roman
}

// Var генерация имени переменной
func (s *Scope) Var(prefix string) string {
	name := prefix
	for {
		_, ok := s.vars[prefix]
		if !ok && !token.IsKeyword(prefix) {
			s.vars[prefix] = 1
			return prefix
		}

		s.vars[name]++
		prefix = name + strings.ToLower(s.rmn.ToRoman(s.vars[name]-1))
	}
}
