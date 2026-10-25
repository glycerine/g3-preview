// Source: src/smt/theory_seq.cpp solve_nqs and solve_ne retained word
// expansion, equality reduction and conflict case.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func (t *SMTSequenceTheory) checkReducedDisequations() (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	for index := 0; index < t.disequations.Len(); index++ {
		originalLeft, originalRight, leftCases, rightCases, literals, dependency, entryErr := t.disequations.At(index)
		if entryErr != nil {
			return false, entryErr
		}
		allLiteralsTrue := true
		for _, literal := range literals {
			assignment, assignmentErr := state.Assignment(literal)
			if assignmentErr != nil {
				return false, assignmentErr
			}
			if assignment == lbool.False {
				allLiteralsTrue = false
				break
			}
			if assignment != lbool.True {
				allLiteralsTrue = false
				break
			}
			dependency = t.dependencies.Join(dependency, t.dependencies.Leaf(smtSequenceAssumption{literal: literal}))
		}
		if !allLiteralsTrue {
			continue
		}
		allCasesEqual := true
		satisfied := false
		var residualLeft, residualRight []ast.ASTHandle
		for caseIndex := range leftCases {
			left, updated, expandErr := t.expandWord(leftCases[caseIndex], dependency)
			if expandErr != nil {
				return false, expandErr
			}
			right, updated, expandErr := t.expandWord(rightCases[caseIndex], updated)
			if expandErr != nil {
				return false, expandErr
			}
			left, right, updated, _, unequal, reduceErr := t.cancelWordBoundaries(left, right, updated)
			if reduceErr != nil {
				return false, reduceErr
			}
			dependency = updated
			if unequal || (len(left) == 0 && len(right) != 0 && smtSequenceAllStrings(right, t)) || (len(right) == 0 && len(left) != 0 && smtSequenceAllStrings(left, t)) {
				allCasesEqual = false
				satisfied = true
				break
			}
			if len(left) != 0 || len(right) != 0 {
				allCasesEqual = false
				if len(leftCases) == 1 && len(literals) == 0 {
					residualLeft = left
					residualRight = right
				}
				break
			}
		}
		if allCasesEqual {
			if entryErr = t.setDependencyConflict(dependency); entryErr != nil {
				return false, entryErr
			}
			return true, nil
		}
		if satisfied || residualLeft == nil && residualRight == nil {
			continue
		}
		sort, sortErr := state.manager.GetSort(originalLeft)
		if sortErr != nil {
			return false, sortErr
		}
		leftWord, makeErr := t.makeWord(residualLeft, sort)
		if makeErr != nil {
			return false, makeErr
		}
		rightWord, makeErr := t.makeWord(residualRight, sort)
		if makeErr != nil {
			return false, makeErr
		}
		originalEquality, makeErr := state.manager.MkEq(originalLeft, originalRight)
		if makeErr != nil {
			return false, makeErr
		}
		residualEquality, makeErr := state.manager.MkEq(leftWord, rightWord)
		if makeErr != nil {
			return false, makeErr
		}
		if originalEquality == residualEquality {
			continue
		}
		if makeErr = state.Internalize(residualEquality, true); makeErr != nil {
			return false, makeErr
		}
		literal, makeErr := state.Literal(residualEquality)
		if makeErr != nil {
			return false, makeErr
		}
		progress, makeErr := t.propagateSequenceLiteral(literal.Negated(), dependency)
		if makeErr != nil || progress {
			return progress, makeErr
		}
	}
	return false, nil
}

// Source propagate_lit retains both literal and enode-equality antecedents.
// A cancellation under established equalities is not an unconditional lemma.
func (t *SMTSequenceTheory) propagateSequenceLiteral(literal SMTLiteral, dependency *smtSequenceDependency) (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	value, err := state.Assignment(literal)
	if err != nil || value == lbool.True {
		return false, err
	}
	if literal == SMTFalseLiteral {
		return true, t.setDependencyConflict(dependency)
	}
	expression, err := state.BoolExpression(literal.Var())
	if err != nil {
		return false, err
	}
	if err = state.markRelevant(expression); err != nil {
		return false, err
	}
	literals, equalities := t.dependencyEvidence(dependency)
	for _, antecedent := range literals {
		assigned, err := state.Assignment(antecedent)
		if err != nil || assigned != lbool.True {
			return false, err
		}
	}
	owner, err := NewSMTExtTheoryPropagationJustification(state.manager, t.FamilyID(), literals, equalities, literal, nil)
	if err != nil {
		return false, err
	}
	wrapped, err := NewSMTBooleanJustification(owner)
	if err != nil {
		return false, errors.Join(err, owner.Release(state.manager))
	}
	state.ownedJustifications = append(state.ownedJustifications, owner)
	if err = state.AssignLiteral(literal, wrapped); err != nil {
		return false, err
	}
	state.theoryProgress++
	return true, nil
}
