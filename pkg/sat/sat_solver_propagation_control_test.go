package sat

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func satPropagationFixture(t *testing.T, n uint32) (*satSolverPropagationState, *satSolverVariableState, *arena.Budget) {
	t.Helper()
	return satPropagationFixtureWithBudget(t, n, arena.NewBudget(^uint64(0)))
}

func satPropagationFixtureWithBudget(t *testing.T, n uint32, b *arena.Budget) (*satSolverPropagationState, *satSolverVariableState, *arena.Budget) {
	t.Helper()
	v, _, _ := satVariableStateFixtureWithBudget(t, b)
	for i := uint32(0); i < n; i++ {
		_, e := v.mkVar(false, true)
		satOK(t, e)
	}
	a, e := NewSATClauseAllocator(satClauseAllocatorConfig{Storage: arena.SmallBytesConfig{Descriptors: arena.Config{Budget: b}}})
	satOK(t, e)
	mc, e := newSATModelConverter(b)
	satOK(t, e)
	r, e := control.NewReslimit(b)
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
		r.Close()
		satOK(t, cfg.Close())
		satOK(t, ctx.Del(&p))
	})
	return s, v, b
}
func TestSATSolverCheckpointActualResourceAndMemoryOrder(t *testing.T) {
	s, v, _ := satPropagationFixture(t, 1)
	s.checkpointEnabled = false
	s.limit.SetCancel(1)
	satOK(t, s.checkpoint())
	if s.limit.Count() != 0 {
		t.Fatal("disabled checkpoint")
	}
	s.checkpointEnabled = true
	v.modelIsCurrent = true
	_, e := s.converter.Mk(satModelElimVar, 0)
	satOK(t, e)
	if e = s.checkpoint(); !errors.Is(e, ErrSATSolverCanceled) {
		t.Fatal(e)
	}
	if !s.converter.Empty() || v.modelIsCurrent || s.numCheckpoints != 0 || s.limit.Count() != 1 {
		t.Fatal("cancel prefix")
	}
	s.limit.ResetCancel()
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
	want := uint64(reflect.TypeOf(satSolverPropagationState{}).Size()) + v.Stats().TotalBytes
	todo := v.elimTodo
	want += todo.ownerBytes
	a := s.clauses
	small := a.storage.Stats()
	want += a.ownerBytes + a.largeBytes + a.largeDirectory + small.OwnerBytes + small.PayloadBytes + small.MetadataBytes + small.DirectoryBytes
	mc := s.converter
	want += mc.ownerBytes
	r := s.limit
	want += uint64(reflect.TypeOf(control.Reslimit{}).Size()) + 8*uint64(cap(r.Limits)) + uint64(reflect.TypeOf((*control.Reslimit)(nil)).Size())*uint64(cap(r.Children))
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
func TestSATSolverPropagationConcreteExceptionAndNativeMarks(t *testing.T) {
	s, v, b := satPropagationFixture(t, 1)
	v.scopeLevel = 10
	c, e := s.clauses.MkClause([]Literal{0, 0, 0}, false)
	satOK(t, e)
	satOK(t, s.assign(0, satJustification{}))
	satPropagationAssertGraph(t, s)
	before := b.Used()
	b.FailAfter(0)
	count, e := s.numDiffLevels(c)
	if e != nil || count != 1 || len(s.diffLevels.data) != 11 || b.Used() != before {
		t.Fatal("native marks must grow without arena reservation", count, e, len(s.diffLevels.data), b.Used(), before)
	}
	satPropagationAssertGraph(t, s)
	b.FailAfter(-1)
	var ex SATSolverException
	if !errors.As(ErrSATSolverCanceled, &ex) || ex.ErrorCode() != z3exception.ErrorOK || ex.Message() != "canceled" {
		t.Fatal("source solver_exception category")
	}
	var base z3exception.DefaultException
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
