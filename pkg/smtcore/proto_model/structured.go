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
func (f *structuredFactory) SomeValues(s ast.ASTHandle, factory ValueFactory) (ast.ASTHandle, ast.ASTHandle, bool, error) {
	set, err := f.valueSet(s)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, false, err
	}
	for len(set.values) < 2 {
		var v ast.ASTHandle
		if len(set.values) == 0 {
			v, err = factory.FreshValue(s)
		} else {
			v, err = factory.FreshValue(s)
		}
		if err != nil || v.IsNull() {
			return ast.ASTHandle{}, ast.ASTHandle{}, false, err
		}
		set = f.sets[s]
	}
	return set.values[0], set.values[1], true, nil
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
