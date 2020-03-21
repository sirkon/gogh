package gogh

import (
	"fmt"
	"reflect"

	"github.com/sirkon/go-format"
)

func formatLine(line string, defaultCtx interface{}, a ...interface{}) string {
	switch len(a) {
	case 0:
		var args []interface{}
		if defaultCtx != nil {
			return formatLine(line, nil, defaultCtx)
		}
		return format.Formatp(line, args...)
	case 1:
		switch v := a[0].(type) {
		case fmt.Stringer:
			return format.Formatp(line, v.String())
		case error:
			return format.Formatp(line, v.Error())
		case map[string]interface{}:
			return format.Formatm(line, v)
		default:
			t := reflect.TypeOf(a[0])
			switch t.Kind() {
			case reflect.Slice:
				f := reflect.ValueOf(format.Formatp)
				v := reflect.ValueOf(a[0])
				var values []reflect.Value
				values = append(
					values,
					reflect.ValueOf(line),
				)
				for i := 0; i < v.Len(); i++ {
					values = append(values, v.Index(i))
				}
				return f.Call(values)[0].String()
			case reflect.Struct:
				return format.Formatg(line, a[0])
			case reflect.Ptr:
				if t.Elem().Kind() == reflect.Struct {
					return format.Formatg(line, reflect.ValueOf(a[0]).Elem().Interface())
				} else {
					return formatLine(line, nil, reflect.ValueOf(a[0]).Elem().Interface())
				}
			default:
				return format.Formatp(line, a[0])
			}
		}
	default:
		return format.Formatp(line, a...)
	}
}
