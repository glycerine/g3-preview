// Source: src/smt/theory_arith_core.h row bound propagation and fixed-value
// equality propagation. Only unconditional replay bounds justify equalities.
package smtcore

import (
	"math/big"

	"github.com/glycerine/g3/pkg/math/simplex"
	"github.com/glycerine/g3/pkg/util/lbool"
)

// Derive intervals for each variable in a small canonical row. Keeping only
// newly fixed original columns avoids an infinite sequence of fractional
// improvements when no available bound atom requires the intermediate bounds.
func (t *SMTLinearArithmeticTheory) propagateFixedSharedValues() (bool, error) {
	var shared []uint32
	for variable, column := range t.simplexVariables {
		node, err := t.Enode(SMTTheoryVar(variable))
		if err != nil {
			return false, err
		}
		isShared, err := t.sharedArithmeticNode(node)
		if err != nil {
			return false, err
		}
		if isShared {
			shared = append(shared, column)
		}
	}
	if len(shared) < 2 {
		return false, nil
	}
	solver, status, err := t.buildSimplex()
	if err != nil || status == lbool.Undef {
		return false, err
	}
	if status == lbool.False {
		return true, t.derivedBoundConflict()
	}
	alreadyFixed := make(map[uint32]bool)
	for _, column := range t.simplexVariables {
		_, alreadyFixed[column] = smtFixedSimplexColumn(solver, column)
	}
	if conflict, err := t.propagateArithmeticRowBounds(solver); err != nil {
		return false, err
	} else if conflict {
		return true, t.derivedBoundConflict()
	}
	status, err = solver.MakeFeasible()
	if err != nil || status == lbool.Undef {
		return false, err
	}
	if status == lbool.False {
		return true, t.derivedBoundConflict()
	}
	progress := false
	for _, column := range t.simplexVariables {
		value, fixed := smtFixedSimplexColumn(solver, column)
		if !fixed || alreadyFixed[column] {
			continue
		}
		bound := simplex.RationalInfinitesimal(value)
		t.derivedBounds = append(t.derivedBounds, smtArithmeticReplayBound{column, true, bound.Clone()}, smtArithmeticReplayBound{column, false, bound.Clone()})
		progress = true
	}
	if progress {
		t.solver = solver
		t.state.theoryProgress++
	}
	literals := make([]SMTLiteral, 0, len(t.assertions))
	for _, assertion := range t.assertions {
		literals = append(literals, smtLinearAssertionLiteral(&t.atoms[assertion.atom], assertion))
	}
	for i, first := range shared {
		firstValue, fixed := smtFixedSimplexColumn(solver, first)
		if !fixed {
			continue
		}
		for _, second := range shared[i+1:] {
			secondValue, fixed := smtFixedSimplexColumn(solver, second)
			if !fixed || firstValue.Cmp(secondValue) != 0 {
				continue
			}
			merged, err := t.propagateEquality(t.state, first, second, literals)
			if err != nil || merged {
				return true, err
			}
		}
	}
	return progress, nil
}

// Derive bounds through canonical equality and comparison rows. The caller
// owns replay and records only bounds supported by unconditional assertions.
func (t *SMTLinearArithmeticTheory) propagateArithmeticRowBounds(solver *simplex.Solver) (bool, error) {
	var manager smtArithmeticBoundDependencyManager
	for column := uint32(0); column < solver.NumVariables(); column++ {
		row, basic := solver.BasicRow(column)
		if !basic {
			continue
		}
		entries, err := solver.RowEntries(row)
		if err != nil {
			return false, err
		}
		if len(entries) >= 128 {
			continue
		} // Source default max_lemma_size.
		for _, target := range entries {
			interval, err := newSMTOldIntervalPoint(&manager, new(big.Rat), nil, nil)
			if err != nil {
				return false, err
			}
			for _, other := range entries {
				if other.Variable == target.Variable {
					continue
				}
				value, err := t.nonlinearColumnInterval(solver, other.Variable, &manager)
				if err != nil {
					return false, err
				}
				scalar, err := newSMTOldIntervalPoint(&manager, other.Coefficient, nil, nil)
				if err != nil {
					return false, err
				}
				if err = value.mul(scalar); err != nil {
					return false, err
				}
				if err = interval.add(value); err != nil {
					return false, err
				}
			}
			coefficient := new(big.Rat).Quo(big.NewRat(-1, 1), target.Coefficient)
			scalar, err := newSMTOldIntervalPoint(&manager, coefficient, nil, nil)
			if err != nil {
				return false, err
			}
			if err = interval.mul(scalar); err != nil {
				return false, err
			}
			_, conflict := t.updateNonlinearBounds(solver, target.Variable, interval)
			if conflict {
				return true, nil
			}
		}
	}
	return false, nil
}
