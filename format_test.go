package gogh

import (
	"io"
	"testing"
)

type stringerSample struct{}

func (stringerSample) String() string {
	return "I am a stringer"
}

type defaultCtx struct {
	Value string
}

func Test_formatLine(t *testing.T) {
	value := 12
	tests := []struct {
		name       string
		line       string
		defaultCtx interface{}
		a          []interface{}
		want       string
	}{
		{
			name: "no-format",
			line: "abcd",
			a:    nil,
			want: "abcd",
		},
		{
			name: "undoubtably-positional",
			line: "$0 $1",
			a:    []interface{}{1, "a"},
			want: "1 a",
		},
		{
			name: "stringer",
			line: "$",
			a:    []interface{}{stringerSample{}},
			want: "I am a stringer",
		},
		{
			name: "error",
			line: "$",
			a:    []interface{}{io.EOF},
			want: io.EOF.Error(),
		},
		{
			name: "hidden-positional",
			line: "$0 $1",
			a:    []interface{}{[]interface{}{1, "a"}},
			want: "1 a",
		},
		{
			name: "map",
			line: "$a ${b}",
			a: []interface{}{map[string]interface{}{
				"a": 1,
				"b": "a",
			}},
			want: "1 a",
		},
		{
			name: "struct-value",
			line: "$A $B",
			a: []interface{}{
				struct {
					A int
					B string
				}{
					1,
					"a",
				},
			},
			want: "1 a",
		},
		{
			name: "struct-value",
			line: "$A $B",
			a: []interface{}{
				&struct {
					A int
					B string
				}{
					1,
					"a",
				},
			},
			want: "1 a",
		},
		{
			name: "pointer-dereference",
			line: "$0",
			a:    []interface{}{&value},
			want: "12",
		},
		{
			name: "with-ctx",
			line: "${Value}",
			defaultCtx: defaultCtx{
				Value: "value",
			},
			want: "value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatLine(tt.line, tt.defaultCtx, tt.a...); got != tt.want {
				t.Errorf("formatLine() = %v, want %v", got, tt.want)
			}
		})
	}
}
