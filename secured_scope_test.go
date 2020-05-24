package gogh_test

import (
	"testing"

	"github.com/chonla/roman-number-go"

	"github.com/sirkon/gogh"
)

func TestScope_Var(t *testing.T) {
	type fields struct {
		vars map[string]int
		rmn  *roman.Roman
	}
	type args struct {
	}
	tests := []struct {
		name   string
		s      func() *gogh.Scope
		prefix string
		want   string
	}{
		{
			name: "conflict-missing",
			s: func() *gogh.Scope {
				return gogh.NewScope()
			},
			prefix: "var",
			want:   "var",
		},
		{
			name: "conflict-same-prefix",
			s: func() *gogh.Scope {
				s := gogh.NewScope()
				s.Var("var")
				s.Var("var")

				return s
			},
			prefix: "var",
			want:   "varii",
		},
		{
			name: "conflict-different-prefixes",
			s: func() *gogh.Scope {
				s := gogh.NewScope()
				s.Var("var")
				s.Var("var")
				s.Var("vari")
				s.Var("varii")

				return s
			},
			prefix: "var",
			want:   "variv",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.s()
			if got := s.Var(tt.prefix); got != tt.want {
				t.Errorf("Var() = %v, want %v", got, tt.want)
			}
		})
	}
}
