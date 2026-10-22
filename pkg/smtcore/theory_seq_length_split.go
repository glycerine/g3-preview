// Source: src/smt/theory_seq.cpp reduce_length_eq and reduce_length.
package smtcore

import (
	"math"
	"math/big"

	"github.com/glycerine/g3/pkg/ast"
)

func (t *SMTSequenceTheory) exactSequenceLength(expression ast.ASTHandle) (uint64, bool, error) {
	state, err := t.State()
	if err != nil {
		return 0, false, err
	}
	if text, matched, textErr := state.manager.SequenceStringValue(expression); textErr != nil || matched {
		return uint64(len(text)), matched, textErr
	}
	if empty, emptyErr := state.manager.IsSequenceEmpty(expression); emptyErr != nil || empty {
		return 0, empty, emptyErr
	}
	if _, unit, unitErr := state.manager.SequenceUnitArgument(expression); unitErr != nil || unit {
		return 1, unit, unitErr
	}
	length, err := state.manager.MkSequenceLength(expression)
	if err != nil {
		return 0, false, err
	}
	arithmeticFamily, err := state.manager.ArithmeticFamilyID()
	if err != nil {
		return 0, false, err
	}
	arithmetic, ok := state.Theory(arithmeticFamily).(*SMTLinearArithmeticTheory)
	if !ok {
		return 0, false, nil
	}
	value, _, exact, err := arithmetic.fixedPolynomialValue(length, arithmetic.fixedLinearAssignments())
	if err != nil {
		return 0, false, err
	}
	if !exact {
		if node := state.context.enodes[length]; node != nil {
			text, integer, numeral, numeralErr := state.manager.ArithmeticNumeralText(node.Root().Owner())
			if numeralErr != nil {
				return 0, false, numeralErr
			}
			if integer && numeral {
				value, exact = new(big.Rat).SetString(text)
			}
		}
	}
	if !exact || value == nil || !value.IsInt() || value.Sign() < 0 || !value.Num().IsUint64() {
		return 0, false, nil
	}
	return value.Num().Uint64(), true, nil
}

func (t *SMTSequenceTheory) fixedLengthPartition(parts []ast.ASTHandle, front bool) ([]uint64, bool, error) {
	// The final part remains on the residual side of every proper split, so its
	// length need not be fixed.
	lengths := make([]uint64, len(parts)-1)
	for index := range lengths {
		partIndex := index
		if !front {
			partIndex = len(parts) - 1 - index
		}
		length, exact, err := t.exactSequenceLength(parts[partIndex])
		if err != nil || !exact {
			return nil, exact, err
		}
		lengths[index] = length
	}
	return lengths, true, nil
}

func smtSequenceEqualPartition(left, right []uint64) (int, int, bool) {
	var leftSum, rightSum uint64
	for leftCount, rightCount := 0, 0; leftCount < len(left) || rightCount < len(right); {
		if leftSum <= rightSum && leftCount < len(left) {
			if left[leftCount] > math.MaxUint64-leftSum {
				return 0, 0, false
			}
			leftSum += left[leftCount]
			leftCount++
		} else if rightCount < len(right) {
			if right[rightCount] > math.MaxUint64-rightSum {
				return 0, 0, false
			}
			rightSum += right[rightCount]
			rightCount++
		} else {
			break
		}
		if leftSum == rightSum && leftSum != 0 && leftCount != 0 && rightCount != 0 {
			return leftCount, rightCount, true
		}
	}
	return 0, 0, false
}

func (t *SMTSequenceTheory) splitFixedLengthEquation(left, right []ast.ASTHandle) (bool, error) {
	if len(left) < 2 || len(right) < 2 {
		return false, nil
	}
	state, err := t.State()
	if err != nil {
		return false, err
	}
	sort, err := state.manager.GetSort(left[0])
	if err != nil {
		return false, err
	}
	for _, front := range []bool{true, false} {
		leftLengths, exact, lengthErr := t.fixedLengthPartition(left, front)
		if lengthErr != nil {
			return false, lengthErr
		}
		if !exact {
			continue
		}
		rightLengths, exact, lengthErr := t.fixedLengthPartition(right, front)
		if lengthErr != nil {
			return false, lengthErr
		}
		if !exact {
			continue
		}
		leftCount, rightCount, found := smtSequenceEqualPartition(leftLengths, rightLengths)
		if !found {
			continue
		}
		leftFirst, leftRest := left[:leftCount], left[leftCount:]
		rightFirst, rightRest := right[:rightCount], right[rightCount:]
		if !front {
			leftFirst, leftRest = left[len(left)-leftCount:], left[:len(left)-leftCount]
			rightFirst, rightRest = right[len(right)-rightCount:], right[:len(right)-rightCount]
		}
		leftPrefix, makeErr := t.makeWord(leftFirst, sort)
		if makeErr != nil {
			return false, makeErr
		}
		rightPrefix, makeErr := t.makeWord(rightFirst, sort)
		if makeErr != nil {
			return false, makeErr
		}
		leftRemainder, makeErr := t.makeWord(leftRest, sort)
		if makeErr != nil {
			return false, makeErr
		}
		rightRemainder, makeErr := t.makeWord(rightRest, sort)
		if makeErr != nil {
			return false, makeErr
		}
		leftWord, makeErr := t.makeWord(left, sort)
		if makeErr != nil {
			return false, makeErr
		}
		rightWord, makeErr := t.makeWord(right, sort)
		if makeErr != nil {
			return false, makeErr
		}
		wordEquality, makeErr := state.manager.MkEq(leftWord, rightWord)
		if makeErr != nil {
			return false, makeErr
		}
		leftLength, makeErr := state.manager.MkSequenceLength(leftPrefix)
		if makeErr != nil {
			return false, makeErr
		}
		rightLength, makeErr := state.manager.MkSequenceLength(rightPrefix)
		if makeErr != nil {
			return false, makeErr
		}
		lengthEquality, makeErr := state.manager.MkEq(leftLength, rightLength)
		if makeErr != nil {
			return false, makeErr
		}
		antecedent, makeErr := state.manager.MkAnd(wordEquality, lengthEquality)
		if makeErr != nil {
			return false, makeErr
		}
		prefixEquality, makeErr := state.manager.MkEq(leftPrefix, rightPrefix)
		if makeErr != nil {
			return false, makeErr
		}
		remainderEquality, makeErr := state.manager.MkEq(leftRemainder, rightRemainder)
		if makeErr != nil {
			return false, makeErr
		}
		consequence, makeErr := state.manager.MkAnd(prefixEquality, remainderEquality)
		if makeErr != nil {
			return false, makeErr
		}
		constraint, makeErr := state.manager.MkImplies(antecedent, consequence)
		if makeErr != nil {
			return false, makeErr
		}
		if _, exists := t.axioms.set[constraint]; exists {
			continue
		}
		return true, t.axioms.Push(constraint)
	}
	return false, nil
}
