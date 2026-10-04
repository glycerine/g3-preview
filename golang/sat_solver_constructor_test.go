// Source: src/sat/sat_solver.cpp:34–74,2763–2771; src/sat/sat_solver.h:74–483.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2011.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/utilhash"
	"testing"
)

func satSolverCheckFixture(t *testing.T) (*satSolver, *ParamsContext, ParamsRef) {
	t.Helper()
	ctx, _, _ := paramsTestOwner(t)
	p, e := ctx.NewValue()
	satOK(t, e)
	t.Cleanup(func() { satOK(t, ctx.Del(&p)) })
	runtime, e := newSATRuntime(satRuntimeConfig{Budget: ctx.budget, ParamsContext: ctx})
	satOK(t, e)
	t.Cleanup(func() { satOK(t, runtime.Close()) })
	limit, e := newUtilReslimit(ctx.budget)
	satOK(t, e)
	t.Cleanup(limit.close)
	s, e := NewSATSolver(satSolverConfig{ParamsContext: ctx, Params: p, RLimit: limit, Budget: ctx.budget, Runtime: runtime})
	satOK(t, e)
	t.Cleanup(func() { satOK(t, s.Close()) })
	return s, ctx, p
}
func TestSATSolverCompleteMemberConstructorHasRealDefaultChildren(t *testing.T) {
	s, ctx, params := satSolverCheckFixture(t)
	if s.variables.queue == nil || s.variables.watches == nil || s.variables.activity == nil || s.variables.trail == nil || s.simplifier == nil || s.scc == nil || s.asymm == nil || s.probing == nil || !s.probing.constructed || s.converter.store == nil || !s.converter.constructed || s.antecedents == nil || s.antecedents.constructedEntries != 8 {
		t.Fatal("full constructor lacks actual source members")
	}
	if s.runtime.liveSolvers != 1 || s.nextSimplify != 0 || s.conflict.conflicts != 0 || s.conflict.conflictsSinceGC != 0 || s.propagation.numCheckpoints != 0 {
		t.Fatal("source ctor body")
	}
	if s.mus.isActive || s.mus.maxNumRestarts != ^uint32(0) {
		t.Fatal("source MUS constructor fields")
	}
	key := "random_seed"
	satOK(t, ctx.SetUIntName(&params, &key, 17))
	satOK(t, s.UpdateParams(&params))
	var expected utilhash.Random
	expected.SetSeed(17)
	if got, want := s.decision.rng.Next(), expected.Next(); got != want {
		t.Fatal("sole source random member reseeded last", got, want)
	}
	satOK(t, s.Close())
	if s.runtime.liveSolvers != 0 {
		t.Fatal("retained runtime owner after actual completed member cleanup")
	}
}
func TestSATSolverConstructorRefusesForeignRuntimeBeforeActualHeaderReserve(t *testing.T) {
	s, ctx, params := satSolverCheckFixture(t)
	wrong := arena.NewBudget(^uint64(0))
	wrong.FailAfter(0)
	if child, e := NewSATSolver(satSolverConfig{ParamsContext: ctx, Params: params, RLimit: s.limit, Budget: wrong, Runtime: s.runtime}); child != nil || e != ErrSATOwner {
		t.Fatal(child, e)
	}
	if wrong.Used() != 0 || s.runtime.liveSolvers != 1 {
		t.Fatal("constructor domain refusal publication")
	}
}

// Explicit completed-member/uncompleted-wrapper protocol state: a C++ failed
// solver constructor has no solver destructor body. All children here are real
// constructed owners, and the empty clause lists need no ownership repair.
func TestSATSolverUncompletedWrapperUnwindDoesNotRunDestructorBody(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	s.constructed = false
	before := s.lifecycle.nonLearnedGeneration
	satOK(t, s.Close())
	if s.lifecycle.nonLearnedGeneration != before {
		t.Fatal("failed constructor ran source destructor body", before, s.lifecycle.nonLearnedGeneration)
	}
}
