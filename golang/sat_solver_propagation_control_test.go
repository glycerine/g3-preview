// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"testing"
)

func satPropagationFixture(t *testing.T, n uint32) (*satSolverPropagationState, *satSolverVariableState, *arena.Budget) {
	t.Helper()
	v, _, b := satVariableStateFixture(t)
	for i := uint32(0); i < n; i++ {
		_, e := v.mkVar(false, true)
		satOK(t, e)
	}
	a, e := NewSATClauseAllocator(satClauseAllocatorConfig{Storage: arena.SmallBytesConfig{Descriptors: arena.Config{Budget: b}}})
	satOK(t, e)
	mc, e := newSATModelConverter(arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{Budget: b}})
	satOK(t, e)
	r, e := newUtilReslimit(b)
	satOK(t, e)
	mem := v.memory
	ctx, _, g, cb := satMainConfigTestRuntime(t)
	p, e := ctx.NewValue()
	satOK(t, e)
	cfg, e := newSATMainConfig(g, &p, cb)
	satOK(t, e)
	s, e := newSATSolverPropagationState(v, a, mc, r, mem, cfg, b)
	satOK(t, e)
	t.Cleanup(func() {
		satOK(t, s.Close())
		satOK(t, a.Close())
		satOK(t, mc.Close())
		r.close()
		satOK(t, cfg.Close())
		satOK(t, ctx.Del(&p))
	})
	return s, v, b
}
func TestSATSolverCheckpointActualResourceAndMemoryOrder(t *testing.T) {
	s, v, _ := satPropagationFixture(t, 1)
	s.checkpointEnabled = false
	s.limit.setCancel(1)
	satOK(t, s.checkpoint())
	if s.limit.count() != 0 {
		t.Fatal("disabled checkpoint")
	}
	s.checkpointEnabled = true
	v.modelIsCurrent = true
	_, e := s.converter.Mk(satModelElimVar, 0)
	satOK(t, e)
	if e = s.checkpoint(); !errors.Is(e, ErrSATSolverCanceled) {
		t.Fatal(e)
	}
	if !s.converter.Empty() || v.modelIsCurrent || s.numCheckpoints != 0 || s.limit.count() != 1 {
		t.Fatal("cancel prefix")
	}
	s.limit.resetCancel()
	s.config.state.MaxMemory = 0
	h, e := s.memory.Allocate(8)
	satOK(t, e)
	defer func() { satOK(t, s.memory.Deallocate(&h)) }()
	for i := 0; i < 9; i++ {
		satOK(t, s.checkpoint())
	}
	if e = s.checkpoint(); !errors.Is(e, ErrSATSolverMaxMemory) || s.numCheckpoints != 0 {
		t.Fatal(e, s.numCheckpoints)
	}
}

func satPropagationAssertGraph(t *testing.T, s *satSolverPropagationState) {
	t.Helper()
	v := s.variables
	b := s.budget
	want := uint64(reflect.TypeOf(satSolverPropagationState{}).Size()) + uint64(cap(s.diffLevels.data)) + v.Stats().TotalBytes
	todo := v.elimTodo
	want += todo.ownerBytes + uint64(cap(todo.inSet)) + 4*uint64(cap(todo.values))
	a := s.clauses
	small := a.storage.Stats()
	want += a.ownerBytes + 4*uint64(cap(a.ids.free)) + a.largeBytes + a.largeDirectory + small.OwnerBytes + small.PayloadBytes + small.MetadataBytes + small.DirectoryBytes
	mc := s.converter
	want += mc.ownerBytes + mc.entryBytes + mc.pendingBytes + mc.store.ownerBytes + mc.store.directoryBytes + satWatchGraphBytes(mc.store.words.Stats())
	r := s.limit
	want += uint64(reflect.TypeOf(utilReslimit{}).Size()) + 8*uint64(cap(r.limits)) + uint64(reflect.TypeOf((*utilReslimit)(nil)).Size())*uint64(cap(r.children))
	want += s.memory.Stats().TotalBytes
	if b.Used() != want {
		t.Fatalf("actual once-only borrowed owner graph %d != %d", b.Used(), want)
	}
	satOK(t, v.checkPhysicalOwners())
	// Config has its own real registry graph/budget; never add its borrowed
	// pointer to this budget. Its literal header must remain charged there.
	if s.config.ownerBytes != uint64(reflect.TypeOf(satMainConfig{}).Size()) || s.config.budget.Used() < s.config.ownerBytes {
		t.Fatal("config owner graph")
	}
}
func TestSATSolverPropagationConcreteExceptionAndResizePrefix(t *testing.T) {
	s, v, b := satPropagationFixture(t, 1)
	v.scopeLevel = 10
	c, e := s.clauses.MkClause([]satLiteral{0, 0, 0}, false)
	satOK(t, e)
	satOK(t, s.assign(0, satJustification{}))
	satPropagationAssertGraph(t, s)
	b.FailAfter(1)
	_, e = s.numDiffLevels(c)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if len(s.diffLevels.data) != 0 || cap(s.diffLevels.data) != 2 {
		t.Fatal("source resize unpublished size", len(s.diffLevels.data), cap(s.diffLevels.data))
	}
	satPropagationAssertGraph(t, s)
	b.FailAfter(-1)
	var ex SATSolverException
	if !errors.As(ErrSATSolverCanceled, &ex) || ex.ErrorCode() != ErrorOK || ex.Message() != "canceled" {
		t.Fatal("source solver_exception category")
	}
	var base DefaultException
	if !errors.As(ErrSATSolverCanceled, &base) || base.Message() != "canceled" {
		t.Fatal("base exception catch")
	}
	if errors.Is(ErrSATSolverCanceled, ErrSATSolverMaxMemory) {
		t.Fatal("distinct errors")
	}
}

func TestSATSolverPropagationOwnerBudgetAndCounterBoundaries(t *testing.T) {
	s, v, b := satPropagationFixture(t, 1)
	satPropagationAssertGraph(t, s)
	before := b.Used()
	b.FailAfter(0)
	bad, e := newSATSolverPropagationState(v, s.clauses, s.converter, s.limit, s.memory, s.config, b)
	if bad != nil || !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != before {
		t.Fatal("cold reserve before construction", bad, e)
	}
	satPropagationAssertGraph(t, s)
	b.FailAfter(-1)
	copy := *s
	if e = copy.assign(0, satJustification{}); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	if e = copy.Close(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	var zero satSolverPropagationState
	if e = zero.Close(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satOK(t, s.assign(0, satJustification{}))
	s.cleanerCounter = -1 << 31
	if _, e = s.propagateCore(false); !errors.Is(e, ErrSATSignedCounterDomain) || v.qhead != 0 {
		t.Fatal("source undefined decrement before qhead", e)
	}
	s.cleanerCounter = 0
	s.asymmCounter = -1 << 31
	s.probingCounter = -1 << 31
	w, e := newSATBinaryWatch(0, false)
	satOK(t, e)
	satPropagationPush(t, s, 0, w)
	ok, e := s.propagateCore(false)
	satOK(t, e)
	if !ok || s.asymmCounter != 1<<31-1 || s.probingCounter != 1<<31-1 {
		t.Fatal("source unsigned subtraction conversion")
	}
	satOK(t, s.Close())
	if _, e = s.propagateCore(false); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
