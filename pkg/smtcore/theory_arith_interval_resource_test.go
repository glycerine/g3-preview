package smtcore

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/grobner"
	"github.com/glycerine/g3/pkg/math/simplex"
)

func TestSMTNonlinearIntervalMultiplicationChargesResultEndpoints(t *testing.T) {
	for _, tc := range []struct {
		name, lower, upper, coefficient, wantLower, wantUpper string
		lowerOpen, upperOpen                                  bool
		power                                                 uint32
		cost                                                  uint64
	}{
		{"rational-power", "2/3", "2/3", "1", "4/9", "4/9", false, false, 2, 14},
		{"coefficient-cancellation", "2/3", "2/3", "3", "2", "2", false, false, 1, 4},
		{"zero", "0", "0", "1", "0", "0", false, false, 1, 2},
		{"negative-power", "-3", "-2", "1", "4", "9", false, false, 2, 7},
		{"open-endpoints", "-1024", "2048", "1", "-1024", "2048", true, true, 1, 2},
		{"half-infinite", "-1024", "", "1", "-1024", "", false, false, 1, 12},
		{"unbounded", "", "", "1", "", "", false, false, 1, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := smtModelGeneratorFixture(t)
			theory := &SMTLinearArithmeticTheory{SMTTheoryBase: &SMTTheoryBase{state: &SMTInternalizerState{manager: m}}}
			solver := simplex.NewSolver(nil)
			rat := func(text string) *big.Rat {
				value, ok := new(big.Rat).SetString(text)
				if !ok {
					t.Fatal(text)
				}
				return value
			}
			if tc.lower != "" {
				value := simplex.RationalInfinitesimal(rat(tc.lower))
				if tc.lowerOpen {
					value.Epsilon.SetInt64(1)
				}
				astTestMust(t, solver.SetLowerWithDependencies(1, value, nil))
			}
			if tc.upper != "" {
				value := simplex.RationalInfinitesimal(rat(tc.upper))
				if tc.upperOpen {
					value.Epsilon.SetInt64(-1)
				}
				astTestMust(t, solver.SetUpperWithDependencies(1, value, nil))
			}
			var dependencies smtArithmeticBoundDependencyManager
			target, err := newSMTOldIntervalPoint(&dependencies, rat(tc.coefficient), nil, nil)
			astTestMust(t, err)
			// The source charges both endpoints even when the first charge
			// exhausts the limit; interruption is observed by later checks.
			astTestMust(t, m.Limit.Push(1))
			before := m.Limit.Count()
			astTestMust(t, theory.multiplyNonlinearColumnBound(solver, 1, tc.power, target))
			if got := m.Limit.Count() - before; got != tc.cost {
				t.Fatal("endpoint resource charge", got, tc.cost)
			}
			if m.Limit.IncOffset(0) {
				t.Fatal("resource exhaustion not visible to next checkpoint")
			}
			m.Limit.Pop()
			for index, want := range []string{tc.wantLower, tc.wantUpper} {
				endpoint := &target.lower
				if index == 1 {
					endpoint = &target.upper
				}
				if want == "" {
					if !endpoint.infinite() {
						t.Fatal("expected infinite endpoint", index)
					}
				} else if endpoint.infinite() || endpoint.value.Cmp(rat(want)) != 0 {
					t.Fatal("interval value changed by resource charging", index, endpoint.value.String(), want)
				}
			}
		})
	}
}

func TestSMTGrobnerMonomialIntervalChargesPoweredAccumulator(t *testing.T) {
	m, _ := smtModelGeneratorFixture(t)
	theory := &SMTLinearArithmeticTheory{SMTTheoryBase: &SMTTheoryBase{state: &SMTInternalizerState{manager: m}}}
	solver := simplex.NewSolver(nil)
	for column, value := range []*big.Rat{big.NewRat(2, 3), big.NewRat(4, 5)} {
		bound := simplex.RationalInfinitesimal(value)
		astTestMust(t, solver.SetLowerWithDependencies(uint32(column+1), bound, nil))
		astTestMust(t, solver.SetUpperWithDependencies(uint32(column+1), bound, nil))
	}
	family, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 0, nil)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "resource-x"), sort)
	astTestMust(t, err)
	y, err := m.MkConst(astTestSymbol(t, m, "resource-y"), sort)
	astTestMust(t, err)
	gb, err := grobner.New(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, gb.Close()) })
	monomial, err := gb.NewMonomial(big.NewRat(3, 1), []ast.ASTHandle{x, x, y})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, gb.DelMonomial(monomial)) })
	var dependencies smtArithmeticBoundDependencyManager
	before := m.Limit.Count()
	interval, err := theory.grobnerMonomialInterval(solver, map[ast.ASTHandle]uint32{x: 1, y: 2}, &dependencies, monomial)
	astTestMust(t, err)
	// 3*(2/3)^2=4/3 costs 5 bits per endpoint, then 16/15
	// costs 9 bits per endpoint. Repeated factors are charged once as powers.
	if got := m.Limit.Count() - before; got != 28 {
		t.Fatal("powered accumulator charge", got, 28)
	}
	if interval.lower.value.Cmp(big.NewRat(16, 15)) != 0 || interval.upper.value.Cmp(big.NewRat(16, 15)) != 0 {
		t.Fatal("monomial interval", interval.lower.value.String(), interval.upper.value.String())
	}
}
