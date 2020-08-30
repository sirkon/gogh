package gogh_test

import (
	"fmt"
	"reflect"
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

func TestCommas_Mutliline(t *testing.T) {
	tests := []struct {
		name   string
		commas func() gogh.Commas
		want   string
	}{
		{
			name: "less than two lines",
			commas: func() gogh.Commas {
				var commas gogh.Commas
				commas.Append("val")
				return commas
			},
			want: "val",
		},
		{
			name: "more than two lines",
			commas: func() gogh.Commas {
				var commas gogh.Commas
				commas.Append("val1")
				commas.Append("2")
				return commas
			},
			want: "\nval1,\n2,\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.commas().Mutliline().String()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Mutliline() = %v, want %v", got, tt.want)
			}
		})
	}
}
