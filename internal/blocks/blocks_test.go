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
		action func(b *blocks.Manager)
		want   string
	}

	tests := []test{
		{
			name: "trivial",
			action: func(b *blocks.Manager) {
				b.Data().WriteString("abc")
			},
			want: "abc",
		},
		{
			name: "single z",
			action: func(b *blocks.Manager) {
				b.Data().WriteString("B")
				b.Insert()
				c := b.Prev()
				b.Data().WriteString("Б")
				c.Data().WriteString("C")
			},
			want: "BCБ",
		},
		{
			name: "multi z",
			action: func(b *blocks.Manager) {
				b.Data().WriteString("B")
				c := b.Insert().Prev()
				c.Data().WriteString("C")
				c.Insert()
				d := c.Insert().Prev()
				d.Data().WriteString("D")
				c.Data().WriteString("Ц")
				b.Data().WriteString("Б")
				d.Data().WriteString("Д")
			},
			want: "BCDДЦБ",
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
