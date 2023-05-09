package gogh

import (
	"strings"
	"testing"
)

func TestPrivate(t *testing.T) {
	tests := []struct {
		arg   string
		parts []string
		want  string
	}{
		{
			arg:  "I",
			want: "i",
		},
		{
			arg:  "ID",
			want: "id",
		},
		{
			arg:  "idNum",
			want: "idNum",
		},
	}
	for _, tt := range tests {
		name := tt.arg
		if len(tt.parts) > 0 {
			name = name + "_" + strings.Join(tt.parts, "_")
		}
		t.Run(name, func(t *testing.T) {
			if got := Private(tt.arg, tt.parts...); got != tt.want {
				t.Errorf("Private() = %v, want %v", got, tt.want)
			}
		})
	}
}
