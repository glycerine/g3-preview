// Source: src/smt/theory_arith_core.h row bound propagation and fixed-value
// equality propagation. Only unconditional replay bounds justify equalities.
package smtcore

import (
	"errors"
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
		return true, t.derivedBoundConflict(solver)
	}
	alreadyFixed := make(map[uint32]bool)
	for _, column := range t.simplexVariables {
		_, alreadyFixed[column] = smtFixedSimplexColumn(solver, column)
	}
	if conflict, err := t.propagateArithmeticRowBounds(solver); err != nil {
		return false, err
	} else if conflict {
		return true, t.derivedBoundConflict(solver)
	}
	status, err = solver.MakeFeasible()
	if err != nil || status == lbool.Undef {
		return false, err
	}
	if status == lbool.False {
		return true, t.derivedBoundConflict(solver)
	}
	progress := false
	for _, column := range t.simplexVariables {
		value, fixed := smtFixedSimplexColumn(solver, column)
		if !fixed || alreadyFixed[column] {
			continue
		}
		bound := simplex.RationalInfinitesimal(value)
		t.derivedBounds = append(t.derivedBounds,
			smtArithmeticReplayBound{column: column, lower: true, value: bound.Clone(), explanation: smtArithmeticCopyBoundExplanation(solver, column, true)},
			smtArithmeticReplayBound{column: column, lower: false, value: bound.Clone(), explanation: smtArithmeticCopyBoundExplanation(solver, column, false)})
		progress = true
	}
	if progress {
		t.solver = solver
		t.state.theoryProgress++
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
			merged, err := t.propagateFixedEquality(solver, first, second)
			if err != nil || merged {
				return true, err
			}
		}
	}
	return progress, nil
}

// Source theory_arith_eq.h fixed_var_eh pushes four endpoint justifications
// in this order, using zero coefficients for its equality proof rule.
func (t *SMTLinearArithmeticTheory) propagateFixedEquality(solver *simplex.Solver, first, second uint32) (progress bool, result error) {
	if !t.canPropagateArithmeticEqualities() {
		return false, nil
	}
	proofs, err := t.state.manager.ProofsEnabled()
	if err != nil {
		return false, err
	}
	uses := []simplex.ArithmeticBoundUse{
		{Column: first, Lower: true},
		{Column: second, Lower: false},
		{Column: second, Lower: true},
		{Column: first, Lower: false},
	}
	antecedents, known, err := smtArithmeticExplainBounds(solver, uses, proofs, t.equalityBounds)
	if err != nil {
		return false, err
	}
	if !known {
		// Logical replay may contain an endpoint whose explanation has not
		// yet been ported. Preserve the existing conservative justification.
		antecedents, err := t.allArithmeticAntecedents()
		if err != nil {
			return false, err
		}
		return t.propagateEqualityWithParameters(t.state, first, second, antecedents.literals, nil, antecedents.equalities)
	}
	parameters, err := antecedents.parameters(t.state.manager, "eq-propagate")
	if err != nil {
		return false, err
	}
	defer func() { result = errors.Join(result, smtArithmeticReleaseParameters(t.state.manager, parameters)) }()
	return t.propagateEqualityWithParameters(t.state, first, second, antecedents.literals, parameters, antecedents.equalities)
}

// Derive bounds through canonical equality and comparison rows. The caller
// owns replay and records only bounds supported by unconditional assertions.
func (t *SMTLinearArithmeticTheory) propagateArithmeticRowBounds(solver *simplex.Solver) (bool, error) {
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
			for _, lower := range []bool{true, false} {
				var bound simplex.Infinitesimal
				var uses []simplex.ArithmeticBoundUse
				finite := true
				for _, other := range entries {
					if other.Variable == target.Variable {
						continue
					}
					coefficient := new(big.Rat).Neg(new(big.Rat).Quo(other.Coefficient, target.Coefficient))
					useLower := lower == (coefficient.Sign() > 0)
					endpoint, exists := solver.Upper(other.Variable)
					if useLower {
						endpoint, exists = solver.Lower(other.Variable)
					}
					if !exists {
						finite = false
						break
					}
					// Linear row arithmetic operates on inf_numeral, preserving the
					// coefficient and sign of epsilon. Rational interval projection can
					// strengthen a bound such as 5-epsilon to 5 and invent an equality.
					bound.Infinity.Add(&bound.Infinity, new(big.Rat).Mul(coefficient, &endpoint.Infinity))
					bound.Main.Add(&bound.Main, new(big.Rat).Mul(coefficient, &endpoint.Main))
					bound.Epsilon.Add(&bound.Epsilon, new(big.Rat).Mul(coefficient, &endpoint.Epsilon))
					uses = append(uses, simplex.ArithmeticBoundUse{Column: other.Variable, Lower: useLower, Coefficient: *coefficient})
				}
				if !finite {
					continue
				}
				if t.integer || t.integerColumns[target.Variable] {
					integer := floorSMTRational(&bound.Main)
					if lower && (!bound.Main.IsInt() || bound.Epsilon.Sign() > 0) {
						integer.Add(integer, big.NewInt(1))
					} else if !lower && bound.Main.IsInt() && bound.Epsilon.Sign() < 0 {
						integer.Sub(integer, big.NewInt(1))
					}
					bound.Infinity.SetInt64(0)
					bound.Main.SetInt(integer)
					bound.Epsilon.SetInt64(0)
				}
				explanation := smtArithmeticWeightedBoundExplanation(solver, uses)
				if _, err := smtTightenReplaySimplexBound(solver, target.Variable, lower, bound, explanation); err != nil {
					t.recordArithmeticBoundConflict(solver, target.Variable, lower, explanation)
					return true, nil
				}
			}
		}
	}
	return false, nil
}
