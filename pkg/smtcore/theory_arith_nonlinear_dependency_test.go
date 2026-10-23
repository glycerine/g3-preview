package smtcore

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/simplex"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func smtTestIntervalBound(t *testing.T, solver *simplex.Solver, column uint32, lower bool, value int64, literal int32) {
	t.Helper()
	_, err := smtTightenExplainedSimplexBound(solver, column, lower, simplex.RationalInfinitesimal(big.NewRat(value, 1)), []simplex.BoundDependency{{Literal: literal, Coefficient: *big.NewRat(-7, 3)}})
	astTestMust(t, err)
}

func TestSMTRejectedNonlinearCandidateKeepsLogicalContextConsistent(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		t.Run(fmt.Sprint(proofs), func(t *testing.T) {
			m, cfg := smtModelGeneratorFixture(t)
			if proofs {
				astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
			}
			s, err := NewSMTSolver(m, nil, "QF_NRA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, s.Close()) })
			s.kernel.preprocess = false
			family, err := m.ArithmeticFamilyID()
			astTestMust(t, err)
			sort, err := m.MkFamilySort(family, 0, nil)
			astTestMust(t, err)
			x, err := m.MkConst(astTestSymbol(t, m, "candidate-dependency-x"), sort)
			astTestMust(t, err)
			y, err := m.MkConst(astTestSymbol(t, m, "candidate-dependency-y"), sort)
			astTestMust(t, err)
			product, err := m.MkArithmeticMul(x, y)
			astTestMust(t, err)
			var formulas []ast.ASTHandle
			for _, entry := range []struct {
				term  ast.ASTHandle
				op    int32
				value int64
			}{{x, smtArithOpGE, 1}, {x, smtArithOpLE, 3}, {y, smtArithOpGE, 1}, {y, smtArithOpLE, 3}, {product, smtArithOpLE, 5}} {
				value, err := m.ArithmeticNumeralRat(big.NewRat(entry.value, 1), false)
				astTestMust(t, err)
				formula, err := m.MkFamilyApp(family, entry.op, nil, []ast.ASTHandle{entry.term, value})
				astTestMust(t, err)
				ref, err := m.NewRef(formula)
				astTestMust(t, err)
				t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
				formulas = append(formulas, formula)
			}
			for repeat := 0; repeat < 2; repeat++ {
				astTestMust(t, s.Push())
				for _, formula := range formulas {
					astTestMust(t, s.Assert(formula))
				}
				status, err := s.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.True {
					t.Fatal("consistent logical bounds", status)
				}
				theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
				xColumn, err := theory.ensureVariable(x)
				astTestMust(t, err)
				yColumn, err := theory.ensureVariable(y)
				astTestMust(t, err)
				before := theory.statistics.conflicts
				rejected, status, err := theory.buildSimplexWithValues(map[uint32]*big.Rat{xColumn: big.NewRat(3, 1), yColumn: big.NewRat(3, 1)})
				astTestMust(t, err)
				if status != lbool.False || rejected == nil {
					t.Fatal("invalid candidate not rejected", status)
				}
				if theory.state.inconsistent || theory.statistics.conflicts != before {
					t.Fatal("rejected candidate became a logical conflict")
				}
				if _, known := rejected.BoundConflictDependencies(); known {
					t.Fatal("candidate values acquired a logical explanation")
				}
				status, err = s.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.True {
					t.Fatal("candidate rejection poisoned later check", status)
				}
				astTestMust(t, s.Pop(1))
			}
		})
	}
}

func smtTestPlainIntervalPremises(t *testing.T, solver *simplex.Solver, column uint32, lower bool, literals ...int32) {
	t.Helper()
	dependencies, known := solver.BoundDependencies(column, lower)
	if !known || len(dependencies) != len(literals) {
		t.Fatal("interval premises", column, lower, dependencies, known, literals)
	}
	want := make(map[int32]bool)
	for _, literal := range literals {
		want[literal] = true
	}
	for _, dependency := range dependencies {
		if !want[dependency.Literal] || dependency.Coefficient.Cmp(big.NewRat(1, 1)) != 0 {
			t.Fatal("plain interval premise or coefficient", dependency)
		}
		delete(want, dependency.Literal)
	}
}

