package heuristics

import (
	"fmt"
	"strings"

	"github.com/sirkon/gogh/internal/consts"
)

type kind uint32

const (
	boolean = iota + 1
	number
	str
	array
	slice
	maptype
	pointer
	channel
	errortype
	unclear
)

func guessKind(typ string) kind {
	switch typ {
	case "bool":
		return boolean
	case "string":
		return str
	case "byte", "uintptr", "uint", "int":
		return number
	case "error":
		return errortype
	}

	switch {
	case strings.HasPrefix(typ, "int"):
		return number
	case strings.HasPrefix(typ, "uint"):
		return number
	case strings.HasPrefix(typ, "float"):
		return number
	case strings.HasPrefix(typ, "complex"):
		return number
	case strings.HasPrefix(typ, "[]"):
		return slice
	case strings.HasPrefix(typ, "["):
		// Because array is [N]<typeName> and slices with their [
		// have just passed.
		return array
	case strings.HasPrefix(typ, "map["):
		return maptype
	case strings.HasPrefix(typ, "*"):
		return pointer
	case strings.HasPrefix(typ, "chan"):
		return channel
	default:
		return unclear
	}
}

func kindZero(typ string, v kind) string {
	switch v {
	case boolean:
		return "false"
	case number:
		return "0"
	case str:
		return `""`
	case array:
		return typ + "{}"
	case slice, maptype, pointer, channel:
		return "nil"
	case errortype:
		return consts.ErrorTypeZeroSign
	default:
		panic(fmt.Sprintf("unsupported kind %d", v))
	}
}
