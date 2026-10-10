// Source: src/sat/sat_solver.h; src/sat/sat_solver.cpp; src/sat/sat_var_queue.h; src/util/vector.h; src/util/memory_manager.cpp.
package sat

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/lbool"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverNativeActivityAndTrailAccounting(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	_, e := v.mkVar(false, true)
	satOK(t, e)
	n, e := v.memory.AllocationSize()
	satOK(t, e)
	if n != 0 || v.memory.Stats().LiveBlocks != 0 {
		t.Fatal("remaining source allocations", n, v.memory.Stats())
	}
	satOK(t, v.trailPush(MakeLiteral(0, true)))
	n, e = v.memory.AllocationSize()
	satOK(t, e)
	if n != 0 || v.memory.Stats().LiveBlocks != 0 {
		t.Fatal("native trail changed Runtime allocation", n, v.memory.Stats())
	}
}

func TestSATSolverActivityIncrementDecayAndRescaleSourceOrder(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	for i := 0; i < 3; i++ {
		_, e := v.mkVar(false, true)
		satOK(t, e)
	}
	satOK(t, v.activity.Set(0, 10))
	satOK(t, v.activity.Set(1, 20))
	satOK(t, v.activity.Set(2, 30))
	satOK(t, v.incActivity(0))
	next, e := v.queue.NextVar()
	satOK(t, e)
	if next != 0 {
		t.Fatal("current native activity comparator", next)
	}
	v.decayActivity()
	if v.activityInc != 140 {
		t.Fatal("source integer decay", v.activityInc)
	}
	satOK(t, v.activity.Set(1, 1<<24))
	satOK(t, v.incActivity(1))
	for i, want := range []uint32{0, 1024, 0} {
		got, e := v.activity.Get(uint32(i))
		satOK(t, e)
		if got != want {
			t.Fatal(i, got, want)
		}
	}
	if v.activityInc != 0 {
		t.Fatal("source coarse increment shift", v.activityInc)
	}
	v.activityInc = 128
	satOK(t, v.activity.Set(2, ^uint32(0)))
	satOK(t, v.incActivity(2))
	got, e := v.activity.Get(2)
	satOK(t, e)
	if got != 127 || v.activityInc != 128 {
		t.Fatal("source wrap before threshold", got, v.activityInc)
	}
}

func TestSATSolverNativeTrailGrowthBoundsAndStableRoot(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	root := v.trailRef
	before, raw := v.budget.Used(), v.memory.State()
	v.budget.FailAfter(0)
	want := make([]Literal, 4097)
	for i := range want {
		want[i] = Literal(uint32(i) * 0x9e3779b9)
		satOK(t, v.trailPush(want[i]))
	}
	v.budget.FailAfter(-1)
	if v.trailRef != root || v.budget.Used() != before || v.memory.State() != raw {
		t.Fatal("native growth changed owner or allocator", v.trailRef, root)
	}
	for i, l := range want {
		got, e := v.trailRead(uint32(i))
		satOK(t, e)
		if got != l {
			t.Fatal(i, got, l)
		}
	}
	satOK(t, v.trail.Shrink(root, 1))
	_, e := v.trail.Read(root, 4096)
	bounds := error(arena.ErrBounds)
	if satStorageSourceDebug {
		bounds = ErrSATAssertion
	}
	if !errors.Is(e, bounds) {
		t.Fatal("discarded tail must not be readable", e, bounds)
	}
	if e = v.trail.Shrink(root, 2); !errors.Is(e, bounds) {
		t.Fatal("shrink cannot publish uninitialized values", e)
	}
	satOK(t, v.trailPush(DefaultLiteral()))
	got, e := v.trail.Read(root, 1)
	satOK(t, e)
	if got != DefaultLiteral() {
		t.Fatal(got)
	}
	satScalarAuditOwner(t, v)
	copied := *v.trail
	if _, e = copied.Size(root); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	foreign, _, _ := satVariableStateFixture(t)
	if _, e = foreign.trail.Size(root); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
}
func TestSATSolverNativeTrailAssignmentWithoutBackingReservation(t *testing.T) {
	s, v, b := satPropagationFixture(t, 1)
	v.scopeLevel = 1
	before, raw := b.Used(), v.memory.State()
	b.FailAfter(0)
	e := s.assignCore(0, satJustification{})
	b.FailAfter(-1)
	satOK(t, e)
	if v.assignment[0] != lbool.True || v.level[0] != 1 || v.phase[0] != 0 || !v.assignedSinceGC[0] {
		t.Fatal("source scalar assignment prefix")
	}
	n, e := v.trail.Size(v.trailRef)
	satOK(t, e)
	l, e := v.trail.Read(v.trailRef, 0)
	satOK(t, e)
	if n != 1 || l != 0 || b.Used() != before || v.memory.State() != raw {
		t.Fatal("native assignment trail changed allocator", n, l)
	}
	satScalarAuditRoots(t, v)
	satPropagationAssertGraph(t, s)
}
func TestSATSolverNativeTrailDoesNotReachRuntimeTLSCheckpoint(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	todo, e := newSATSolverUintSet(b)
	satOK(t, e)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true})
	satOK(t, e)
	v, e := newSATSolverVariableState(todo, b, arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{Budget: b}}, r)
	satOK(t, e)
	_, e = v.mkVar(false, true)
	satOK(t, e)
	// A Runtime allocation would reach its TLS checkpoint here. Native trail
	// growth must leave that borrowed allocator untouched.
	prefix, e := r.Allocate(uint64(100000 - r.State().LocalSize - 8))
	satOK(t, e)
	satOK(t, r.SetMaxSize(1))
	before := r.State()
	satOK(t, v.trailPush(MakeLiteral(0, true)))
	if r.State() != before {
		t.Fatal("native trail reached Runtime", r.State(), before)
	}
	satScalarAuditOwner(t, v, prefix)
	if n, e := v.trail.Size(v.trailRef); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	satOK(t, v.Close())
	satOK(t, r.Deallocate(&prefix))
	satOK(t, todo.Close())
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestSATSolverNativeActivityCurrentBorrowAfterGrowthAndColdGuards(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	for i := 0; i < 2; i++ {
		_, e := v.mkVar(false, true)
		satOK(t, e)
	}
	for i := 2; i < 8; i++ {
		_, e := v.mkVar(false, true)
		satOK(t, e)
	}
	used, raw := v.budget.Used(), v.memory.State()
	v.budget.FailAfter(0)
	satOK(t, v.activity.Set(0, 1))
	satOK(t, v.rescaleActivity())
	v.budget.FailAfter(-1)
	if v.budget.Used() != used || v.memory.State() != raw {
		t.Fatal("native activity mutation/rescaling changed allocation state")
	}
	satOK(t, v.activity.Set(7, 100))
	satOK(t, v.queue.ActivityIncreased(7))
	x, e := v.queue.NextVar()
	satOK(t, e)
	if x != 7 {
		t.Fatal("actual comparator borrowed current member", x)
	}
	copied := *v.activity
	if e = copied.Close(); !errors.Is(e, ErrSATOwner) {
		t.Fatal("copied cold activity owner", e)
	}
	if _, e = copied.Get(0); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satScalarAuditOwner(t, v)
}
