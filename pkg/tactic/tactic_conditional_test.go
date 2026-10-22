// Source: src/tactic/tactical.cpp:1225-1387.
package tactic

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func tacticOwnForTest(t *testing.T, value Tactic) {
	t.Helper()
	astTestMust(t, value.IncRef())
	t.Cleanup(func() { astTestMust(t, value.DecRef()) })
}

func TestConditionalTacticsChooseWhenAndFailIf(t *testing.T) {
	m := astTestManager(t)
	g := goalTestNew(t, m, false, false)
	truth, err := NewProbe(m.Budget(), ProbeConstant, 1)
	astTestMust(t, err)
	falsity, err := NewProbe(m.Budget(), ProbeConstant, 0)
	astTestMust(t, err)

	chooseTrue, err := NewCondTactic(truth, NewSkipTactic(), NewFailTactic())
	astTestMust(t, err)
	tacticOwnForTest(t, chooseTrue)
	result := &TacticResult{}
	astTestMust(t, chooseTrue.Apply(g, result))
	astTestMust(t, result.Close())

	chooseFalse, err := NewCondTactic(falsity, NewFailTactic(), NewSkipTactic())
	astTestMust(t, err)
	tacticOwnForTest(t, chooseFalse)
	result = &TacticResult{}
	astTestMust(t, chooseFalse.Apply(g, result))
	astTestMust(t, result.Close())

	when, err := NewWhenTactic(falsity, NewFailTactic())
	astTestMust(t, err)
	tacticOwnForTest(t, when)
	result = &TacticResult{}
	astTestMust(t, when.Apply(g, result))
	astTestMust(t, result.Close())

	failIf, err := NewFailIfTactic(truth)
	astTestMust(t, err)
	tacticOwnForTest(t, failIf)
	if err = failIf.Apply(g, &TacticResult{}); err == nil {
		t.Fatal("fail-if accepted a true probe")
	}
	failIfNot, err := NewFailIfNotTactic(falsity)
	astTestMust(t, err)
	tacticOwnForTest(t, failIfNot)
	if err = failIfNot.Apply(g, &TacticResult{}); err == nil {
		t.Fatal("fail-if-not accepted a false probe")
	}
}

func TestConditionalTacticsGoalProductionGuards(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	tests := []struct {
		name    string
		goal    *Goal
		factory func(Tactic) (Tactic, error)
	}{
		{"proofs", func() *Goal { g, e := NewGoal(m, true, false, false); astTestMust(t, e); return g }(), NewIfNoProofsTactic},
		{"models", func() *Goal { g, e := NewGoal(m, false, true, false); astTestMust(t, e); return g }(), NewIfNoModelsTactic},
		{"cores", func() *Goal { g, e := NewGoal(m, false, false, true); astTestMust(t, e); return g }(), NewIfNoUnsatCoresTactic},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() { astTestMust(t, tc.goal.Close()) })
			guard, err := tc.factory(NewFailTactic())
			astTestMust(t, err)
			tacticOwnForTest(t, guard)
			out := &TacticResult{}
			astTestMust(t, guard.Apply(tc.goal, out))
			if len(out.Goals) != 1 || out.Goals[0] != tc.goal {
				t.Fatal("enabled feature did not skip child")
			}
			astTestMust(t, out.Close())
		})
	}
	disabled, err := NewGoal(m, false, false, false)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, disabled.Close()) })
	guard, err := NewIfNoModelsTactic(NewFailTactic())
	astTestMust(t, err)
	tacticOwnForTest(t, guard)
	if err = guard.Apply(disabled, &TacticResult{}); err == nil {
		t.Fatal("disabled feature did not apply child")
	}
}
