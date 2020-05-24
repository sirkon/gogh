package gogh

import (
	"strings"

	"github.com/chonla/roman-number-go"
)

// NewScope конструктор Scope
func NewScope() *Scope {
	return &Scope{
		vars: map[string]int{},
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
		if !ok {
			s.vars[prefix] = 1
			return prefix
		}

		s.vars[name]++
		prefix = name + strings.ToLower(s.rmn.ToRoman(s.vars[name]-1))
	}
}
