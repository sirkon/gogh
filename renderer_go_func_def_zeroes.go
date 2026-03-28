package gogh

import (
	"fmt"
	"go/types"

	"github.com/sirkon/protoast/v2/past"

	"github.com/sirkon/gogh/internal/consts"
)

func zeroValueOfTypesType[T Importer](r *GoRenderer[T], t types.Type, isLast bool) (res string) {
	defer func() {
		switch res {
		case "":
			res = "nil"
		}
	}()

	switch v := t.(type) {
	case *types.Basic:
		switch v.Kind() {
		case types.Bool:
			return "false"
		case
			types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
			types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64,
			types.Uintptr,
			types.Float32, types.Float64,
			types.Complex64, types.Complex128:

			return "0"
		case types.String:
			return `""`
		}
	case *types.Named:
		vv := zeroValueOfTypesType[T](r, v.Underlying(), isLast)
		switch vv {
		case "", consts.ErrorTypeZeroSign:
			return vv
		case "{}":
			return r.Type(t) + vv
		default:
			return r.S(`$0($1)`, r.Type(t))
		}
	case *types.Array:
		return r.Type(v) + "{}"
	case *types.Slice, *types.Map, *types.Chan, *types.Pointer:
		return
	case *types.Struct:
		return "{}"
	case *types.Interface:
		if isErrorCompatibleInterface(v) {
			return consts.ErrorTypeZeroSign
		}

		return
	default:
		panic(fmt.Sprintf("unsupported type %T", t))
	}

	panic("have no idea how did we get here")
}

func zeroValueOfProtoType[T Importer](r *GoRenderer[T], t past.Type) string {
	switch t.(type) {
	case *past.Bytes, *past.Repeated, *past.Map, *past.Message:
		return "nil"

	case *past.Bool:
		return "false"

	case *past.Enum,
		*past.Int32, *past.Int64,
		*past.Uint32, *past.Uint64,
		*past.Fixed32, *past.Fixed64,
		*past.Sfixed32, *past.Sfixed64,
		*past.Sint32, *past.Sint64,
		*past.Float, *past.Double:

		return "0"

	case *past.String:
		return `""`

	default:
		panic(fmt.Sprintf("unsupported protobuf type %T", t))
	}
}
