package gogh

import (
	"reflect"
	"sort"
	"testing"
)

func TestImports_Add(t *testing.T) {
	tests := []struct {
		name            string
		existingImports map[string]string
		alias           string
		path            string

		wantPanic bool
	}{
		{
			name:            "ok",
			existingImports: map[string]string{},
			alias:           "",
			path:            "math",
			wantPanic:       false,
		},
		{
			name: "ok-with-duplicate",
			existingImports: map[string]string{
				"math": "",
			},
			alias:     "",
			path:      "math",
			wantPanic: false,
		},
		{
			name: "failure-different-custom-alias",
			existingImports: map[string]string{
				"math": "",
			},
			alias:     "m",
			path:      "math",
			wantPanic: true,
		},
		{
			name: "failure-empty-alias-after-custom",
			existingImports: map[string]string{
				"math": "m",
			},
			alias:     "",
			path:      "math",
			wantPanic: true,
		},
		{
			name: "failure-different-alias",
			existingImports: map[string]string{
				"math": "m",
			},
			alias:     "mth",
			path:      "math",
			wantPanic: true,
		},
		{
			name:            "failure-empty-import-path",
			existingImports: map[string]string{},
			alias:           "",
			path:            "",
			wantPanic:       true,
		},
		{
			name:            "failure-aliased-C-import",
			existingImports: map[string]string{},
			alias:           "c",
			path:            "C",
			wantPanic:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Log(r)
						return
					}

					t.Errorf("panic was expected")
				}()
			}
			i := &Imports{
				pkgs: tt.existingImports,
			}
			i.Add(tt.alias, tt.path)
		})
	}
}

func TestImports_Result(t *testing.T) {
	imports := NewImports(GenericWeighter())

	imports.Add("", "C")
	imports.Add("", "math")
	imports.Add("pkg", "github.com/sirkon/gogh")

	sample := []ImportsGroup{
		{
			{
				Path: "C",
			},
		},
		{
			{
				Alias: "",
				Path:  "math",
			},
		},
		{
			{
				Alias: "pkg",
				Path:  "github.com/sirkon/gogh",
			},
		},
	}
	if !reflect.DeepEqual(sample, imports.Result()) {
		t.Errorf("%#v expected, got %#v", sample, imports.Result())
	}
}

func TestImport_String(t *testing.T) {
	tests := []struct {
		name  string
		alias string
		path  string
		want  string
	}{
		{
			name:  "non-aliased",
			alias: "",
			path:  "github.com/sirkon/gogh",
			want:  `"github.com/sirkon/gogh"`,
		},
		{
			name:  "aliased",
			alias: "alias",
			path:  "github.com/sirkon/gogh",
			want:  `alias "github.com/sirkon/gogh"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := Import{
				Alias: tt.alias,
				Path:  tt.path,
			}
			if got := i.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_heuristicCmp(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{
			name: "no-numeric",
			a:    "github.com/sirkon/gogh",
			b:    "github.com/sirkon/gogh/v2",
			want: true,
		},
		{
			name: "no-numeric",
			a:    "github.com/sirkon/gogh/v2",
			b:    "github.com/sirkon/gogh",
			want: false,
		},
		{
			name: "head_numbers",
			a:    "dos_v1.1.1/pkg",
			b:    "dos_v1.1.2/pkg",
			want: true,
		},
		{
			name: "head_only_number",
			a:    "dos/pkg",
			b:    "dos1.1.1/pkg",
			want: true,
		},
		{
			name: "head_only_number",
			a:    "dos1.1.1/pkg",
			b:    "dos/pkg",
			want: false,
		},
		{
			name: "same",
			a:    "dos/pkg",
			b:    "dos/pkg",
			want: false,
		},
		{
			name: "roman-suffix",
			a:    "github.com/lewisIX",
			b:    "github.com/lewisXIV",
			want: true,
		},
		{
			name: "roman-suffix",
			a:    "github.com/lewisx",
			b:    "github.com/lewisy",
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := heuristicCmp(tt.a, tt.b); got != tt.want {
				t.Errorf("heuristicCmp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestImportsGroup(t *testing.T) {
	g := ImportsGroup{
		{
			Path: "github.com/sirkon/go-format/v2",
		},
		{
			Path: "github.com/sirkon/go-format",
		},
		{
			Path: "github.com/sirkon/gosrcfmt",
		},
	}
	sample := ImportsGroup{
		{
			Path: "github.com/sirkon/go-format",
		},
		{
			Path: "github.com/sirkon/go-format/v2",
		},
		{
			Path: "github.com/sirkon/gosrcfmt",
		},
	}
	sort.Sort(g)
	if !reflect.DeepEqual(sample, g) {
		t.Errorf("expected %#v after sorting, got %#v", sample, g)
	}
}
