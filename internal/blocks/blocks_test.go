package blocks_test

import (
	"io"
	"strings"
	"testing"

	"github.com/sirkon/gogh/internal/blocks"
)

func TestBlocks(t *testing.T) {
	type test struct {
		name   string
		action func(b *blocks.Blocks)
		want   string
	}

	tests := []test{
		{
			name: "trivial",
			action: func(b *blocks.Blocks) {
				b.Data().WriteString("abc")
			},
			want: "abc",
		},
		{
			name: "grew once",
			action: func(b *blocks.Blocks) {
				b.Next().Data().WriteString("def")
				b.Data().WriteString("abc")
			},
			want: "abcdef",
		},
		{
			name: "grew twice",
			action: func(b *blocks.Blocks) {
				b.Next().Next().Data().WriteString("ghi")
				b.Data().WriteString("abc")
				b.Next().Data().WriteString("def")
			},
			want: "abcdefghi",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := blocks.New()
			tt.action(b)

			var result strings.Builder
			for _, buf := range b.Collect() {
				_, _ = io.Copy(&result, buf)
			}

			if tt.want != result.String() {
				t.Errorf("unexpected result '%s', wanted '%s'", result.String(), tt.want)
			}
		})
	}
}
