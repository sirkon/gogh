package main

import (
	"go/types"

	"github.com/sirkon/errors"
	"github.com/sirkon/message"
)

func getSourceType(l *souceLoader, pnt sourcePoint) (*types.Named, error) {
	pkg, err := l.loadPkg(pnt.Path)
	if err != nil {
		return nil, errors.Wrap(err, "load package").Str("pkg-path", pnt.Path)
	}

	scope := pkg.Types.Scope()
	typCand := scope.Lookup(pnt.ID)
	if typCand == nil {
		return nil, errors.New("type was not found").
			Str("pkg-path", pnt.Path).
			Str("type-name", pnt.ID)
	}

	switch v := typCand.(type) {
	case *types.TypeName:
		return v.Type().(*types.Named), nil
	default:
		return nil, errors.New("an object the name references is not a type").
			Str("name", pnt.ID).
			Str("object", v.String())
	}
}

func getChainableTypeConstructors(pkg *types.Package, typ *types.Named) (res []*types.Func) {
	scope := pkg.Scope()
	for _, itemName := range scope.Names() {
		item := scope.Lookup(itemName)
		if item == nil {
			message.Warning(
				errors.New("item name listed yet its object was not found").Str("name", itemName),
			)
		}

		if f := getChainableFunc(item, typ); f != nil {
			res = append(res, f)
		}
	}

	return res
}

func getTypePublicChainableMethods(typ *types.Named) (res []*types.Func) {
	for i := 0; i < typ.NumMethods(); i++ {
		if m := getChainableFunc(typ.Method(i), typ); m != nil {
			res = append(res, m)
		}
	}

	return res
}

func getChainableFunc(o types.Object, typ types.Type) *types.Func {
	f, ok := o.(*types.Func)
	if !ok {
		return nil
	}

	if !f.Exported() {
		return nil
	}

	s := f.Type().(*types.Signature)
	if s.Results().Len() != 1 {
		return nil
	}

	if s.Results().At(0).Type() != typ {
		return nil
	}

	return f
}
