// Source: src/sat/sat_solver.h; src/sat/sat_solver.cpp; src/sat/sat_var_queue.h; src/util/vector.h; src/util/memory_manager.cpp.
package sat

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverScalarColumnsReachSameRuntime(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	_, e := v.mkVar(true, true)
	satOK(t, e)
	amount, e := v.memory.AllocationSize()
	satOK(t, e)
	if amount != 242+24+576 || v.memory.Stats().LiveBlocks != 14 {
		t.Fatal("all actual scalar source allocations", amount, v.memory.Stats())
	}
}

func satScalarTestUint(t *testing.T, c *RuntimeUintColumn, i uint32) uint32 {
	t.Helper()
	v, e := c.Read(i)
	satOK(t, e)
	return v
}
func satScalarTestUintSize(t *testing.T, c *RuntimeUintColumn) uint32 {
	t.Helper()
	v, e := c.Size()
	satOK(t, e)
	return v
}

func satScalarAuditRoots(t *testing.T, v *satSolverVariableState, external ...util.Block) {
	t.Helper()
	roots := append([]util.Block{}, external...)
	for _, c := range []*RuntimeVectorColumn{&v.decision, &v.mark, &v.literalMark, &v.eliminated, &v.external, &v.phase, &v.previousPhase, &v.assignedSinceGC, &v.level.Raw, &v.activity.values.Raw, &v.trail.values.Raw, &v.justification.raw} {
		if !c.PendingReplaces && c.Block != (util.Block{}) {
			roots = append(roots, c.Block)
		}
		if c.Pending != (util.Block{}) {
			roots = append(roots, c.Pending)
		}
	}
	if v.queue != nil && v.queue.runtimeBacked() {
		for _, c := range []*RuntimeVectorColumn{&v.queue.runtimeValues.Raw, &v.queue.runtimeIndices.Raw} {
			if !c.PendingReplaces && c.Block != (util.Block{}) {
				roots = append(roots, c.Block)
			}
			if c.Pending != (util.Block{}) {
				roots = append(roots, c.Pending)
			}
		}
	}
	satAssignmentAuditBlocks(t, &v.assignment, roots...)
}

func satScalarAuditOwner(t *testing.T, v *satSolverVariableState, external ...util.Block) {
	t.Helper()
	satScalarAuditRoots(t, v, external...)
	todo := v.elimTodo.Stats()
	todoBytes := todo.OwnerBytes
	if v.Stats().TotalBytes+todoBytes+v.memory.Stats().TotalBytes != v.budget.Used() {
		t.Fatal("once-only scalar graph", v.Stats(), v.memory.Stats(), v.budget.Used())
	}
}
func TestSATSolverScalarSourceBytesAndRetainedSlots(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	_, e := v.mkVar(true, true)
	satOK(t, e)
	for _, test := range []struct {
		c    *RuntimeVectorColumn
		want byte
		size uint32
	}{{&v.decision, 1, 1}, {&v.mark, 0, 1}, {&v.literalMark, 0, 2}, {&v.eliminated, 0, 1}, {&v.external, 1, 1}, {&v.phase, 2, 1}, {&v.previousPhase, 2, 1}, {&v.assignedSinceGC, 0, 1}} {
		if satAssignmentTestSize(t, test.c) != test.size {
			t.Fatal("source size")
		}
		raw, e := v.memory.Size(test.c.Block)
		satOK(t, e)
		if raw != 10 {
			t.Fatal(raw)
		}
		x, e := v.memory.Read(test.c.Block, 8)
		satOK(t, e)
		if x != test.want {
			t.Fatal(x, test.want)
		}
	}
	raw, e := v.memory.Size(v.level.Raw.Block)
	satOK(t, e)
	if raw != 16 {
		t.Fatal(raw)
	}
	for i := uint64(8); i < 12; i++ {
		x, e := v.memory.Read(v.level.Raw.Block, i)
		satOK(t, e)
		if x != 255 {
			t.Fatal(i, x)
		}
	}
	satScalarAuditOwner(t, v)
	u := &v.level
	_, e = u.Read(1)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, util.ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
	e = u.Write(1, 0x12345678)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else {
		satOK(t, e)
		if satScalarTestUint(t, u, 1) != 0x12345678 {
			t.Fatal("spare direct uint write")
		}
	}
	old := u.Raw.Block
	satOK(t, u.Push(17))
	satOK(t, u.Push(29))
	if _, e = v.memory.Read(old, 8); e == nil {
		t.Fatal("realloc captured generation")
	}
	satOK(t, u.Shrink(1))
	x, e := u.Read(2)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else {
		satOK(t, e)
		if x != 29 {
			t.Fatal(x)
		}
	}
	copied := *u
	if _, e = copied.Read(0); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satScalarAuditOwner(t, v)
}
func TestSATSolverScalarUnsignedGrowthLiteralWidths(t *testing.T) {
	for _, test := range []struct{ old, n, bytes uint32 }{{0, 2, 16}, {2, 3, 20}, {3, 5, 28}, {5, 8, 40}, {8, 12, 56}, {1 << 30, 1610612736, 2147483656}} {
		n, b, e := RuntimeScalarColumnNext(test.old, 4)
		satOK(t, e)
		if n != test.n || b != test.bytes {
			t.Fatal(test, n, b)
		}
	}
}

