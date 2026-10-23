package smtcore

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/simplex"
)

func smtNonlinearSelectionFixture(t *testing.T, powers []uint32) (*SMTLinearArithmeticTheory, *simplex.Solver, []uint32, uint32) {
	t.Helper()
	m, cfg := smtModelGeneratorFixture(t)
	s, err := NewSMTSolver(m, nil, "QF_NRA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	family, err := m.ArithmeticFamilyID()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 0, nil)
	astTestMust(t, err)
	theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
	columns := make([]uint32, len(powers))
	var factors []ast.ASTHandle
	for index, power := range powers {
		term, err := m.MkConst(astTestSymbol(t, m, string(rune('x'+index))), sort)
		astTestMust(t, err)
		columns[index], err = theory.ensureVariable(term)
		astTestMust(t, err)
		for repeat := uint32(0); repeat < power; repeat++ {
			factors = append(factors, term)
		}
	}
	product, err := m.MkFamilyApp(family, smtArithOpMul, nil, factors)
	astTestMust(t, err)
	handled, err := theory.InternalizeTerm(product)
	astTestMust(t, err)
	if !handled {
		t.Fatal("product was not internalized")
	}
	column, err := theory.ensureVariable(product)
	astTestMust(t, err)
	if theory.state.relevancy != nil {
		astTestMust(t, theory.state.relevancy.MarkRelevant(product))
	}
	return theory, simplex.NewSolver(nil), columns, column
}

func TestSMTNonlinearPropagationSelectsSourceDirections(t *testing.T) {
	for _, tc := range []struct {
		name      string
		powers    []uint32
		bounds    [][3]int64 // index (factor or final product), lower, upper
		cost      uint64
		changed   bool
		inspect   int
		wantLower string
		wantUpper string
	}{
		{"two-free-odd-factors", []uint32{1, 1}, [][3]int64{{2, 8, 12}}, 0, false, 0, "", ""},
		{"one-free-factor-free-product", []uint32{1, 1}, [][3]int64{{0, 2, 3}}, 0, false, 2, "", ""},
		{"free-even-power-upward-only", []uint32{2}, nil, 2, true, 1, "0", ""},
		{"bounded-factors-free-product-upward-only", []uint32{1, 1}, [][3]int64{{0, 2, 3}, {1, 4, 5}}, 12, true, 2, "8", "15"},
		{"one-free-factor-downward-only", []uint32{1, 1}, [][3]int64{{0, 2, 3}, {2, 8, 12}}, 4, true, 1, "8/3", "6"},
		{"odd-power-needs-unsupported-root", []uint32{1, 3}, [][3]int64{{0, 2, 3}, {2, 8, 12}}, 0, false, 1, "", ""},
		{"even-power-zero-divisor", []uint32{2, 1}, [][3]int64{{2, 8, 12}}, 2, false, 1, "", ""},
		{"bounded-product-both-directions", []uint32{1, 1}, [][3]int64{{0, 2, 3}, {1, 4, 5}, {2, 8, 10}}, 22, true, 0, "2", "5/2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			theory, solver, factors, product := smtNonlinearSelectionFixture(t, tc.powers)
			columns := append(factors, product)
			for _, bound := range tc.bounds {
				smtTestIntervalBound(t, solver, columns[bound[0]], true, bound[1], 20+int32(bound[0])*2)
				smtTestIntervalBound(t, solver, columns[bound[0]], false, bound[2], 21+int32(bound[0])*2)
			}
			before := theory.state.manager.Limit.Count()
			changed, conflict, err := theory.propagateNonlinearBounds(solver)
			astTestMust(t, err)
			if changed != tc.changed || conflict {
				t.Fatal("propagation result", changed, conflict)
			}
			if cost := theory.state.manager.Limit.Count() - before; cost != tc.cost {
				t.Fatal("selected interval work", cost, tc.cost)
			}
			column := columns[tc.inspect]
			for _, lower := range []bool{true, false} {
				bound, found := solver.Upper(column)
				want := tc.wantUpper
				if lower {
					bound, found = solver.Lower(column)
					want = tc.wantLower
				}
				if found != (want != "") || found && (bound.Main.RatString() != want || bound.Epsilon.Sign() != 0) {
					t.Fatal("selected bound", lower, bound, found, want)
				}
				if found {
					if _, known := solver.BoundDependencies(column, lower); !known {
						t.Fatal("selected bound lost premises")
					}
				}
			}
		})
	}
}

func TestSMTNonlinearPropagationBatchesMonomialsBeforeApplyingBounds(t *testing.T) {
	theory, solver, factors, _ := smtNonlinearSelectionFixture(t, []uint32{1, 1, 1})
	x, y, z := factors[0], factors[1], factors[2]
	m := theory.state.manager
	makeProduct := func(first, second uint32) uint32 {
		var arguments []ast.ASTHandle
		for _, column := range []uint32{first, second} {
			variable, ok := theory.theoryVariable(column)
			if !ok {
				t.Fatal("missing factor variable")
			}
			node, err := theory.Enode(variable)
			astTestMust(t, err)
			arguments = append(arguments, node.owner)
		}
		term, err := m.MkArithmeticMul(arguments[0], arguments[1])
		astTestMust(t, err)
		handled, err := theory.InternalizeTerm(term)
		astTestMust(t, err)
		if !handled {
			t.Fatal("product was not internalized")
		}
		column, err := theory.ensureVariable(term)
		astTestMust(t, err)
		if theory.state.relevancy != nil {
			astTestMust(t, theory.state.relevancy.MarkRelevant(term))
		}
		return column
	}
	first, second := makeProduct(x, y), makeProduct(y, z)
	astTestMust(t, solver.SetLowerWithDependencies(x, simplex.RationalInfinitesimal(big.NewRat(2, 1)), nil))
	astTestMust(t, solver.SetLowerWithDependencies(z, simplex.RationalInfinitesimal(big.NewRat(1, 1)), nil))
	astTestMust(t, solver.SetLowerWithDependencies(first, simplex.RationalInfinitesimal(big.NewRat(8, 1)), nil))
	astTestMust(t, solver.SetUpperWithDependencies(first, simplex.RationalInfinitesimal(big.NewRat(8, 1)), nil))
	before := m.Limit.Count()
	changed, conflict, err := theory.propagateNonlinearBounds(solver)
	astTestMust(t, err)
	if !changed || conflict || m.Limit.Count()-before != 3 {
		t.Fatal("first pass", changed, conflict, m.Limit.Count()-before)
	}
	if _, found := solver.Upper(y); !found {
		t.Fatal("first monomial failed to derive y<=4")
	}
	if _, found := solver.Lower(second); found {
		t.Fatal("second monomial observed a bound queued during the same pass")
	}
	changed, conflict, err = theory.propagateNonlinearBounds(solver)
	astTestMust(t, err)
	if !changed || conflict {
		t.Fatal("second pass", changed, conflict)
	}
	if lower, found := solver.Lower(second); !found || lower.Main.Sign() != 0 || lower.Epsilon.Sign() <= 0 {
		t.Fatal("next pass failed to consume the prior y>0 bound", lower, found)
	}
}
