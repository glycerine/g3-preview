package smtcore

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/grobner"
	"github.com/glycerine/g3/pkg/math/simplex"
)

type smtGrobnerIntervalFixture struct {
	theory  *SMTLinearArithmeticTheory
	solver  *simplex.Solver
	gb      *grobner.Solver
	vars    []ast.ASTHandle
	columns map[ast.ASTHandle]uint32
}

func newSMTGrobnerIntervalFixture(t *testing.T) *smtGrobnerIntervalFixture {
	t.Helper()
	m, _ := smtModelGeneratorFixture(t)
	family, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 0, nil)
	astTestMust(t, err)
	gb, err := grobner.New(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, gb.Close()) })
	f := &smtGrobnerIntervalFixture{
		theory:  &SMTLinearArithmeticTheory{SMTTheoryBase: &SMTTheoryBase{state: &SMTInternalizerState{manager: m}}},
		solver:  simplex.NewSolver(nil),
		gb:      gb,
		columns: make(map[ast.ASTHandle]uint32),
	}
	for index, name := range []string{"interval-x", "interval-y", "interval-z"} {
		variable, err := m.MkConst(astTestSymbol(t, m, name), sort)
		astTestMust(t, err)
		f.vars = append(f.vars, variable)
		f.columns[variable] = uint32(index + 1)
	}
	return f
}

func (f *smtGrobnerIntervalFixture) monomial(t *testing.T, coefficient int64, variables ...ast.ASTHandle) *grobner.Monomial {
	t.Helper()
	monomial, err := f.gb.NewMonomial(big.NewRat(coefficient, 1), variables)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, f.gb.DelMonomial(monomial)) })
	return monomial
}

func TestSMTGrobnerIntervalSumStopsOnlyWhenFullyUnbounded(t *testing.T) {
	for _, tc := range []struct {
		name         string
		lower, upper bool
		cost         uint64
		unbounded    bool
	}{
		{"unbounded-prefix", false, false, 2, true},
		{"lower-bounded-prefix", true, false, 25, false},
		{"upper-bounded-prefix", false, true, 25, false},
		{"bounded-prefix", true, true, 25, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSMTGrobnerIntervalFixture(t)
			if tc.lower {
				smtTestIntervalBound(t, f.solver, 1, true, -1, 10)
			}
			if tc.upper {
				smtTestIntervalBound(t, f.solver, 1, false, 1, 11)
			}
			smtTestIntervalBound(t, f.solver, 2, true, 1024, 12)
			smtTestIntervalBound(t, f.solver, 2, false, 2048, 13)
			monomials := []*grobner.Monomial{f.monomial(t, 1, f.vars[0]), f.monomial(t, 1, f.vars[1])}
			var dependencies smtArithmeticBoundDependencyManager
			zero, err := newSMTOldIntervalPoint(&dependencies, new(big.Rat), nil, nil)
			astTestMust(t, err)
			limit := &f.theory.state.manager.Limit
			before := limit.Count()
			sum, err := f.theory.grobnerSumInterval(f.solver, f.columns, &dependencies, zero, monomials)
			astTestMust(t, err)
			if got := limit.Count() - before; got != tc.cost {
				t.Fatal("monomial prefix resource charge", got, tc.cost)
			}
			if (sum.minusInfinity() && sum.plusInfinity()) != tc.unbounded {
				t.Fatal("sum interval", sum)
			}
			if !zero.zero() {
				t.Fatal("sum mutated its initial interval", zero)
			}
			if !tc.unbounded {
				lower := tc.lower
				root := sum.upperDep
				if lower {
					root = sum.lowerDep
				}
				explanation := smtArithmeticIntervalExplanation(&dependencies, root)
				if explanation == nil || len(explanation.dependencies) != 2 {
					t.Fatal("finite prefix endpoint lost its premises", explanation)
				}
			}
		})
	}
}

func TestSMTGrobnerSquareReevaluatesSurvivorsWithTheirEndpointPremises(t *testing.T) {
	f := newSMTGrobnerIntervalFixture(t)
	x, y, z := f.vars[0], f.vars[1], f.vars[2]
	smtTestIntervalBound(t, f.solver, 3, true, 2, 20)
	smtTestIntervalBound(t, f.solver, 3, false, 3, 21)
	monomials := []*grobner.Monomial{f.monomial(t, 1, x, x), f.monomial(t, 1, y, y), f.monomial(t, -2, x, y), f.monomial(t, 1, z)}
	var dependencies smtArithmeticBoundDependencyManager
	limit := &f.theory.state.manager.Limit
	before := limit.Count()
	interval, err := f.theory.grobnerSquareConflictInterval(f.solver, f.columns, &dependencies, monomials)
	astTestMust(t, err)
	// Initial intervals cost 2+2+4+4. The surviving z monomial is
	// evaluated again after replacing (x-y)^2 by [0,+infinity), costing 4.
	if got := limit.Count() - before; got != 16 {
		t.Fatal("square survivor resource charge", got, 16)
	}
	if interval == nil || interval.lower.value.Cmp(big.NewRat(2, 1)) != 0 || !interval.plusInfinity() {
		t.Fatal("square interval", interval)
	}
	explanation := smtArithmeticIntervalExplanation(&dependencies, interval.lowerDep)
	if explanation == nil || len(explanation.dependencies) != 1 || explanation.dependencies[0].Literal != 20 || explanation.dependencies[0].Coefficient.Cmp(big.NewRat(1, 1)) != 0 {
		t.Fatal("square conflict must use only z's lower endpoint", explanation)
	}
}
