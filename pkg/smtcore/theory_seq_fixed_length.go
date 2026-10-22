// Source: src/smt/theory_seq.cpp fixed_length and mk_decompose.
package smtcore

import (
	"math/big"

	"github.com/glycerine/g3/pkg/ast"
)

const smtSequenceMaximumFixedDecomposition = 2048

func (t *SMTSequenceTheory) addFixedLengthDecomposition() (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	arithmeticFamily, err := state.manager.ArithmeticFamilyID()
	if err != nil {
		return false, err
	}
	arithmetic, ok := state.Theory(arithmeticFamily).(*SMTLinearArithmeticTheory)
	if !ok {
		return false, nil
	}
	fixed := arithmetic.fixedLinearAssignments()
	seen := make(map[ast.ASTHandle]struct{})
	for _, node := range t.variables {
		if node == nil {
			continue
		}
		expression := node.Owner()
		if _, found := seen[expression]; found {
			continue
		}
		seen[expression] = struct{}{}
		variable, variableErr := t.sequenceVariable(expression)
		if variableErr != nil {
			return false, variableErr
		}
		if !variable {
			continue
		}
		length, makeErr := state.manager.MkSequenceLength(expression)
		if makeErr != nil {
			return false, makeErr
		}
		value, _, exact, valueErr := arithmetic.fixedPolynomialValue(length, fixed)
		if valueErr != nil {
			return false, valueErr
		}
		if !exact {
			lengthNode := state.context.enodes[length]
			if lengthNode != nil {
				text, integer, numeral, numeralErr := state.manager.ArithmeticNumeralText(lengthNode.Root().Owner())
				if numeralErr != nil {
					return false, numeralErr
				}
				if integer && numeral {
					if parsed, parsedOK := new(big.Rat).SetString(text); parsedOK {
						value, exact = parsed, true
					}
				}
			}
		}
		if !exact || !value.IsInt() || value.Sign() < 0 || !value.Num().IsUint64() {
			continue
		}
		count := value.Num().Uint64()
		if count > smtSequenceMaximumFixedDecomposition {
			continue
		}
		sort, makeErr := state.manager.GetSort(expression)
		if makeErr != nil {
			return false, makeErr
		}
		elementSort, sequence, makeErr := state.manager.SequenceSortElement(sort)
		if makeErr != nil || !sequence {
			if makeErr != nil {
				return false, makeErr
			}
			return false, ast.ErrASTDomain
		}
		parts := make([]ast.ASTHandle, 0, int(count))
		internal := make([]ast.ASTHandle, 0, int(count)*2)
		for index := uint64(0); index < count; index++ {
			position, positionErr := t.integer(index)
			if positionErr != nil {
				return false, positionErr
			}
			element, positionErr := t.relationSkolem("seq.nth", []ast.ASTHandle{expression, position}, elementSort)
			if positionErr != nil {
				return false, positionErr
			}
			unit, positionErr := state.manager.MkSequenceUnit(element)
			if positionErr != nil {
				return false, positionErr
			}
			parts = append(parts, unit)
			internal = append(internal, element, unit)
		}
		word, makeErr := t.makeWord(parts, sort)
		if makeErr != nil {
			return false, makeErr
		}
		lengthValue, makeErr := t.integer(count)
		if makeErr != nil {
			return false, makeErr
		}
		fixedLength, makeErr := state.manager.MkEq(length, lengthValue)
		if makeErr != nil {
			return false, makeErr
		}
		definition, makeErr := state.manager.MkEq(expression, word)
		if makeErr != nil {
			return false, makeErr
		}
		constraint, makeErr := state.manager.MkImplies(fixedLength, definition)
		if makeErr != nil {
			return false, makeErr
		}
		if _, found := t.axioms.set[constraint]; found {
			continue
		}
		internal = append(internal, word)
		t.trackInternalTerms(internal...)
		return true, t.axioms.Push(constraint)
	}
	return false, nil
}