func TestSATSolverScalarLevelTLSFailureRetainsNamedPendingAndPrefix(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	todo, e := newSATSolverUintSet(b)
	satOK(t, e)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true})
	satOK(t, e)
	v, e := newSATSolverVariableState(todo, b, arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{Budget: b}}, r)
	satOK(t, e)
	prefix, e := r.Allocate(99256)
	satOK(t, e)
	satOK(t, r.SetMaxSize(1))
	_, e = v.mkVar(true, true)
	if !satAssignmentActualRefusal(r, e) || v.level.Raw.Pending == (util.Block{}) || v.level.Raw.Block != (util.Block{}) {
		t.Fatal("actual level pre-publication refusal", e, r.State())
	}
	if satAssignmentTestSize(t, &v.assignment) != 2 || satAssignmentTestSize(t, &v.external) != 1 || v.activity.Size() != 1 || satScalarTestUintSize(t, &v.level) != 0 || satAssignmentTestSize(t, &v.mark) != 0 {
		t.Fatal("literal source prefix")
	}
	satScalarAuditOwner(t, v, prefix)
	if _, e = r.Read(v.level.Raw.Pending, 0); !errors.Is(e, util.ErrRuntimeUninitialized) {
		t.Fatal("no source vector header publication", e)
	}
	satOK(t, v.Close())
	satOK(t, r.Deallocate(&prefix))
	satOK(t, todo.Close())
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestSATSolverScalarReverseMemberRawDestruction(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	_, e := v.mkVar(false, true)
	satOK(t, e)
	// Actual Runtime release cursors verify source reverse declaration order;
	// other arena-only members introduce no invented Runtime release events.
	order := []util.Block{v.queue.runtimeIndices.Raw.Block, v.queue.runtimeValues.Raw.Block, v.assignedSinceGC.Block, v.previousPhase.Block, v.phase.Block, v.activity.values.Raw.Block, v.level.Raw.Block, v.external.Block, v.eliminated.Block, v.literalMark.Block, v.mark.Block, v.decision.Block, v.justification.raw.Block, v.assignment.Block}
	satOK(t, v.Close())
	records := testFieldBySourceName(reflect.ValueOf(v.memory).Elem(), "blocks")
	for i, h := range order {
		slot := testFieldBySourceName(reflect.ValueOf(h), "slot").Uint()
		b := records.Index(int(slot - 1))
		if testFieldBySourceName(b, "live").Bool() || testFieldBySourceName(testFieldBySourceName(b, "release"), "serial").Uint() != uint64(i+1) {
			t.Fatal("actual destructor source order", i, testFieldBySourceName(testFieldBySourceName(b, "release"), "serial").Uint())
		}
	}
	if v.memory.Stats().LiveBlocks != 0 {
		t.Fatal("owned scalar roots retained")
	}
}
