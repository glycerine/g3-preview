package ufbv

import (
	"testing"

	"github.com/glycerine/g3/pkg/tactic"
)

func TestUFBVRewriterCommonTacticAdapterBalancesGoalReference(t *testing.T) {
	m := cacheFixture(t)
	context, globals := ufbvTacticServicesFixture(t, m)
	runner, err := NewRewriterTacticAdapter(m, context, globals)
	cacheMust(t, err)
	t.Cleanup(func() { cacheMust(t, runner.Close()) })
	goal, err := tactic.NewGoal(m, false, false, false)
	cacheMust(t, err)
	t.Cleanup(func() { cacheMust(t, goal.Close()) })
	before := goal.RefCount
	var result tactic.TacticResult
	t.Cleanup(func() { cacheMust(t, result.Close()) })
	cacheMust(t, runner.Apply(goal, &result))
	if len(result.Goals) != 1 || result.Goals[0] != goal || goal.RefCount != before {
		t.Fatal("common adapter goal ownership", len(result.Goals), result.Goals[0], goal.RefCount, before)
	}
}
