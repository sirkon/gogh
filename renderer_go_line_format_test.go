package gogh

import (
	"slices"
	"testing"

	"github.com/sirkon/deepequal"
)

func Test_inlineUniqueParts(t *testing.T) {
	type testcase struct {
		name   string
		line   string
		parts  []inlinePart
		panics bool
	}

	tests := []testcase{
		{
			name: "no inline",
			line: "func ($s *$service) Name(ctx $ctx.Context, req *NameRequest) (*NameResponse, error) {",
			parts: []inlinePart{
				{
					typ: inlinePartTypeText,
					val: "func ($s *$service) Name(ctx $ctx.Context, req *NameRequest) (*NameResponse, error) {",
				},
			},
			panics: false,
		},
		{
			name: "escape",
			line: "@@head@@tail@@",
			parts: []inlinePart{
				{
					typ: inlinePartTypeText,
					val: "@",
				},
				{
					typ: inlinePartTypeText,
					val: "head",
				},
				{
					typ: inlinePartTypeText,
					val: "@",
				},
				{
					typ: inlinePartTypeText,
					val: "tail",
				},
				{
					typ: inlinePartTypeText,
					val: "@",
				},
			},
		},
		{
			name: "fenced inline",
			line: "@{fenced}",
			parts: []inlinePart{
				{
					typ: inlinePartTypeUnique,
					val: "fenced",
				},
			},
			panics: false,
		},
		{
			name: "identifier inline",
			line: "@ident rest",
			parts: []inlinePart{
				{
					typ: inlinePartTypeUnique,
					val: "ident",
				},
				{
					typ: inlinePartTypeText,
					val: " rest",
				},
			},
			panics: false,
		},
		{
			name: "two identifiers",
			line: "@head @{tail}@@",
			parts: []inlinePart{
				{
					typ: inlinePartTypeUnique,
					val: "head",
				},
				{
					typ: inlinePartTypeText,
					val: " ",
				},
				{
					typ: inlinePartTypeUnique,
					val: "tail",
				},
				{
					typ: inlinePartTypeText,
					val: "@",
				},
			},
		},
		{
			name:   "missing identifier",
			line:   "just@",
			panics: true,
		},
		{
			name:   "missing close brace",
			line:   "just@{ident",
			parts:  nil,
			panics: true,
		},
		{
			name:   "invalid identifier",
			line:   "just@{a+b}",
			parts:  nil,
			panics: true,
		},
		{
			name:   "missing ident identifier",
			line:   "just@.ident",
			parts:  nil,
			panics: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				var panicked bool
				if err := recover(); err != nil {
					panicked = true
					if tt.panics {
						t.Logf("expected panic: %v", err)
					} else {
						t.Errorf("got panic: %v", err)
					}
				}

				if !panicked && tt.panics {
					t.Errorf("panic expected")
				}
			}()

			parts := slices.Collect(inlineUniqueParts(tt.line))
			if !deepequal.Equal(tt.parts, parts) {
				deepequal.SideBySide(t, "inline unique parts", tt.parts, parts)
			}
		})
	}
}

func Test_scrapIdentifier(t *testing.T) {
	tests := []struct {
		name      string
		ident     string
		wantIdent string
		wantTail  string
	}{
		{
			name:      "simple",
			ident:     "abcd",
			wantIdent: "abcd",
			wantTail:  "",
		},
		{
			name:      "split",
			ident:     "_1abcd123 abcd",
			wantIdent: "_1abcd123",
			wantTail:  " abcd",
		},
		{
			name:      "not identifier",
			ident:     "1abcd",
			wantIdent: "",
			wantTail:  "1abcd",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIdent, gotTail := scrapIdentifier(tt.ident)
			if gotIdent != tt.wantIdent {
				t.Errorf("scrapIdentifier() gotIdent = %v, want %v", gotIdent, tt.wantIdent)
			}
			if gotTail != tt.wantTail {
				t.Errorf("scrapIdentifier() gotTail = %v, want %v", gotTail, tt.wantTail)
			}
		})
	}
}
