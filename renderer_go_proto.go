package gogh

// ProtocType сущность для различных сценариев использования сгенерированного protoc-gen-go типа
type ProtocType struct {
	pointer  bool
	source   string
	selector string
}

// String имя сгенерированного типа
func (s ProtocType) String() string {
	if s.source != "" {
		return s.source + "." + s.selector
	} else {
		return s.selector
	}
}

// Impl используемый тип (добавляется указатель, если proto.Message реализуется на указателе на значение типа)
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

// Local локальное имя сгенерированного типа
func (s ProtocType) Local() string {
	return s.selector
}

// LocalImpl локальный используемый тип (добавляется указатель, если proto.Message реализуется на указателе на значение типа)
func (s ProtocType) LocalImpl() string {
	if s.pointer {
		return "*" + s.selector
	}
	return s.selector
}

// Pkg возвращает название пакета
func (s ProtocType) Pkg() string {
	return s.source
}

func raw(value string) ProtocType {
	return ProtocType{
		selector: value,
	}
}
