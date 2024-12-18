package gogh

import "maps"

// valScope a hierarchy of named values with the following rules:
//
//   - You can set a value with the given name once at the current scope.
//   - You can get a value set for the current scope or any parent scope.
//   - You cannot override existing value defined before for the current scope.
//   - You can redefine a value if it exists in some parent scope but the current scope.
//   - You can get a map with values defined in the current and all parent scopes, here
//     values from younger scopes replaces old ones.
type valScope struct {
	parent *valScope
	data   map[string]any
}

func newEmptyValScope() *valScope {
	return &valScope{
		data: map[string]any{},
	}
}

// Next creates child scope over the existing one.
func (v *valScope) Next() *valScope {
	return &valScope{
		parent: v,
		data:   map[string]any{},
	}
}

// Get an existing value.
func (v *valScope) Get(name string) (any, bool) {
	val, ok := v.data[name]
	if ok {
		return val, true
	}

	if v.parent != nil {
		return v.parent.Get(name)
	}

	return nil, false
}

// CheckScope returns true if value exists in the current scope.
func (v *valScope) CheckScope(name string) bool {
	_, ok := v.data[name]
	return ok
}

// Set new value into the current scope.
// Returns false if the given name is already exists in the current scope.
// True otherwise.
func (v *valScope) Set(name string, val any) bool {
	_, ok := v.data[name]
	if ok {
		return false
	}

	v.data[name] = val
	return true
}

// Map returns merged map of values from all scopes, where younger scopes override
// old values for same names.
func (v *valScope) Map() map[string]any {
	var res map[string]any

	if v.parent != nil {
		res = v.parent.Map()
	} else {
		return maps.Clone(v.data)
	}

	for k, v := range v.data {
		res[k] = v
	}

	return res
}
