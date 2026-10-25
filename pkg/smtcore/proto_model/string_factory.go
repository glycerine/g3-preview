// Source: src/smt/theory_str.h str_value_factory.
package protomodel

import (
	"errors"
	"strconv"

	"github.com/glycerine/g3/pkg/ast"
)

// StringFactory belongs to the separate theory_str backend. In particular,
// register_value is intentionally ignored; SequenceFactory's collision and
// delimiter policy is not the policy of this native factory.
type StringFactory struct {
	manager *ast.ASTManager
	family  ast.FamilyID
	strings map[string]struct{}
	next    uint32
	refs    []ast.ASTRef
	owned   map[ast.ASTHandle]struct{}
}

func NewStringFactory(model *ProtoModel, family ast.FamilyID) (*StringFactory, error) {
	if model == nil || family == ast.NullFamilyID {
		return nil, ast.ErrASTDomain
	}
	if err := model.open(); err != nil {
		return nil, err
	}
	return &StringFactory{manager: model.manager, family: family, strings: make(map[string]struct{}), owned: make(map[ast.ASTHandle]struct{})}, nil
}

func (f *StringFactory) FamilyID() ast.FamilyID { return f.family }

// The Go factory interface returns handles rather than the source's expr_ref
// output parameters. Retain generated expressions until the factory closes.
func (f *StringFactory) retain(value ast.ASTHandle) (ast.ASTHandle, error) {
	if _, found := f.owned[value]; found {
		return value, nil
	}
	ref, err := f.manager.NewRef(value)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	f.refs = append(f.refs, ref)
	f.owned[value] = struct{}{}
	return value, nil
}

func (f *StringFactory) stringValue(text string) (ast.ASTHandle, error) {
	if f == nil || f.manager == nil {
		return ast.ASTHandle{}, ast.ErrASTHandle
	}
	value, err := f.manager.MkString(text)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return f.retain(value)
}

func (f *StringFactory) SomeValue(ast.ASTHandle) (ast.ASTHandle, error) {
	return f.stringValue("some value")
}

func (f *StringFactory) SomeValues(ast.ASTHandle) (ast.ASTHandle, ast.ASTHandle, bool, error) {
	first, err := f.stringValue("value 1")
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, false, err
	}
	second, err := f.stringValue("value 2")
	return first, second, err == nil, err
}

func (f *StringFactory) FreshValue(sort ast.ASTHandle) (ast.ASTHandle, error) {
	if f == nil || f.manager == nil {
		return ast.ASTHandle{}, ast.ErrASTHandle
	}
	stringSort, err := f.manager.IsStringSort(sort)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if stringSort {
		for {
			text := "!" + strconv.FormatUint(uint64(f.next), 16) + "!"
			f.next++
			if _, found := f.strings[text]; found {
				continue
			}
			f.strings[text] = struct{}{}
			return f.stringValue(text)
		}
	}
	sequenceSort, regex, err := f.manager.RegexSequenceSort(sort)
	if err != nil || !regex {
		if err == nil {
			err = ast.ErrASTDomain
		}
		return ast.ASTHandle{}, err
	}
	value, err := f.FreshValue(sequenceSort)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	regexValue, err := f.manager.MkSequenceToRegex(value)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return f.retain(regexValue)
}

func (*StringFactory) RegisterValue(ast.ASTHandle) error { return nil }

func (f *StringFactory) Close() (result error) {
	if f == nil || f.manager == nil {
		return nil
	}
	for i := len(f.refs) - 1; i >= 0; i-- {
		result = errors.Join(result, f.manager.DelRef(&f.refs[i]))
	}
	f.refs, f.strings, f.owned, f.manager = nil, nil, nil, nil
	return result
}

var _ ValueFactory = (*StringFactory)(nil)
