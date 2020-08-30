package gogh

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// Commas represent comma-separated list of setup
type Commas struct {
	values []string
}

func (c Commas) String() string {
	return strings.Join(c.values, ", ")
}

// Mutliline returns a stringer rendering commas in multiple lines in case there are more two or more items collected
func (c Commas) Mutliline() fmt.Stringer {
	return multilineCommas{
		commas: c,
	}
}

// Append appends a new value into the list. Raises a panic if value cannot be casted easily to the string
func (c *Commas) Append(value interface{}) {
	var val string
	switch v := value.(type) {
	case bool:
		val = strconv.FormatBool(v)
	case string:
		val = v
	case int8:
		val = strconv.FormatInt(int64(v), 10)
	case int16:
		val = strconv.FormatInt(int64(v), 10)
	case int32:
		val = strconv.FormatInt(int64(v), 10)
	case int64:
		val = strconv.FormatInt(v, 10)
	case int:
		val = strconv.FormatInt(int64(v), 10)
	case uint8:
		val = strconv.FormatUint(uint64(v), 10)
	case uint16:
		val = strconv.FormatUint(uint64(v), 10)
	case uint32:
		val = strconv.FormatUint(uint64(v), 10)
	case uint64:
		val = strconv.FormatUint(v, 10)
	case uint:
		val = strconv.FormatUint(uint64(v), 10)
	case fmt.Stringer:
		val = v.String()
	default:
		panic(fmt.Errorf("type %T is not supported", value))
	}
	c.values = append(c.values, val)
}

type multilineCommas struct {
	commas Commas
}

func (n multilineCommas) String() string {
	if len(n.commas.values) < 2 {
		return n.commas.String()
	}

	var dest bytes.Buffer
	dest.WriteByte('\n')
	for _, item := range n.commas.values {
		dest.WriteString(item)
		dest.WriteString(",\n")
	}

	return dest.String()
}