func TestSMTNonlinearProductAndPowerEndpointExplanations(t *testing.T) {
	theory := &SMTLinearArithmeticTheory{}
	solver := simplex.NewSolver(nil)
	for _, bound := range []struct {
		column  uint32
		lower   bool
		value   int64
		literal int32
	}{{1, true, -2, 10}, {1, false, 3, 12}, {2, true, 4, 14}, {2, false, 5, 16}, {3, true, 99, 18}} {
		smtTestIntervalBound(t, solver, bound.column, bound.lower, bound.value, bound.literal)
	}
	var manager smtArithmeticBoundDependencyManager
	x, err := theory.nonlinearColumnInterval(solver, 1, &manager)
	astTestMust(t, err)
	y, err := theory.nonlinearColumnInterval(solver, 2, &manager)
	astTestMust(t, err)
	product := x.clone()
	astTestMust(t, product.mul(y))
	changed, conflict := theory.updateNonlinearBounds(solver, 0, product)
	if !changed || conflict {
		t.Fatal(changed, conflict)
	}
	smtTestPlainIntervalPremises(t, solver, 0, true, 10, 14, 16)
	smtTestPlainIntervalPremises(t, solver, 0, false, 12, 14, 16)
	lo, _ := solver.Lower(0)
	hi, _ := solver.Upper(0)
	if lo.Main.Cmp(big.NewRat(-10, 1)) != 0 || hi.Main.Cmp(big.NewRat(15, 1)) != 0 {
		t.Fatal(lo, hi)
	}
	x.power(2)
	changed, conflict = theory.updateNonlinearBounds(solver, 4, x)
	if !changed || conflict {
		t.Fatal(changed, conflict)
	}
	// A square is nonnegative without any bound premise. Its finite upper
	// endpoint uses both original endpoints to cover both signs.
	smtTestPlainIntervalPremises(t, solver, 4, true)
	smtTestPlainIntervalPremises(t, solver, 4, false, 10, 12)
	manager.Reset()
	// Installed explanations own their flattened premises after the temporary
	// interval dependency graph is released.
	smtTestPlainIntervalPremises(t, solver, 0, true, 10, 14, 16)
}

func TestSMTNonlinearUnknownEndpointDoesNotBecomeUnconditional(t *testing.T) {
	theory := &SMTLinearArithmeticTheory{}
	solver := simplex.NewSolver(nil)
	astTestMust(t, solver.SetLower(1, simplex.RationalInfinitesimal(big.NewRat(2, 1))))
	astTestMust(t, solver.SetUpper(1, simplex.RationalInfinitesimal(big.NewRat(3, 1))))
	var manager smtArithmeticBoundDependencyManager
	interval, err := theory.nonlinearColumnInterval(solver, 1, &manager)
	astTestMust(t, err)
	if interval.lowerDep == nil || interval.upperDep == nil {
		t.Fatal("unknown finite endpoints lost dependency leaves")
	}
	interval.power(2)
	changed, conflict := theory.updateNonlinearBounds(solver, 2, interval)
	if !changed || conflict {
		t.Fatal(changed, conflict)
	}
	for _, lower := range []bool{true, false} {
		if _, known := solver.BoundDependencies(2, lower); known {
			t.Fatal("candidate endpoint became a logical premise")
		}
	}
}

func TestSMTLinearMonomialZeroFactorExplainsBothBounds(t *testing.T) {
	theory := &SMTLinearArithmeticTheory{nonlinearColumns: []uint32{0}, nonlinearFactors: map[uint32][]uint32{0: {1, 2, 3}}}
	solver := simplex.NewSolver(nil)
	smtTestIntervalBound(t, solver, 1, true, 2, 10)
	smtTestIntervalBound(t, solver, 1, false, 2, 12)
	smtTestIntervalBound(t, solver, 2, true, 0, 14)
	smtTestIntervalBound(t, solver, 2, false, 0, 16)
	conflict, err := theory.propagateLinearMonomials(solver)
	astTestMust(t, err)
	if conflict {
		t.Fatal("consistent zero product rejected")
	}
	// Source discards earlier fixed factors once a zero factor suffices.
	smtTestPlainIntervalPremises(t, solver, 0, true, 14, 16)
	smtTestPlainIntervalPremises(t, solver, 0, false, 14, 16)
}

func TestSMTLinearMonomialAffineRowRetainsFixedFactorPremises(t *testing.T) {
	theory := &SMTLinearArithmeticTheory{nonlinearColumns: []uint32{0}, nonlinearFactors: map[uint32][]uint32{0: {1, 2}}}
	solver := simplex.NewSolver(nil)
	smtTestIntervalBound(t, solver, 1, true, 2, 10)
	smtTestIntervalBound(t, solver, 1, false, 2, 12)
	smtTestIntervalBound(t, solver, 2, true, 3, 14)
	smtTestIntervalBound(t, solver, 2, false, 4, 16)
	base := solver.NumVariables()
	conflict, err := theory.propagateLinearMonomials(solver)
	astTestMust(t, err)
	if conflict {
		t.Fatal("consistent affine product rejected")
	}
	smtTestPlainIntervalPremises(t, solver, base, true, 10, 12)
	smtTestPlainIntervalPremises(t, solver, base, false, 10, 12)
	conflict, err = theory.propagateArithmeticRowBounds(solver)
	astTestMust(t, err)
	if conflict {
		t.Fatal("consistent affine product bounds rejected")
	}
	for _, lower := range []bool{true, false} {
		dependencies, known := solver.BoundDependencies(0, lower)
		if !known {
			t.Fatal("affine product lost logical premises")
		}
		found := make(map[int32]bool)
		for _, dependency := range dependencies {
			found[dependency.Literal] = true
		}
		if !found[10] || !found[12] {
			t.Fatal("fixed factor became unconditional", dependencies)
		}
	}
}
