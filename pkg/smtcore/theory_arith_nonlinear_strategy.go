// Source: src/smt/theory_arith_nl.h process_non_linear's rotating strategies
// and scoped derived nonlinear bounds. Replay bounds have logical premises;
// temporary candidate-model assignments never justify logical conflicts.
package smtcore

import (
	"github.com/glycerine/g3/pkg/math/simplex"
	"github.com/glycerine/g3/pkg/util/lbool"
)

type smtArithmeticReplayBound struct {
	column uint32
	lower  bool
	value  simplex.Infinitesimal
}

func smtTightenSimplexBound(solver *simplex.Solver, column uint32, lower bool, value simplex.Infinitesimal) (bool, error) {
	if lower {
		if old, found := solver.Lower(column); found && old.Cmp(&value) >= 0 {
			return false, nil
		}
		return true, solver.SetLower(column, value)
	}
	if old, found := solver.Upper(column); found && old.Cmp(&value) <= 0 {
		return false, nil
	}
	return true, solver.SetUpper(column, value)
}

func (t *SMTLinearArithmeticTheory) derivedBoundConflict() error {
	literals := make([]SMTLiteral, 0, len(t.assertions))
	for _, assertion := range t.assertions {
		literals = append(literals, smtLinearAssertionLiteral(&t.atoms[assertion.atom], assertion))
	}
	t.statistics.conflicts++
	return t.state.SetTheoryConflict(t.FamilyID(), literals)
}

func (t *SMTLinearArithmeticTheory) propagateNonlinearStrategyBounds() (bool, error) {
	solver, status, err := t.buildSimplex()
	if err != nil || status == lbool.Undef {
		return false, err
	}
	if status == lbool.False {
		return true, t.derivedBoundConflict()
	}
	before := make(map[uint32][2]smtArithmeticReplayBound, len(t.simplexVariables))
	valid := make(map[uint32][2]bool, len(t.simplexVariables))
	for _, column := range t.simplexVariables {
		lower, hasLower := solver.Lower(column)
		upper, hasUpper := solver.Upper(column)
		before[column] = [2]smtArithmeticReplayBound{{column, true, lower}, {column, false, upper}}
		valid[column] = [2]bool{hasLower, hasUpper}
	}
	conflict, err := t.propagateArithmeticRowBounds(solver)
	if err != nil {
		return false, err
	}
	if !conflict {
		_, conflict, err = t.propagateNonlinearBounds(solver)
	}
	if err != nil {
		return false, err
	}
	if !conflict {
		_, conflict, err = t.propagateArithmeticDefinitionBounds(solver)
		if err != nil {
			return false, err
		}
	}
	if conflict {
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
		for index, lower := range []bool{true, false} {
			value, hasBound := solver.Upper(column)
			if lower {
				value, hasBound = solver.Lower(column)
			}
			if !hasBound {
				continue
			}
			old := before[column][index]
			if valid[column][index] && (lower && value.Cmp(&old.value) <= 0 || !lower && value.Cmp(&old.value) >= 0) {
				continue
			}
			t.derivedBounds = append(t.derivedBounds, smtArithmeticReplayBound{column, lower, value.Clone()})
			progress = true
		}
	}
	if progress {
		t.solver = solver
		// The Go search loop's no-progress guard must see numeric facts,
		// even when they introduce no Boolean atom, clause, or equality.
		t.state.theoryProgress++
	}
	return progress, nil
}

func (t *SMTLinearArithmeticTheory) runNonlinearStrategies() (bool, error) {
	start := t.nonlinearStrategy
	for {
		var progress bool
		var err error
		switch t.nonlinearStrategy {
		case 0:
			progress, err = t.propagateNonlinearStrategyBounds()
		case 1:
			progress, err = t.checkCrossNestedConsistency()
		case 2:
			if t.nonlinearGB {
				progress, err = t.checkNonlinearGrobner()
			}
		case 3:
			if t.nonlinearBranching {
				progress, err = t.branchNonlinearIntegerVariable()
			}
		}
		if err != nil {
			return false, err
		}
		t.nonlinearStrategy = (t.nonlinearStrategy + 1) % 4
		if progress {
			return true, nil
		}
		if t.nonlinearStrategy == start {
			return false, nil
		}
	}
}
