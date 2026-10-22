// Source: src/smt/theory_seq_empty.h seq_factory.
package protomodel

import (
	"strconv"
	"strings"

	"github.com/glycerine/g3/pkg/ast"
)

// SequenceFactory supplies canonical and fresh values for sequence and
// regular-expression sorts during SMT model construction.
type SequenceFactory struct {
	manager   *ast.ASTManager
	family    ast.FamilyID
	model     *ProtoModel
	strings   map[string]struct{}
	next      uint64
	delimiter string
}

func NewSequenceFactory(model *ProtoModel, family ast.FamilyID) (*SequenceFactory, error) {
	if model == nil || family == ast.NullFamilyID {
		return nil, ast.ErrASTDomain
	}
	if err := model.open(); err != nil {
		return nil, err
	}
	return &SequenceFactory{
		manager: model.manager, family: family, model: model,
		strings: map[string]struct{}{"": {}, "a": {}, "b": {}}, delimiter: "!",
	}, nil
}

func (f *SequenceFactory) FamilyID() ast.FamilyID { return f.family }

func (f *SequenceFactory) classify(sort ast.ASTHandle) (sequence, regex bool, element ast.ASTHandle, err error) {
	if f == nil || f.manager == nil || f.model == nil {
		return false, false, ast.ASTHandle{}, ast.ErrASTHandle
	}
	element, sequence, err = f.manager.SequenceSortElement(sort)
	if err != nil || sequence {
		return sequence, false, element, err
	}
	element, regex, err = f.manager.RegexSequenceSort(sort)
	return false, regex, element, err
}

func (f *SequenceFactory) SomeValue(sort ast.ASTHandle) (ast.ASTHandle, error) {
	sequence, regex, element, err := f.classify(sort)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if sequence {
		return f.manager.MkSequenceEmpty(sort)
	}
	if regex {
		empty, err := f.manager.MkSequenceEmpty(element)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		return f.manager.MkSequenceToRegex(empty)
	}
	return ast.ASTHandle{}, ast.ErrASTDomain
}

func (f *SequenceFactory) SomeValues(sort ast.ASTHandle) (ast.ASTHandle, ast.ASTHandle, bool, error) {
	stringSort, err := f.manager.IsStringSort(sort)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, false, err
	}
	if stringSort {
		first, err := f.manager.MkString("a")
		if err != nil {
			return ast.ASTHandle{}, ast.ASTHandle{}, false, err
		}
		second, err := f.manager.MkString("b")
		return first, second, err == nil, err
	}
	sequence, _, element, err := f.classify(sort)
	if err != nil || !sequence {
		return ast.ASTHandle{}, ast.ASTHandle{}, false, err
	}
	first, second, ok, err := f.model.SomeValues(element)
	if err != nil || !ok {
		return ast.ASTHandle{}, ast.ASTHandle{}, false, err
	}
	first, err = f.manager.MkSequenceUnit(first)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, false, err
	}
	second, err = f.manager.MkSequenceUnit(second)
	return first, second, err == nil, err
}

func (f *SequenceFactory) freshString() (ast.ASTHandle, error) {
	for {
		text := f.delimiter + strconv.FormatUint(f.next, 16) + f.delimiter
		f.next++
		if _, found := f.strings[text]; found {
			continue
		}
		f.strings[text] = struct{}{}
		return f.manager.MkString(text)
	}
}

func (f *SequenceFactory) FreshValue(sort ast.ASTHandle) (ast.ASTHandle, error) {
	stringSort, err := f.manager.IsStringSort(sort)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if stringSort {
		return f.freshString()
	}
	sequence, regex, element, err := f.classify(sort)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if regex {
		value, err := f.FreshValue(element)
		if err != nil || value.IsNull() {
			return ast.ASTHandle{}, err
		}
		return f.manager.MkSequenceToRegex(value)
	}
	if sequence {
		value, err := f.model.FreshValue(element)
		if err != nil || value.IsNull() {
			return ast.ASTHandle{}, err
		}
		return f.manager.MkSequenceUnit(value)
	}
	character, err := f.manager.IsCharacterSort(sort)
	if err != nil || character {
		// The retained AST surface has no standalone character literal
		// constructor. Character values are reached through string constants.
		return ast.ASTHandle{}, err
	}
	return ast.ASTHandle{}, ast.ErrASTDomain
}

func (f *SequenceFactory) RegisterValue(value ast.ASTHandle) error {
	if f == nil || f.manager == nil {
		return ast.ErrASTHandle
	}
	text, matched, err := f.manager.SequenceStringValue(value)
	if err != nil || !matched {
		return err
	}
	f.strings[text] = struct{}{}
	if !strings.Contains(text, f.delimiter) {
		return nil
	}
	for {
		f.delimiter += "!"
		collision := false
		for existing := range f.strings {
			if strings.Contains(existing, f.delimiter) {
				collision = true
				break
			}
		}
		if !collision {
			return nil
		}
	}
}

func (f *SequenceFactory) Close() error {
	if f != nil {
		f.manager = nil
		f.model = nil
		f.strings = nil
	}
	return nil
}
