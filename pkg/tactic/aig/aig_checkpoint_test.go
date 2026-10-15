package aig

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/cooperate"
	"github.com/glycerine/g3/pkg/util/messages"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestAIGCheckpointSourceMemoryCancelAndCooperateOrder(t *testing.T) {
	m, budget := aigFixture(t)
	sort, e := m.MkBoolSort()
	aigMust(t, e)
	p, e := m.MkFreshConst("p", sort)
	aigMust(t, e)
	q, e := m.MkFreshConst("q", sort)
	aigMust(t, e)
	and, e := m.MkAnd(p, q)
	aigMust(t, e)
	aigPinExpr(t, m, and)
	g, e := NewManager(m)
	aigMust(t, e)
	runtime := m.NumeralService().RuntimeOwner()
	block, e := runtime.Allocate(32)
	aigMust(t, e)
	used, e := runtime.AllocationSize()
	aigMust(t, e)
	if used == 0 {
		t.Fatal("runtime allocation did not reach source memory counter")
	}
	aigMust(t, g.SetMaxMemory(used-1))
	beforeCount := m.Limit.Count()
	if ref, e := g.MkAIG(and); ref != nil || e == nil || e.Error() != messages.MaxMemory {
		t.Fatalf("memory checkpoint: %v %v", ref, e)
	}
	if m.Limit.Count() != beforeCount {
		t.Fatal("memory refusal reached cancellation check")
	}
	aigMust(t, runtime.Deallocate(&block))
	aigMust(t, g.SetMaxMemory(^uint64(0)))
	m.Limit.Cancel()
	if ref, e := g.MkAIG(and); ref != nil || e == nil || e.Error() != messages.Canceled {
		t.Fatalf("canceled checkpoint: %v %v", ref, e)
	}
	if m.Limit.Count() != beforeCount+1 {
		t.Fatalf("canceled checkpoint count %d, want %d", m.Limit.Count(), beforeCount+1)
	}
	m.Limit.ResetCancel()
	owner, e := cooperate.NewCooperation(budget)
	aigMust(t, e)
	aigMust(t, owner.BeginSection())
	aigMust(t, g.BindCooperation(owner))
	ref, e := g.MkAIG(and)
	aigMust(t, e)
	if owner.OwnerThread != 0 {
		t.Fatal("AIG checkpoint did not reach process cooperation")
	}
	aigMust(t, ref.Close())
	aigMust(t, owner.EndSection())
	aigMust(t, owner.Close())
	aigMust(t, g.Close())
}

func TestAIGGoalOutputCheckpointUsesSameSourceGuards(t *testing.T) {
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
	g, e := NewManager(m)
	aigMust(t, e)
	r, e := g.MkAIG(or)
	aigMust(t, e)
	goal, e := tactic.NewGoal(m, false, false, false)
	aigMust(t, e)
	aigMust(t, goal.Assert(p))
	before := m.Limit.Count()
	m.Limit.Cancel()
	if e := g.ToGoal(r, goal); e == nil || e.Error() != messages.Canceled {
		t.Fatalf("goal-output cancellation: %v", e)
	}
	if m.Limit.Count() != before+1 {
		t.Fatal("goal output skipped its first source checkpoint")
	}
	size, e := goal.Size()
	aigMust(t, e)
	if size != 0 {
		t.Fatalf("source reset did not precede goal-output checkpoint: %d", size)
	}
	m.Limit.ResetCancel()
	aigMust(t, g.ToGoal(r, goal))
	aigMust(t, goal.Close())
	aigMust(t, r.Close())
	aigMust(t, g.Close())
}

func TestAIGPassParamsBindMemoryEncodingAndWholeGoalBranch(t *testing.T) {
	m, _ := aigFixture(t)
	b := m.Budget()
	ctx, e := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{
		Budget: b, PageEntries: 1,
		Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}},
	})
	aigMust(t, e)
	v, e := ctx.NewValue()
	aigMust(t, e)
	pass := NewPass()
	aigMust(t, pass.UpdateParams(ctx, &v))
	if pass.maxMemory != ^uint64(0) || !pass.DefaultGateEncoding || !pass.PerAssertion {
		t.Fatal("default params lost source UINT_MAX sentinel or true flags")
	}
	memoryName, encodingName, perName := "max_memory", "aig_default_gate_encoding", "aig_per_assertion"
	aigMust(t, ctx.SetUIntName(&v, &memoryName, 0))
	aigMust(t, ctx.SetBoolName(&v, &encodingName, false))
	aigMust(t, ctx.SetBoolName(&v, &perName, false))
	aigMust(t, pass.UpdateParams(ctx, &v))
	if pass.maxMemory != 0 || pass.DefaultGateEncoding || pass.PerAssertion {
		t.Fatal("parameter update did not select source whole-goal alternate encoding")
	}
	sort, e := m.MkBoolSort()
	aigMust(t, e)
	p, e := m.MkFreshConst("p", sort)
	aigMust(t, e)
	q, e := m.MkFreshConst("q", sort)
	aigMust(t, e)
	r, e := m.MkFreshConst("r", sort)
	aigMust(t, e)
	or1, e := m.MkOr(p, q)
	aigMust(t, e)
	or2, e := m.MkOr(q, r)
	aigMust(t, e)
	goal, e := tactic.NewGoal(m, false, false, false)
	aigMust(t, e)
	aigMust(t, goal.Assert(or1))
	aigMust(t, goal.Assert(or2))
	runtime := m.NumeralService().RuntimeOwner()
	block, e := runtime.Allocate(32)
	aigMust(t, e)
	if out, e := pass.Apply(goal); out != nil || e == nil || e.Error() != messages.MaxMemory {
		t.Fatalf("zero-MB tactic limit: %v %v", out, e)
	}
	size, e := goal.Size()
	aigMust(t, e)
	if size != 2 {
		t.Fatal("memory refusal changed the input goal")
	}
	aigMust(t, ctx.SetUIntName(&v, &memoryName, 1))
	aigMust(t, pass.UpdateParams(ctx, &v))
	if pass.maxMemory != 1<<20 {
		t.Fatal("megabyte parameter did not convert to bytes")
	}
	out, e := pass.Apply(goal)
	aigMust(t, e)
	size, e = goal.Size()
	aigMust(t, e)
	if size != 2 {
		t.Fatalf("whole-goal branch emitted %d assertions, want two", size)
	}
	aigMust(t, out.DecRef())
	aigMust(t, runtime.Deallocate(&block))
	aigMust(t, ctx.Del(&v))
	aigMust(t, ctx.Close())
}
