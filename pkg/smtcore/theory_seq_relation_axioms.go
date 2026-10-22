// Source: src/smt/theory_seq.cpp assign_eh positive prefix, suffix and
// containment witnesses plus propagate_non_empty guards.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

func (t *SMTSequenceTheory) relationSkolem(name string, arguments []ast.ASTHandle, sort ast.ASTHandle) (ast.ASTHandle, error) {
	state, err := t.State()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	symbol, err := state.manager.Symbols.Intern(name)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return state.manager.MkSequenceSkolem(symbol, arguments, sort)
}

func (t *SMTSequenceTheory) addRelationAxioms(expression ast.ASTHandle, operation int32) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	left, err := state.manager.GetArg(expression, 0)
	if err != nil {
		return err
	}
	right, err := state.manager.GetArg(expression, 1)
	if err != nil {
		return err
	}
	sort, err := state.manager.GetSort(left)
	if err != nil {
		return err
	}
	arguments := []ast.ASTHandle{left, right}
	var definition ast.ASTHandle
	switch operation {
	case 3:
		remainder, makeErr := t.relationSkolem("seq.p.suffix", arguments, sort)
		if makeErr != nil {
			return makeErr
		}
		word, makeErr := state.manager.MkSequenceConcat(left, remainder)
		if makeErr != nil {
			return makeErr
		}
		definition, err = state.manager.MkEq(word, right)
	case 4:
		remainder, makeErr := t.relationSkolem("seq.s.prefix", arguments, sort)
		if makeErr != nil {
			return makeErr
		}
		word, makeErr := state.manager.MkSequenceConcat(remainder, left)
		if makeErr != nil {
			return makeErr
		}
		definition, err = state.manager.MkEq(word, right)
	case 5:
		prefix, makeErr := t.relationSkolem("seq.indexof_left", arguments, sort)
		if makeErr != nil {
			return makeErr
		}
		suffix, makeErr := t.relationSkolem("seq.indexof_right", arguments, sort)
		if makeErr != nil {
			return makeErr
		}
		tail, makeErr := state.manager.MkSequenceConcat(right, suffix)
		if makeErr != nil {
			return makeErr
		}
		word, makeErr := state.manager.MkSequenceConcat(prefix, tail)
		if makeErr != nil {
			return makeErr
		}
		definition, err = state.manager.MkEq(word, left)
	default:
		return ast.ErrASTDomain
	}
	if err != nil {
		return err
	}
	witness, err := state.manager.MkImplies(expression, definition)
	if err != nil {
		return err
	}
	if err = t.axioms.Push(witness); err != nil {
		return err
	}

	nonemptyArgument := left
	if operation == 5 {
		nonemptyArgument = right
	}
	empty, err := state.manager.MkSequenceEmpty(sort)
	if err != nil {
		return err
	}
	emptyEquality, err := state.manager.MkEq(nonemptyArgument, empty)
	if err != nil {
		return err
	}
	nonempty, err := state.manager.MkNot(emptyEquality)
	if err != nil {
		return err
	}
	notRelation, err := state.manager.MkNot(expression)
	if err != nil {
		return err
	}
	guard, err := state.manager.MkImplies(notRelation, nonempty)
	if err != nil {
		return err
	}
	if err = t.axioms.Push(guard); err != nil {
		return err
	}
	if operation != 5 {
		return nil
	}
	prefix, err := state.manager.MkSequencePrefix(right, left)
	if err != nil {
		return err
	}
	notPrefix, err := state.manager.MkNot(prefix)
	if err != nil {
		return err
	}
	if err = t.pushSequenceImplication([]ast.ASTHandle{notRelation}, notPrefix); err != nil {
		return err
	}
	suffix, err := state.manager.MkSequenceSuffix(right, left)
	if err != nil {
		return err
	}
	notSuffix, err := state.manager.MkNot(suffix)
	if err != nil {
		return err
	}
	return t.pushSequenceImplication([]ast.ASTHandle{notRelation}, notSuffix)
}
