// Source: src/smt/theory_seq.cpp is_binary_eq, solve_binary_eq and
// branch_binary_variable for the retained one-character boundary case.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

func (t *SMTSequenceTheory) branchBinaryUnitEquation(left, right []ast.ASTHandle) (bool, error) {
	for pass := 0; pass < 2; pass++ {
		if len(left) == 2 && len(right) == 2 {
			leftVariable, err := t.sequenceVariable(left[0])
			if err != nil {
				return false, err
			}
			rightVariable, err := t.sequenceVariable(right[1])
			if err != nil {
				return false, err
			}
			state, stateErr := t.State()
			if stateErr != nil {
				return false, stateErr
			}
			leftText, leftString, textErr := state.manager.SequenceStringValue(left[1])
			if textErr != nil {
				return false, textErr
			}
			rightText, rightString, textErr := state.manager.SequenceStringValue(right[0])
			if textErr != nil {
				return false, textErr
			}
			if leftVariable && rightVariable && len(leftText) == 1 && leftString && len(rightText) == 1 && rightString {
				sort, sortErr := state.manager.GetSort(left[0])
				if sortErr != nil {
					return false, sortErr
				}
				tail, makeErr := t.relationSkolem("seq.binary.tail", []ast.ASTHandle{left[0], right[1], left[1], right[0]}, sort)
				if makeErr != nil {
					return false, makeErr
				}
				leftValue, makeErr := state.manager.MkSequenceConcat(right[0], tail)
				if makeErr != nil {
					return false, makeErr
				}
				rightValue, makeErr := state.manager.MkSequenceConcat(tail, left[1])
				if makeErr != nil {
					return false, makeErr
				}
				leftEquality, makeErr := state.manager.MkEq(left[0], leftValue)
				if makeErr != nil {
					return false, makeErr
				}
				rightEquality, makeErr := state.manager.MkEq(right[1], rightValue)
				if makeErr != nil {
					return false, makeErr
				}
				alternatives, makeErr := state.manager.MkAnd(leftEquality, rightEquality)
				if makeErr != nil {
					return false, makeErr
				}
				if leftText == rightText {
					empty, emptyErr := state.manager.MkSequenceEmpty(sort)
					if emptyErr != nil {
						return false, emptyErr
					}
					leftEmpty, emptyErr := state.manager.MkEq(left[0], empty)
					if emptyErr != nil {
						return false, emptyErr
					}
					rightEmpty, emptyErr := state.manager.MkEq(right[1], empty)
					if emptyErr != nil {
						return false, emptyErr
					}
					bothEmpty, emptyErr := state.manager.MkAnd(leftEmpty, rightEmpty)
					if emptyErr != nil {
						return false, emptyErr
					}
					alternatives, makeErr = state.manager.MkOr(bothEmpty, alternatives)
					if makeErr != nil {
						return false, makeErr
					}
				}
				leftWord, makeErr := t.makeWord(left, sort)
				if makeErr != nil {
					return false, makeErr
				}
				rightWord, makeErr := t.makeWord(right, sort)
				if makeErr != nil {
					return false, makeErr
				}
				equality, makeErr := state.manager.MkEq(leftWord, rightWord)
				if makeErr != nil {
					return false, makeErr
				}
				constraint, makeErr := state.manager.MkImplies(equality, alternatives)
				if makeErr != nil {
					return false, makeErr
				}
				if _, found := t.axioms.set[constraint]; found {
					return false, nil
				}
				return true, t.axioms.Push(constraint)
			}
		}
		left, right = right, left
	}
	return false, nil
}
