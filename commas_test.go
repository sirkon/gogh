package gogh_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sirkon/gogh"
)

type customStringer struct{}

func (customStringer) String() string {
	return "i am a stringer"
}

func ExampleCommas() {
	var c gogh.Commas
	c.Append(true)
	c.Append("str")
	c.Append(int8(1))
	c.Append(int16(2))
	c.Append(int32(3))
	c.Append(int64(4))
	c.Append(5)
	c.Append(uint8(6))
	c.Append(uint16(7))
	c.Append(uint32(8))
	c.Append(uint64(9))
	c.Append(uint(10))
	c.Append(customStringer{})
	fmt.Println(c)

	// Output:
	// true, str, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, i am a stringer
}

func TestCommas_Append(t *testing.T) {

	tests := []struct {
		name      string
		setup     func(c *gogh.Commas)
		want      string
		wantPanic bool
	}{
		{
			name: "panic",
			setup: func(c *gogh.Commas) {
				c.Append(nil)
			},
			want:      "",
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		name := tt.name
		wantPanic := tt.wantPanic
		t.Run(name, func(t *testing.T) {
			defer func() {
				r := recover()
				switch {
				case r != nil && !wantPanic:
					t.Error(r)
				case r == nil && wantPanic:
					t.Log("panic expected but not raised")
				}
			}()
			var c gogh.Commas
			tt.setup(&c)
			assert.Equal(t, tt.want, c.String())
		})
	}
}
