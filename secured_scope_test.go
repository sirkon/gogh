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
			want:   "vari",
		},
		{
			name: "conflict-same-prefix",
			s: func() *gogh.Scope {
				s := gogh.NewScope()
				s.Var("var")
				s.Var("var")

				return s
			},
			prefix: "vari",
			want:   "variii",
		},
		{
			name: "conflict-different-prefixes",
			s: func() *gogh.Scope {
				s := gogh.NewScope()
				s.Var("vari")
				s.Var("varii")
				s.Var("variii")
				s.Var("variv")

				return s
			},
			prefix: "var",
			want:   "varv",
		},
		{
			name: "keywords",
			s: func() *gogh.Scope {
				s := gogh.NewScope()

				return s
			},
			prefix: "break",
			want:   "breaki",
		},
		{
			name: "builtins",
			s: func() *gogh.Scope {
				s := gogh.NewScope()

				return s
			},
			prefix: "string",
			want:   "stringi",
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
