// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func TestSATSolverScalarColumnsReachSameRuntime(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	_, e := v.mkVar(true, true)
	satOK(t, e)
	amount, e := v.memory.AllocationSize()
	satOK(t, e)
	if amount != 186 || v.memory.Stats().LiveBlocks != 10 {
		t.Fatal("all actual scalar source allocations", amount, v.memory.Stats())
	}
}

func satScalarTestUint(t *testing.T, c *satSolverRuntimeUintColumn, i uint32) uint32 {
	t.Helper()
	v, e := c.read(i)
	satOK(t, e)
	return v
}
func satScalarTestUintSize(t *testing.T, c *satSolverRuntimeUintColumn) uint32 {
	t.Helper()
	v, e := c.size()
	satOK(t, e)
	return v
}

func satScalarAuditOwner(t *testing.T, v *satSolverVariableState, external ...memory.Block) {
	t.Helper()
	roots := append([]memory.Block{}, external...)
	for _, c := range []*satSolverAssignmentColumn{&v.decision, &v.mark, &v.literalMark, &v.eliminated, &v.external, &v.phase, &v.previousPhase, &v.assignedSinceGC, &v.level.raw} {
		if !c.pendingReplaces && c.block != (memory.Block{}) {
			roots = append(roots, c.block)
		}
		if c.pending != (memory.Block{}) {
			roots = append(roots, c.pending)
		}
	}
	satAssignmentAuditBlocks(t, &v.assignment, roots...)
	todo := v.elimTodo.Stats()
	todoBytes := todo.OwnerBytes + todo.MembershipCapacity + 4*todo.ValuesCapacity
	if v.Stats().TotalBytes+todoBytes+v.memory.Stats().TotalBytes != v.budget.Used() {
		t.Fatal("once-only scalar graph", v.Stats(), v.memory.Stats(), v.budget.Used())
	}
}
func TestSATSolverScalarSourceBytesAndRetainedSlots(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	_, e := v.mkVar(true, true)
	satOK(t, e)
	for _, test := range []struct {
		c    *satSolverAssignmentColumn
		want byte
		size uint32
	}{{&v.decision, 1, 1}, {&v.mark, 0, 1}, {&v.literalMark, 0, 2}, {&v.eliminated, 0, 1}, {&v.external, 1, 1}, {&v.phase, 2, 1}, {&v.previousPhase, 2, 1}, {&v.assignedSinceGC, 0, 1}} {
		if satAssignmentTestSize(t, test.c) != test.size {
			t.Fatal("source size")
		}
		raw, e := v.memory.Size(test.c.block)
		satOK(t, e)
		if raw != 10 {
			t.Fatal(raw)
		}
		x, e := v.memory.Read(test.c.block, 8)
		satOK(t, e)
		if x != test.want {
			t.Fatal(x, test.want)
		}
	}
	raw, e := v.memory.Size(v.level.raw.block)
	satOK(t, e)
	if raw != 16 {
		t.Fatal(raw)
	}
	for i := uint64(8); i < 12; i++ {
		x, e := v.memory.Read(v.level.raw.block, i)
		satOK(t, e)
		if x != 255 {
			t.Fatal(i, x)
		}
	}
	satScalarAuditOwner(t, v)
	u := &v.level
	_, e = u.read(1)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
	e = u.write(1, 0x12345678)
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
	old := u.raw.block
	satOK(t, u.push(17))
	satOK(t, u.push(29))
	if _, e = v.memory.Read(old, 8); e == nil {
		t.Fatal("realloc captured generation")
	}
	satOK(t, u.shrink(1))
	x, e := u.read(2)
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
	if _, e = copied.read(0); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satScalarAuditOwner(t, v)
}
func TestSATSolverScalarUnsignedGrowthLiteralWidths(t *testing.T) {
	for _, test := range []struct{ old, n, bytes uint32 }{{0, 2, 16}, {2, 3, 20}, {3, 5, 28}, {5, 8, 40}, {8, 12, 56}, {1 << 30, 1610612736, 2147483656}} {
		n, b, e := satScalarColumnNext(test.old, 4)
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
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true})
	satOK(t, e)
	v, e := newSATSolverVariableState(todo, b, arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{Budget: b}}, r)
	satOK(t, e)
	prefix, e := r.Allocate(99912)
	satOK(t, e)
	satOK(t, r.SetMaxSize(1))
	_, e = v.mkVar(true, true)
	if !satAssignmentActualRefusal(r, e) || v.level.raw.pending == (memory.Block{}) || v.level.raw.block != (memory.Block{}) {
		t.Fatal("actual level pre-publication refusal", e, r.State())
	}
	if satAssignmentTestSize(t, &v.assignment) != 2 || satAssignmentTestSize(t, &v.external) != 1 || v.activity.Size() != 1 || satScalarTestUintSize(t, &v.level) != 0 || satAssignmentTestSize(t, &v.mark) != 0 {
		t.Fatal("literal source prefix")
	}
	satScalarAuditOwner(t, v, prefix)
	if _, e = r.Read(v.level.raw.pending, 0); !errors.Is(e, memory.ErrRuntimeUninitialized) {
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
	order := []memory.Block{v.assignedSinceGC.block, v.previousPhase.block, v.phase.block, v.level.raw.block, v.external.block, v.eliminated.block, v.literalMark.block, v.mark.block, v.decision.block, v.assignment.block}
	satOK(t, v.Close())
	records := reflect.ValueOf(v.memory).Elem().FieldByName("blocks")
	for i, h := range order {
		slot := reflect.ValueOf(h).FieldByName("slot").Uint()
		b := records.Index(int(slot - 1))
		if b.FieldByName("live").Bool() || b.FieldByName("release").FieldByName("serial").Uint() != uint64(i+1) {
			t.Fatal("actual destructor source order", i, b.FieldByName("release").FieldByName("serial").Uint())
		}
	}
	if v.memory.Stats().LiveBlocks != 0 {
		t.Fatal("owned scalar roots retained")
	}
}
