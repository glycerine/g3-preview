// Source: src/smt/theory_seq.cpp branch_variable, find_branch_candidate and
// assume_equality retained equality case splitting for non-fixed words.
package smtcore

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func (t *SMTSequenceTheory) assumeSequenceEquality(left, right ast.ASTHandle) (lbool.LBool, error) {
	state, err := t.State()
	if err != nil {
		return lbool.Undef, err
	}
	if left == right {
		return lbool.True, nil
	}
	equality, err := state.manager.MkEq(left, right)
	if err != nil {
		return lbool.Undef, err
	}
	if err = state.Internalize(equality, false); err != nil {
		return lbool.Undef, err
	}
	literal, err := state.Literal(equality)
	if err != nil {
		return lbool.Undef, err
	}
	assignment, err := state.Assignment(literal)
	if err != nil || assignment != lbool.Undef {
		return assignment, err
	}
	if state.relevancy != nil {
		if err = state.relevancy.MarkRelevant(equality); err != nil {
			return lbool.Undef, err
		}
	} else if err = state.context.SetRelevant(equality, true); err != nil {
		return lbool.Undef, err
	}
	data, err := state.BooleanData(literal.Var())
	if err != nil {
		return lbool.Undef, err
	}
	data.SetTrueFirst()
	return lbool.Undef, nil
}

func (t *SMTSequenceTheory) wordHasUncoveredPart(parts []ast.ASTHandle) (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	for _, part := range parts {
		if variable, variableErr := t.sequenceVariable(part); variableErr != nil {
			return false, variableErr
		} else if variable {
			continue
		}
		if _, fixed, textErr := state.manager.SequenceStringValue(part); textErr != nil {
			return false, textErr
		} else if fixed {
			continue
		}
		if _, unit, unitErr := state.manager.SequenceUnitArgument(part); unitErr != nil {
			return false, unitErr
		} else if !unit {
			return true, nil
		}
	}
	return false, nil
}

func (t *SMTSequenceTheory) branchVariablePrefixEquality(left, right []ast.ASTHandle) (bool, error) {
	if len(left) == 0 || len(right) == 0 {
		return false, nil
	}
	for pass := 0; pass < 2; pass++ {
		variable, err := t.sequenceVariable(left[0])
		if err != nil {
			return false, err
		}
		uncovered, err := t.wordHasUncoveredPart(right)
		if err != nil {
			return false, err
		}
		if variable && uncovered {
			state, stateErr := t.State()
			if stateErr != nil {
				return false, stateErr
			}
			sort, sortErr := state.manager.GetSort(left[0])
			if sortErr != nil {
				return false, sortErr
			}
			empty, makeErr := state.manager.MkSequenceEmpty(sort)
			if makeErr != nil {
				return false, makeErr
			}
			assignment, assumeErr := t.assumeSequenceEquality(left[0], empty)
			if assumeErr != nil {
				return false, assumeErr
			}
			if assignment == lbool.Undef {
				return true, nil
			}
			for index, part := range right {
				if part == left[0] {
					break
				}
				prefix, prefixErr := t.makeWord(right[:index+1], sort)
				if prefixErr != nil {
					return false, prefixErr
				}
				assignment, assumeErr = t.assumeSequenceEquality(left[0], prefix)
				if assumeErr != nil {
					return false, assumeErr
				}
				if assignment == lbool.Undef {
					return true, nil
				}
			}
		}
		left, right = right, left
	}
	return false, nil
}
