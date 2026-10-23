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

func (t *SMTSequenceTheory) sequenceWordsCanBeEqual(left, right []ast.ASTHandle) (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	index := 0
	for index < len(left) && index < len(right) {
		distinct, exact, distinctErr := t.concreteDistinct(left[index], right[index])
		if distinctErr != nil {
			return false, distinctErr
		}
		if exact && distinct {
			return false, nil
		}
		equal, _ := t.equalWordParts(left[index], right[index], nil)
		if !equal {
			break
		}
		index++
	}
	if index == len(right) {
		left, right = right, left
	}
	if index == len(left) && index < len(right) {
		for _, part := range right[index:] {
			if _, unit, unitErr := state.manager.SequenceUnitArgument(part); unitErr != nil {
				return false, unitErr
			} else if unit {
				return false, nil
			}
		}
	}
	return true, nil
}

func (t *SMTSequenceTheory) branchVariablePrefixEquality(left, right []ast.ASTHandle, dependency *smtSequenceDependency) (bool, error) {
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
		state, stateErr := t.State()
		if stateErr != nil {
			return false, stateErr
		}
		allUnits := true
		for _, part := range right {
			if _, unit, unitErr := state.manager.SequenceUnitArgument(part); unitErr != nil {
				return false, unitErr
			} else if !unit {
				allUnits = false
				break
			}
		}
		if variable && (uncovered || allUnits) {
			sort, sortErr := state.manager.GetSort(left[0])
			if sortErr != nil {
				return false, sortErr
			}
			empty, makeErr := state.manager.MkSequenceEmpty(sort)
			if makeErr != nil {
				return false, makeErr
			}
			possible, possibleErr := t.sequenceWordsCanBeEqual(left[1:], right)
			if possibleErr != nil {
				return false, possibleErr
			}
			if possible {
				assignment, assumeErr := t.assumeSequenceEquality(left[0], empty)
				if assumeErr != nil {
					return false, assumeErr
				}
				if assignment != lbool.False {
					return true, nil
				}
			}
			candidates := []ast.ASTHandle{empty}
			for index, part := range right {
				if part == left[0] {
					break
				}
				possible, possibleErr = t.sequenceWordsCanBeEqual(left[1:], right[index+1:])
				if possibleErr != nil {
					return false, possibleErr
				}
				if !possible {
					continue
				}
				prefix, prefixErr := t.makeWord(right[:index+1], sort)
				if prefixErr != nil {
					return false, prefixErr
				}
				assignment, assumeErr := t.assumeSequenceEquality(left[0], prefix)
				if assumeErr != nil {
					return false, assumeErr
				}
				if assignment != lbool.False {
					return true, nil
				}
				candidates = append(candidates, prefix)
			}
			if allUnits {
				conflictDependency := dependency
				for _, candidate := range candidates {
					equality, equalityErr := state.manager.MkEq(left[0], candidate)
					if equalityErr != nil {
						return false, equalityErr
					}
					if equalityErr = state.Internalize(equality, false); equalityErr != nil {
						return false, equalityErr
					}
					literal, equalityErr := state.Literal(equality)
					if equalityErr != nil {
						return false, equalityErr
					}
					assignment, equalityErr := state.Assignment(literal)
					if equalityErr != nil {
						return false, equalityErr
					}
					if assignment != lbool.False {
						data, dataErr := state.BooleanData(literal.Var())
						if dataErr != nil {
							return false, dataErr
						}
						data.SetTrueFirst()
						return true, nil
					}
					conflictDependency = t.dependencies.Join(conflictDependency,
						t.dependencies.Leaf(smtSequenceAssumption{literal: literal.Negated()}))
				}
				if conflictErr := t.setDependencyConflict(conflictDependency); conflictErr != nil {
					return false, conflictErr
				}
				return true, nil
			}
		}
		left, right = right, left
	}
	return false, nil
}
