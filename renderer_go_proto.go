package gogh

// ProtocType is an entity for various usage scenarios of a generated protoc-gen-go type.
type ProtocType struct {
	pointer  bool
	source   string
	selector string
}

// String returns the name of the generated type.
func (s ProtocType) String() string {
	if s.source != "" {
		return s.source + "." + s.selector
	} else {
		return s.selector
	}
}

// Impl returns the type as used (a pointer is added if proto.Message is
// implemented on a pointer to the value type).
func (s ProtocType) Impl() string {
	var expr string
	if s.source != "" {
		expr = s.source + "." + s.selector
	} else {
		expr = s.selector
	}
	if s.pointer {
		return "*" + expr
	}
	return expr
}

// Local returns the local name of the generated type.
func (s ProtocType) Local() string {
	return s.selector
}

// LocalImpl returns the local type as used (a pointer is added if proto.Message
// is implemented on a pointer to the value type).
func (s ProtocType) LocalImpl() string {
	if s.pointer {
		return "*" + s.selector
	}
	return s.selector
}

// Pkg returns the package name.
func (s ProtocType) Pkg() string {
	return s.source
}

func raw(value string) ProtocType {
	return ProtocType{
		selector: value,
	}
}
