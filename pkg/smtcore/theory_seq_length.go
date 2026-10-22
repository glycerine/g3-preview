// Source: src/smt/theory_seq.cpp add_length_axiom and the retained exact
// concat, unit, empty, string, int.to.str and non-negativity cases.
package smtcore

import (
	"strconv"

	"github.com/glycerine/g3/pkg/ast"
)

func (t *SMTSequenceTheory) assertConstraint(expression ast.ASTHandle) error {
	state, err := t.State()
	if err != nil || state.manager.IsTrue(expression) {
		return err
	}
	if err = state.Internalize(expression, false); err != nil {
		return err
	}
	literal, err := state.Literal(expression)
	if err != nil {
		return err
	}
	return state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{literal})
}

func (t *SMTSequenceTheory) integer(value uint64) (ast.ASTHandle, error) {
	state, err := t.State()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return state.manager.ArithmeticIntegerNumeralText(strconv.FormatUint(value, 10))
}

func (t *SMTSequenceTheory) lengthDefinition(argument ast.ASTHandle) (ast.ASTHandle, bool, error) {
	state, err := t.State()
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	if text, matched, textErr := state.manager.SequenceStringValue(argument); textErr != nil || matched {
		if textErr != nil {
			return ast.ASTHandle{}, false, textErr
		}
		value, makeErr := t.integer(uint64(len(text)))
		return value, true, makeErr
	}
	if empty, emptyErr := state.manager.IsSequenceEmpty(argument); emptyErr != nil || empty {
		if emptyErr != nil {
			return ast.ASTHandle{}, false, emptyErr
		}
		value, makeErr := t.integer(0)
		return value, true, makeErr
	}
	if _, unit, unitErr := state.manager.SequenceUnitArgument(argument); unitErr != nil || unit {
		if unitErr != nil {
			return ast.ASTHandle{}, false, unitErr
		}
		value, makeErr := t.integer(1)
		return value, true, makeErr
	}
	left, right, concat, concatErr := state.manager.SequenceConcatParts(argument)
	if concatErr != nil || !concat {
		return ast.ASTHandle{}, false, concatErr
	}
	leftLength, err := state.manager.MkSequenceLength(left)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	rightLength, err := state.manager.MkSequenceLength(right)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	value, err := state.manager.MkArithmeticAdd(leftLength, rightLength)
	return value, true, err
}

func (t *SMTSequenceTheory) addLengthAxiom(length ast.ASTHandle) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	argument, matched, err := state.manager.SequenceUnaryOperationArgument(length, 9)
	if err != nil || !matched {
		if err != nil {
			return err
		}
		return ast.ErrASTDomain
	}
	definition, defined, err := t.lengthDefinition(argument)
	if err != nil {
		return err
	}
	if defined {
		equality, makeErr := state.manager.MkEq(length, definition)
		if makeErr != nil {
			return makeErr
		}
		return t.axioms.Push(equality)
	}
	zero, err := t.integer(0)
	if err != nil {
		return err
	}
	nonnegative, err := state.manager.MkArithmeticGE(length, zero)
	if err != nil {
		return err
	}
	if err = t.axioms.Push(nonnegative); err != nil {
		return err
	}
	integer, itos, err := state.manager.SequenceIntToStringArgument(argument)
	if err != nil || !itos {
		return err
	}
	one, err := t.integer(1)
	if err != nil {
		return err
	}
	integerNonnegative, err := state.manager.MkArithmeticGE(integer, zero)
	if err != nil {
		return err
	}
	lengthAtLeastOne, err := state.manager.MkArithmeticGE(length, one)
	if err != nil {
		return err
	}
	implication, err := state.manager.MkImplies(integerNonnegative, lengthAtLeastOne)
	if err != nil {
		return err
	}
	return t.axioms.Push(implication)
}
