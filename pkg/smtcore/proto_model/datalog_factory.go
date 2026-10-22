// Source: src/smt/theory_dl.cpp finite-domain model factory.
package protomodel

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
)

type datalogValueSet struct {
	values []ast.ASTHandle
	seen   map[ast.ASTHandle]struct{}
	next   uint64
}

// DatalogFactory enumerates the declared finite domain with its canonical
// datalog numerals.
type DatalogFactory struct {
	manager *ast.ASTManager
	family  ast.FamilyID
	sets    map[ast.ASTHandle]*datalogValueSet
	sorts   []ast.ASTRef
	values  []ast.ASTRef
}

func NewDatalogFactory(manager *ast.ASTManager) (*DatalogFactory, error) {
	if manager == nil {
		return nil, ast.ErrASTHandle
	}
	family, err := manager.RegisterDatalogPlugin()
	if err != nil {
		return nil, err
	}
	return &DatalogFactory{manager: manager, family: family, sets: make(map[ast.ASTHandle]*datalogValueSet)}, nil
}

func (f *DatalogFactory) FamilyID() ast.FamilyID { return f.family }

func (f *DatalogFactory) valueSet(sort ast.ASTHandle) (*datalogValueSet, error) {
	finite, err := f.manager.IsDatalogFiniteSort(sort)
	if err != nil || !finite {
		if err != nil {
			return nil, err
		}
		return nil, ast.ErrASTDomain
	}
	if set := f.sets[sort]; set != nil {
		return set, nil
	}
	ref, err := f.manager.NewRef(sort)
	if err != nil {
		return nil, err
	}
	set := &datalogValueSet{seen: make(map[ast.ASTHandle]struct{})}
	f.sets[sort] = set
	f.sorts = append(f.sorts, ref)
	return set, nil
}

func (f *DatalogFactory) insert(value uint64, sort ast.ASTHandle) (ast.ASTHandle, bool, error) {
	set, err := f.valueSet(sort)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	numeral, err := f.manager.MkDatalogNumeral(value, sort)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	if _, found := set.seen[numeral]; found {
		return numeral, false, nil
	}
	ref, err := f.manager.NewRef(numeral)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	set.seen[numeral] = struct{}{}
	set.values = append(set.values, numeral)
	f.values = append(f.values, ref)
	return numeral, true, nil
}

func (f *DatalogFactory) SomeValue(sort ast.ASTHandle) (ast.ASTHandle, error) {
	set, err := f.valueSet(sort)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if len(set.values) != 0 {
		return set.values[0], nil
	}
	value, _, err := f.insert(0, sort)
	return value, err
}

func (f *DatalogFactory) SomeValues(sort ast.ASTHandle) (ast.ASTHandle, ast.ASTHandle, bool, error) {
	size, matched, err := f.manager.DatalogFiniteSortSize(sort)
	if err != nil || !matched {
		return ast.ASTHandle{}, ast.ASTHandle{}, false, err
	}
	first, err := f.SomeValue(sort)
	if err != nil || size < 2 {
		return ast.ASTHandle{}, ast.ASTHandle{}, false, err
	}
	second, _, err := f.insert(1, sort)
	return first, second, err == nil, err
}

func (f *DatalogFactory) FreshValue(sort ast.ASTHandle) (ast.ASTHandle, error) {
	set, err := f.valueSet(sort)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	size, matched, err := f.manager.DatalogFiniteSortSize(sort)
	if err != nil || !matched {
		return ast.ASTHandle{}, err
	}
	for set.next < size {
		candidate := set.next
		set.next++
		value, fresh, insertErr := f.insert(candidate, sort)
		if insertErr != nil {
			return ast.ASTHandle{}, insertErr
		}
		if fresh {
			return value, nil
		}
	}
	return ast.ASTHandle{}, nil
}

func (f *DatalogFactory) RegisterValue(value ast.ASTHandle) error {
	sort, err := f.manager.GetSort(value)
	if err != nil {
		return err
	}
	set, err := f.valueSet(sort)
	if err != nil {
		return err
	}
	if _, found := set.seen[value]; found {
		return nil
	}
	if _, matched, err := f.manager.DatalogNumeralValue(value); err != nil || !matched {
		if err != nil {
			return err
		}
		return ast.ErrASTDomain
	}
	ref, err := f.manager.NewRef(value)
	if err != nil {
		return err
	}
	set.seen[value] = struct{}{}
	set.values = append(set.values, value)
	f.values = append(f.values, ref)
	return nil
}

func (f *DatalogFactory) Close() (err error) {
	if f == nil || f.manager == nil {
		return nil
	}
	for i := len(f.values) - 1; i >= 0; i-- {
		err = errors.Join(err, f.manager.DelRef(&f.values[i]))
	}
	for i := len(f.sorts) - 1; i >= 0; i-- {
		err = errors.Join(err, f.manager.DelRef(&f.sorts[i]))
	}
	f.values, f.sorts, f.sets = nil, nil, nil
	return err
}
