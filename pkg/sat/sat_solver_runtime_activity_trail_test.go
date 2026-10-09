// Source: src/sat/sat_solver.h; src/sat/sat_solver.cpp; src/sat/sat_var_queue.h; src/util/vector.h; src/util/memory_manager.cpp.
package sat

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/util/lbool"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverActivityAndLiteralTrailUseActualRuntime(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	_, e := v.mkVar(false, true)
	satOK(t, e)
	n, e := v.memory.AllocationSize()
	satOK(t, e)
	if n != 98+600 || v.memory.Stats().LiveBlocks != 6 {
		t.Fatal("actual activity source allocation missing", n, v.memory.Stats())
	}
	satOK(t, v.trailPush(MakeLiteral(0, true)))
	n, e = v.memory.AllocationSize()
	satOK(t, e)
	if n != 122+600 || v.memory.Stats().LiveBlocks != 7 {
		t.Fatal("actual literal32 trail allocation missing", n, v.memory.Stats())
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
		t.Fatal("real current Runtime activity comparator", next)
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

func TestSATSolverRuntimeActivityQueueCopiesKeepConcreteBorrow(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	for i := 0; i < 3; i++ {
		_, e := v.mkVar(false, true)
		satOK(t, e)
	}
	satOK(t, v.activity.Set(2, 99))
	satOK(t, v.queue.ActivityIncreased(2))
	copy, e := v.queue.Clone(v.budget)
	satOK(t, e)
	if copy.runtimeActivity != v.activity {
		satOK(t, copy.Close())
		t.Fatal("source copied comparator lost actual borrowed activity")
	}
	x, e := copy.NextVar()
	satOK(t, e)
	if x != 2 {
		t.Fatal(x)
	}
	satOK(t, copy.Close())
	moved, e := v.queue.MoveClone(v.budget)
	satOK(t, e)
	if moved.runtimeActivity != v.activity {
		satOK(t, moved.Close())
		t.Fatal("source moved comparator lost actual borrowed activity")
	}
	x, e = moved.NextVar()
	satOK(t, e)
	if x != 2 {
		t.Fatal(x)
	}
	satOK(t, moved.Close())
}

func TestSATSolverRuntimeTrailLiteral32AndCapturedBacking(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	literals := []Literal{0, 1, 0xfffffffe, 0xffffffff, 1025}
	satOK(t, v.trailPush(literals[0]))
	root := v.trailRef
	old := v.trail.values.Raw.Block
	for _, l := range literals[1:] {
		satOK(t, v.trailPush(l))
	}
	if v.trailRef != root {
		t.Fatal("source vector object identity moved")
	}
	if _, e := v.memory.Read(old, 8); e == nil {
		t.Fatal("captured realloc pointer remained live")
	}
	n, e := v.memory.Size(v.trail.values.Raw.Block)
	satOK(t, e)
	if n != 28 {
		t.Fatal("literal32 source capacity5 extent", n)
	}
	for i, l := range literals {
		got, e := v.trailRead(uint32(i))
		satOK(t, e)
		if got != l {
			t.Fatal(i, got, l)
		}
		for k := uint64(0); k < 4; k++ {
			b, e := v.memory.Read(v.trail.values.Raw.Block, 8+4*uint64(i)+k)
			satOK(t, e)
			if b != byte(uint32(l)>>(8*k)) {
				t.Fatal("actual literal byte", i, k, b)
			}
		}
	}
	satOK(t, v.trail.Shrink(root, 1))
	l, e := v.trail.Read(root, 4)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else {
		satOK(t, e)
		if l != 1025 {
			t.Fatal(l)
		}
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
func TestSATSolverRuntimeTrailAllActualRefusalPrefixes(t *testing.T) {
	failures, success := 0, 0
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			s, v, b := satPropagationFixture(t, 1)
			v.scopeLevel = 1
			b.FailAfter(ordinal)
			e := s.assignCore(0, satJustification{})
			satScalarAuditRoots(t, v)
			satPropagationAssertGraph(t, s)
			if satAssignmentTestValue(t, &v.assignment, 0) != int8(lbool.True) || satScalarTestUint(t, &v.level, 0) != 1 || v.phase[0] != 0 || !v.assignedSinceGC[0] {
				t.Fatal("source scalar prefix before trail")
			}
			n, x := v.trail.Size(v.trailRef)
			satOK(t, x)
			if e != nil {
				failures++
				if !satAssignmentActualRefusal(v.memory, e) || n != 0 {
					t.Fatal(ordinal, e, n)
				}
			} else {
				success++
				if n != 1 {
					t.Fatal(n)
				}
			}
			b.FailAfter(-1)
		})
		if success != 0 {
			break
		}
	}
	if failures == 0 || success != 1 {
		t.Fatal(failures, success)
	}
	t.Log("actual trail refusal horizon", failures, success)
}
func TestSATSolverRuntimeActivityTLSFailurePendingSourcePrefix(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	todo, e := newSATSolverUintSet(b)
	satOK(t, e)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true})
	satOK(t, e)
	v, e := newSATSolverVariableState(todo, b, arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{Budget: b}}, r)
	satOK(t, e)
	prefix, e := r.Allocate(99334)
	satOK(t, e)
	satOK(t, r.SetMaxSize(1))
	_, e = v.mkVar(true, true)
	if !satAssignmentActualRefusal(r, e) || v.activity.values.Raw.Pending == (util.Block{}) || v.activity.values.Raw.Block != (util.Block{}) {
		t.Fatal("actual activity owned malloc error", e, r.State())
	}
	if uint32(len(v.external)) != 1 || v.activity.Size() != 0 || satScalarTestUintSize(t, &v.level) != 0 {
		t.Fatal("source activity/level publication sequence")
	}
	satScalarAuditOwner(t, v, prefix)
	if _, e = r.Read(v.activity.values.Raw.Pending, 0); !errors.Is(e, util.ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
	satOK(t, v.Close())
	satOK(t, r.Deallocate(&prefix))
	satOK(t, todo.Close())
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestSATSolverRuntimeTrailTLSPendingPreservesVectorPrefix(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	todo, e := newSATSolverUintSet(b)
	satOK(t, e)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true})
	satOK(t, e)
	v, e := newSATSolverVariableState(todo, b, arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{Budget: b}}, r)
	satOK(t, e)
	_, e = v.mkVar(false, true)
	satOK(t, e)
	// Reach the actual TLS checkpoint using the retained Runtime extent.
	// Native mark slices are outside this accounting.
	prefix, e := r.Allocate(uint64(100000 - r.State().LocalSize - 8))
	satOK(t, e)
	satOK(t, r.SetMaxSize(1))
	e = v.trailPush(MakeLiteral(0, true))
	if !satAssignmentActualRefusal(r, e) || v.trail.values.Raw.Pending == (util.Block{}) || v.trail.values.Raw.Block != (util.Block{}) {
		t.Fatal(e, r.State())
	}
	satScalarAuditOwner(t, v, prefix)
	if n, e := v.trail.Size(v.trailRef); e != nil || n != 0 {
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

func TestSATSolverRuntimeActivityCurrentBorrowAfterGrowthAndColdGuards(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	for i := 0; i < 2; i++ {
		_, e := v.mkVar(false, true)
		satOK(t, e)
	}
	old := v.activity.values.Raw.Block
	for i := 2; i < 8; i++ {
		_, e := v.mkVar(false, true)
		satOK(t, e)
	}
	if _, e := v.memory.Read(old, 8); e == nil {
		t.Fatal("captured old activity backing")
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
