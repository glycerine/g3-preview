package protomodel

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
)

type structuredFactory struct {
	manager *ast.ASTManager
	model   *ProtoModel
	family  ast.FamilyID
	sets    map[ast.ASTHandle]*factoryValueSet
	sorts   []ast.ASTRef
	values  []ast.ASTRef
}

func newStructuredFactory(m *ast.ASTManager, family ast.FamilyID, md *ProtoModel) structuredFactory {
	return structuredFactory{manager: m, family: family, model: md, sets: make(map[ast.ASTHandle]*factoryValueSet)}
}
func (f *structuredFactory) FamilyID() ast.FamilyID { return f.family }
func (f *structuredFactory) valueSet(s ast.ASTHandle) (*factoryValueSet, error) {
	if set := f.sets[s]; set != nil {
		return set, nil
	}
	ref, err := f.manager.NewRef(s)
	if err != nil {
		return nil, err
	}
	set := &factoryValueSet{seen: make(map[ast.ASTHandle]struct{})}
	f.sets[s] = set
	f.sorts = append(f.sorts, ref)
	return set, nil
}
func (f *structuredFactory) RegisterValue(v ast.ASTHandle) error {
	s, err := f.manager.GetSort(v)
	if err != nil {
		return err
	}
	set, err := f.valueSet(s)
	if err != nil {
		return err
	}
	if _, found := set.seen[v]; found {
		return nil
	}
	ref, err := f.manager.NewRef(v)
	if err != nil {
		return err
	}
	set.seen[v] = struct{}{}
	set.values = append(set.values, v)
	f.values = append(f.values, ref)
	return nil
}
func (f *structuredFactory) SomeValues(s ast.ASTHandle, factory ValueFactory, first, second ast.ASTRef) (bool, error) {
	set, err := f.valueSet(s)
	if err != nil {
		return false, err
	}
	switch len(set.values) {
	case 0:
		value, err := factory.FreshValue(s)
		if err != nil {
			return false, err
		}
		if err = f.manager.SetRef(first, value); err != nil {
			return false, err
		}
		value, err = factory.FreshValue(s)
		if err != nil {
			return false, err
		}
		if err = f.manager.SetRef(second, value); err != nil {
			return false, err
		}
		a, err := f.manager.RefGet(first)
		if err != nil {
			return false, err
		}
		b, err := f.manager.RefGet(second)
		if err != nil {
			return false, err
		}
		return !a.IsNull() && !b.IsNull(), nil
	case 1:
		value, err := factory.SomeValue(s)
		if err != nil {
			return false, err
		}
		if err = f.manager.SetRef(first, value); err != nil {
			return false, err
		}
		value, err = factory.FreshValue(s)
		if err != nil {
			return false, err
		}
		if err = f.manager.SetRef(second, value); err != nil {
			return false, err
		}
		return !value.IsNull(), nil
	default:
		if err = f.manager.SetRef(first, set.values[0]); err != nil {
			return false, err
		}
		if err = f.manager.SetRef(second, set.values[1]); err != nil {
			return false, err
		}
		return true, nil
	}
}

func (f *structuredFactory) Close() (err error) {
	for i := len(f.values) - 1; i >= 0; i-- {
		err = errors.Join(err, f.manager.DelRef(&f.values[i]))
	}
	for i := len(f.sorts) - 1; i >= 0; i-- {
		err = errors.Join(err, f.manager.DelRef(&f.sorts[i]))
	}
	f.values, f.sorts, f.sets = nil, nil, nil
	return err
}
