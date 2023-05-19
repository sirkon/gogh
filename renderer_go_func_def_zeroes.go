package gogh

import (
	"fmt"
	"go/types"

	"github.com/sirkon/gogh/internal/consts"
	"github.com/sirkon/protoast/ast"
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
		vv := zeroValueOfTypesType(r, v.Underlying(), isLast)
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

func zeroValueOfProtoType[T Importer](r *GoRenderer[T], t ast.Type) string {
	switch t.(type) {
	case *ast.Any, *ast.Bytes, *ast.Repeated, *ast.Map, *ast.Message:
		return "nil"

	case *ast.Bool:
		return "false"

	case *ast.Enum,
		*ast.Int32, *ast.Int64,
		*ast.Uint32, *ast.Uint64,
		*ast.Fixed32, *ast.Fixed64,
		*ast.Sfixed32, *ast.Sfixed64,
		*ast.Sint32, *ast.Sint64,
		*ast.Float32, *ast.Float64:

		return "0"

	case *ast.String:
		return `""`

	default:
		panic(fmt.Sprintf("unsupported protobuf type %T", t))
	}
}
