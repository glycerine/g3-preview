package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTAssumeEqualityScopedPhaseAndRelevancy(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	s, err := NewSMTSolver(m, nil, "QF_UFLRA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	state := s.kernel.internalizer
	family, err := m.ArithmeticFamilyID()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 0, nil)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "phase-x"), sort)
	astTestMust(t, err)
	y, err := m.MkConst(astTestSymbol(t, m, "phase-y"), sort)
	astTestMust(t, err)
	astTestMust(t, state.internalizeTerm(x))
	astTestMust(t, state.internalizeTerm(y))
	left, right := state.context.enodes[x], state.context.enodes[y]
	eq, err := m.MkEq(x, y)
	astTestMust(t, err)
	ref, err := m.NewRef(eq)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
	// A newly created interface equality sets its phase before arithmetic
	// creates the auxiliary comparisons, and all three flags are scoped.
	astTestMust(t, state.PushScope())
	progress, err := state.assumeEquality(left, right)
	astTestMust(t, err)
	if !progress || state.statistics.NumInterfaceEqs != 1 {
		t.Fatal("new interface equality was not recorded")
	}
	variable, found := state.BoolVar(eq)
	if !found || !state.booleanData[variable].TryTrueFirst() || !state.relevancy.IsRelevant(eq) {
		t.Fatal("interface equality has no preferred relevant split")
	}
	for _, op := range []int32{smtArithOpLE, smtArithOpGE} {
		bound, err := m.MkFamilyApp(family, op, nil, []ast.ASTHandle{x, y})
		astTestMust(t, err)
		v, found := state.BoolVar(bound)
		if !found || !state.booleanData[v].TryTrueFirst() {
			t.Fatal("auxiliary bound did not inherit interface phase")
		}
	}
	astTestMust(t, state.PopScopes(1))
	if _, found := state.BoolVar(eq); found || len(state.trueFirstTrail) != 0 {
		t.Fatal("new interface equality survived scope pop")
	}
	// Existing false equalities can receive a scoped preference, but are
	// neither asserted true nor repeatedly reported as fresh progress.
	astTestMust(t, state.Internalize(eq, true))
	variable, _ = state.BoolVar(eq)
	astTestMust(t, state.SetAssignment(variable, lbool.False))
	for repeat := 0; repeat < 2; repeat++ {
		astTestMust(t, state.PushScope())
		progress, err = state.assumeEquality(left, right)
		astTestMust(t, err)
		if !progress || !state.booleanData[variable].TryTrueFirst() {
			t.Fatal("existing interface phase was not set")
		}
		assignment, err := state.Assignment(NewSMTLiteral(variable, false))
		astTestMust(t, err)
		if assignment != lbool.False {
			t.Fatal("model equality was asserted as a fact")
		}
		progress, err = state.assumeEquality(left, right)
		astTestMust(t, err)
		if progress {
			t.Fatal("assigned preferred interface equality reported repeated progress")
		}
		astTestMust(t, state.PopScopes(1))
		if state.booleanData[variable].TryTrueFirst() {
			t.Fatal("existing Boolean variable retained scoped phase")
		}
	}
}
