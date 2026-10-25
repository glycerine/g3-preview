package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTArithmeticPivotPolicySurvivesReplay(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	real, err := m.MkFamilySort(family, 0, nil)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "pivot-x"), real)
	astTestMust(t, err)
	y, err := m.MkConst(astTestSymbol(t, m, "pivot-y"), real)
	astTestMust(t, err)
	one, err := m.ArithmeticRealNumeralText("1")
	astTestMust(t, err)
	two, err := m.ArithmeticRealNumeralText("2")
	astTestMust(t, err)
	three, err := m.ArithmeticRealNumeralText("3")
	astTestMust(t, err)
	twiceY, err := m.MkFamilyApp(family, smtArithOpMul, nil, []ast.ASTHandle{two, y})
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	theory, err := NewSMTLinearArithmeticTheory(family)
	astTestMust(t, err)
	astTestMust(t, state.RegisterTheory(theory))
	for _, terms := range [][]ast.ASTHandle{{x, y, one}, {x, twiceY, three}} {
		sum, e := m.MkFamilyApp(family, smtArithOpAdd, nil, terms[:2])
		astTestMust(t, e)
		bound, e := m.MkFamilyApp(family, smtArithOpGE, nil, []ast.ASTHandle{sum, terms[2]})
		astTestMust(t, e)
		astTestMust(t, state.Internalize(bound, true))
		boolean, found := state.BoolVar(bound)
		if !found {
			t.Fatal("missing bound")
		}
		astTestMust(t, theory.Assign(boolean, true))
	}
	// Reconfiguration after internalization must affect each fresh replay,
	// including a subsequent return to the smallest-variable policy.
	for _, strategy := range []uint32{0, 1, 2, 0} {
		astTestMust(t, theory.ConfigureArithmetic(SMTArithState{PivotStrategy: strategy, BlandsRuleThreshold: 1000}))
		s, status, e := theory.buildSimplexReplay(nil, nil)
		astTestMust(t, e)
		if status != lbool.True {
			t.Fatal("replay feasibility", status)
		}
		want := uint32(2)
		if strategy == 1 {
			want = 1
		}
		if s.Stats.NumPivots != want {
			t.Fatal("replay lost pivot policy", strategy, s.Stats.NumPivots, "want", want)
		}
	}
}
