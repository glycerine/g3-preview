package aig

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
)

func TestAIGGoalConversionSharesExpressionCacheAndConjoins(t *testing.T) {
	m, budget := aigFixture(t)
	sort, e := m.MkBoolSort()
	aigMust(t, e)
	p, e := m.MkFreshConst("p", sort)
	aigMust(t, e)
	q, e := m.MkFreshConst("q", sort)
	aigMust(t, e)
	or, e := m.MkOr(p, q)
	aigMust(t, e)
	g, e := tactic.NewGoal(m, false, false, false)
	aigMust(t, e)
	t.Cleanup(func() { aigMust(t, g.Close()) })
	aigMust(t, g.Assert(p))
	aigMust(t, g.Assert(or))
	manager, e := NewManager(m)
	aigMust(t, e)
	out, e := manager.MkAIGGoal(g)
	aigMust(t, e)
	for mask := uint32(0); mask < 4; mask++ {
		pv, qv := mask&1 != 0, mask&2 != 0
		got := aigEvalFormula(t, m, out.lit, map[ast.ASTHandle]bool{p: pv, q: qv})
		if got != (pv && (pv || qv)) {
			t.Fatalf("goal assignment %02b: %v", mask, got)
		}
	}
	count, e := manager.NumNodes()
	aigMust(t, e)
	if count != 2 {
		t.Fatalf("goal cache/absorption retained %d AIG nodes, want true + p", count)
	}
	aigMust(t, out.Close())
	baseline := budget.Used()
	reached := false
	for prefix := int64(0); prefix < 64; prefix++ {
		budget.FailAfter(prefix)
		trial, failure := manager.MkAIGGoal(g)
		budget.FailAfter(-1)
		if failure == nil {
			aigMust(t, trial.Close())
			reached = true
			break
		}
		if trial != nil || !errors.Is(failure, arena.ErrInjectedFailure) {
			t.Fatalf("prefix %d result %v error %v", prefix, trial, failure)
		}
		if budget.Used() != baseline {
			t.Fatalf("prefix %d retained %d bytes", prefix, budget.Used()-baseline)
		}
	}
	if !reached {
		t.Fatal("goal conversion allocation campaign never succeeded")
	}
	aigMust(t, manager.Close())
}

func TestAIGEmptyGoalIsTrue(t *testing.T) {
	m, _ := aigFixture(t)
	g, e := tactic.NewGoal(m, false, false, false)
	aigMust(t, e)
	t.Cleanup(func() { aigMust(t, g.Close()) })
	manager, e := NewManager(m)
	aigMust(t, e)
	r, e := manager.MkAIGGoal(g)
	aigMust(t, e)
	if r.lit.node != manager.trueNode || r.lit.inverted {
		t.Fatal("empty goal did not return source true")
	}
	aigMust(t, r.Close())
	aigMust(t, manager.Close())
}
