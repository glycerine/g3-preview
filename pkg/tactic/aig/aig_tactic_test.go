package aig

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
)

func TestAIGPassPerAssertionPreservesDependenciesAndDepth(t *testing.T) {
	m, _ := aigFixture(t)
	sort, e := m.MkBoolSort()
	aigMust(t, e)
	p, e := m.MkFreshConst("p", sort)
	aigMust(t, e)
	q, e := m.MkFreshConst("q", sort)
	aigMust(t, e)
	or, e := m.MkOr(p, q)
	aigMust(t, e)
	g, e := tactic.NewGoal(m, false, false, true)
	aigMust(t, e)
	dep, e := m.DependencyLeaf(p)
	aigMust(t, e)
	aigMust(t, g.AssertWithProof(or, ast.ASTHandle{}, dep))
	aigMust(t, g.AssertWithProof(q, ast.ASTHandle{}, dep))
	aigMust(t, g.SetDepth(4))
	out, e := NewPass().Apply(g)
	aigMust(t, e)
	if out != g {
		t.Fatal("AIG pass did not return the input goal")
	}
	size, e := g.Size()
	aigMust(t, e)
	if size != 2 {
		t.Fatalf("per-assertion path changed assertion count: %d", size)
	}
	for i := uint32(0); i < size; i++ {
		d, e := g.Dependency(i)
		aigMust(t, e)
		if d != dep {
			t.Fatalf("assertion %d lost its dependency", i)
		}
		form, e := g.Form(i)
		aigMust(t, e)
		for mask := uint32(0); mask < 4; mask++ {
			pv, qv := mask&1 != 0, mask&2 != 0
			want := qv
			if i == 0 {
				want = pv || qv
			}
			if got := aigEvalASTFormula(t, m, form, map[ast.ASTHandle]bool{p: pv, q: qv}); got != want {
				t.Fatalf("assertion %d assignment %02b: %v want %v", i, mask, got, want)
			}
		}
	}
	depth, e := g.Depth()
	aigMust(t, e)
	if depth != 5 {
		t.Fatalf("depth %d, want 5", depth)
	}
	aigMust(t, out.DecRef())
}

func TestAIGPassWholeGoalCombinesAndRejectsCore(t *testing.T) {
	m, _ := aigFixture(t)
	sort, e := m.MkBoolSort()
	aigMust(t, e)
	p, e := m.MkFreshConst("p", sort)
	aigMust(t, e)
	q, e := m.MkFreshConst("q", sort)
	aigMust(t, e)
	or, e := m.MkOr(p, q)
	aigMust(t, e)
	aigPinExpr(t, m, or)
	coreGoal, e := tactic.NewGoal(m, false, false, true)
	aigMust(t, e)
	aigMust(t, coreGoal.Assert(or))
	pass := NewPass()
	pass.PerAssertion = false
	if _, e := pass.Apply(coreGoal); e == nil || e.Error() != "aig does not support unsat core production" {
		t.Fatalf("whole-goal core guard: %v", e)
	}
	size, e := coreGoal.Size()
	aigMust(t, e)
	if size != 1 {
		t.Fatal("core guard changed the goal")
	}
	aigMust(t, coreGoal.Close())
	g, e := tactic.NewGoal(m, false, false, false)
	aigMust(t, e)
	aigMust(t, g.Assert(or))
	aigMust(t, g.Assert(p))
	out, e := pass.Apply(g)
	aigMust(t, e)
	size, e = g.Size()
	aigMust(t, e)
	if size != 1 {
		t.Fatalf("whole-goal conjunction has %d assertions", size)
	}
	form, e := g.Form(0)
	aigMust(t, e)
	for mask := uint32(0); mask < 4; mask++ {
		pv, qv := mask&1 != 0, mask&2 != 0
		if got := aigEvalASTFormula(t, m, form, map[ast.ASTHandle]bool{p: pv, q: qv}); got != (pv && (pv || qv)) {
			t.Fatalf("whole-goal assignment %02b: %v", mask, got)
		}
	}
	depth, e := g.Depth()
	aigMust(t, e)
	if depth != 1 {
		t.Fatalf("whole-goal depth %d", depth)
	}
	aigMust(t, out.DecRef())
}

func TestAIGPassRejectsProofBeforeGoalMutation(t *testing.T) {
	m, _ := aigFixture(t)
	aigMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	g, e := tactic.NewGoal(m, true, false, false)
	aigMust(t, e)
	sort, e := m.MkBoolSort()
	aigMust(t, e)
	p, e := m.MkFreshConst("p", sort)
	aigMust(t, e)
	aigMust(t, g.Assert(p))
	if _, e := NewPass().Apply(g); e == nil || e.Error() != "aig does not support proof production" {
		t.Fatalf("proof guard: %v", e)
	}
	size, e := g.Size()
	aigMust(t, e)
	if size != 1 {
		t.Fatal("proof guard changed the goal")
	}
	aigMust(t, g.Close())
}
