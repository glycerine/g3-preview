// Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func TestSATSolverAssignmentAllocationReachesActualCheckpoint(t *testing.T) {
	s, v, _ := satPropagationFixture(t, 1)
	s.config.state.MaxMemory = 0
	for i := 0; i < 9; i++ {
		satOK(t, s.checkpoint())
	}
	if e := s.checkpoint(); !errors.Is(e, ErrSATSolverMaxMemory) {
		t.Fatal("actual assignment malloc absent from same Runtime", e, v.numVars())
	}
}

func TestSATSolverAssignmentRuntimeHeadersRetainedBytesAndRealloc(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	satOK(t, v.assignment.push(1))
	satOK(t, v.assignment.push(-1))
	c := &v.assignment
	n, e := c.size()
	satOK(t, e)
	cap, e := c.capacity()
	satOK(t, e)
	if n != 2 || cap != 2 {
		t.Fatal(n, cap)
	}
	raw, e := v.memory.Size(c.block)
	satOK(t, e)
	if raw != 10 {
		t.Fatal("source malloc request", raw)
	}
	amount, e := v.memory.AllocationSize()
	satOK(t, e)
	if amount != 18 {
		t.Fatal("actual malloc size_t header accounting", amount)
	}
	for i, want := range []byte{2, 0, 0, 0, 2, 0, 0, 0, 1, 255} {
		got, e := v.memory.Read(c.block, uint64(i))
		satOK(t, e)
		if got != want {
			t.Fatal(i, got, want)
		}
	}
	old := c.block
	satOK(t, c.push(0))
	if _, e = v.memory.Read(old, 0); e == nil {
		t.Fatal("realloc invalidates captured pointer")
	}
	satOK(t, c.shrink(1))
	got, e := c.read(1)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else {
		satOK(t, e)
		if got != -1 {
			t.Fatal(got)
		}
	}
}

func satAssignmentTestSize(t *testing.T, c *satSolverAssignmentColumn) uint32 {
	t.Helper()
	n, e := c.size()
	satOK(t, e)
	return n
}
func satAssignmentTestValue(t *testing.T, c *satSolverAssignmentColumn, i uint32) int8 {
	t.Helper()
	v, e := c.read(i)
	satOK(t, e)
	return v
}

func TestSATSolverAssignmentReleaseWriteAndShrinkSourceRoutes(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	c := &v.assignment
	e := c.shrink(7)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal("null source assertion", e)
		}
	} else {
		satOK(t, e)
	}
	for i := 0; i < 4; i++ {
		satOK(t, c.push(int8(i)))
	}
	_, e = c.read(4)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("uninitialized spare read", e)
	}
	e = c.write(4, 9)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else {
		satOK(t, e)
		x, e := c.read(4)
		satOK(t, e)
		if x != 9 {
			t.Fatal(x)
		}
	}
	e = c.shrink(99)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || satAssignmentTestSize(t, c) != 4 {
			t.Fatal(e)
		}
	} else {
		satOK(t, e)
		if satAssignmentTestSize(t, c) != 99 {
			t.Fatal("literal header publication")
		}
		x, e := c.read(0)
		satOK(t, e)
		if x != 0 {
			t.Fatal(x)
		}
	}
}
func TestSATSolverAssignmentRawValueReadDoesNotPrefetchSize(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	c := &v.assignment
	h, e := v.memory.Allocate(10)
	satOK(t, e)
	c.block = h
	// Explicit raw-state boundary, not a normally constructed source vector:
	// data is initialized while the source size word has not been written.
	satOK(t, v.memory.Write(h, 8, 17))
	x, e := c.read(0)
	if satStorageSourceDebug {
		if !errors.Is(e, memory.ErrRuntimeUninitialized) {
			t.Fatal("actual debug size read", e)
		}
	} else {
		satOK(t, e)
		if x != 17 {
			t.Fatal(x)
		}
	}
}

func satAssignmentActualRefusal(r *memory.Runtime, e error) bool {
	if errors.Is(e, arena.ErrInjectedFailure) {
		return true
	}
	var f memory.Failure
	return errors.As(e, &f) && f.ErrorCode() == 101 && r.State().OutOfMemory
}

// Enumerate only declared owning roots; never infer ownership from live entries.
func satAssignmentAuditBlocks(t *testing.T, c *satSolverAssignmentColumn, external ...memory.Block) {
	t.Helper()
	roots := append([]memory.Block{}, external...)
	if !c.pendingReplaces && c.block != (memory.Block{}) {
		roots = append(roots, c.block)
	}
	if c.pending != (memory.Block{}) {
		roots = append(roots, c.pending)
	}
	type identity struct{ slot, generation uint32 }
	expected := map[identity]bool{}
	rv := reflect.ValueOf(c.runtime).Elem()
	origin := rv.FieldByName("origin").Uint()
	for _, h := range roots {
		hv := reflect.ValueOf(h)
		if hv.FieldByName("origin").Uint() != origin {
			t.Fatal("foreign declared root")
		}
		id := identity{uint32(hv.FieldByName("slot").Uint()), uint32(hv.FieldByName("generation").Uint())}
		if expected[id] {
			t.Fatal("duplicate owning root", id)
		}
		expected[id] = true
		_, e := c.runtime.Size(h)
		satOK(t, e)
	}
	records := rv.FieldByName("blocks")
	var physical, init uint64
	live := 0
	for i := 0; i < records.Len(); i++ {
		b := records.Index(i)
		if !b.FieldByName("live").Bool() {
			continue
		}
		id := identity{uint32(i + 1), uint32(b.FieldByName("generation").Uint())}
		if !expected[id] {
			t.Fatal("anonymous raw allocation", id)
		}
		delete(expected, id)
		live++
		physical += uint64(b.FieldByName("data").Len())
		init += uint64(b.FieldByName("initialized").Len()) * 8
	}
	st := c.runtime.Stats()
	if len(expected) != 0 || uint32(live) != st.LiveBlocks || physical != st.PhysicalBytes || init != st.InitializationBytes {
		t.Fatal("actual raw graph", expected, st, physical, init)
	}
}
func TestSATSolverAssignmentTLSAllocationFailureRetainsUnpublishedRoot(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	todo, e := newSATSolverUintSet(b)
	satOK(t, e)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true})
	satOK(t, e)
	v, e := newSATSolverVariableState(todo, b, arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{Budget: b}}, r)
	satOK(t, e)
	prefix, e := r.Allocate(99990)
	satOK(t, e)
	satOK(t, r.SetMaxSize(1))
	_, e = v.mkVar(false, true)
	if !satAssignmentActualRefusal(r, e) || v.assignment.block != (memory.Block{}) || v.assignment.pending == (memory.Block{}) {
		t.Fatal("post-malloc/pre-m_data prefix", e, r.State())
	}
	satAssignmentAuditBlocks(t, &v.assignment, prefix)
	if _, e = r.Read(v.assignment.pending, 0); !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("source header must not publish", e)
	}
	todoBytes := todo.Stats().OwnerBytes + todo.Stats().MembershipCapacity + 4*todo.Stats().ValuesCapacity
	if v.Stats().TotalBytes+todoBytes+r.Stats().TotalBytes != b.Used() {
		t.Fatal("once-only physical cohort", b.Used(), v.Stats(), r.Stats())
	}
	satOK(t, v.Close())
	satOK(t, r.Deallocate(&prefix))
	satOK(t, todo.Close())
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestSATSolverAssignmentReallocateRefusalKeepsActualOldBinding(t *testing.T) {
	v, _, b := satVariableStateFixture(t)
	c := &v.assignment
	satOK(t, c.push(7))
	satOK(t, c.push(-3))
	old := c.block
	before := v.memory.State()
	used := b.Used()
	b.FailAfter(0)
	e := c.push(9)
	if !satAssignmentActualRefusal(v.memory, e) || c.block != old || c.pending != (memory.Block{}) {
		t.Fatal(e, c, v.memory.State())
	}
	satAssignmentAuditBlocks(t, c)
	if b.Used() != used || satAssignmentTestSize(t, c) != 2 || satAssignmentTestValue(t, c, 0) != 7 || satAssignmentTestValue(t, c, 1) != -3 {
		t.Fatal("physical old graph changed")
	}
	after := v.memory.State()
	if after.GlobalSize != before.GlobalSize+1 || after.GlobalCount != before.GlobalCount+1 {
		t.Fatal("literal accounting-before-realloc prefix", before, after)
	}
	b.FailAfter(-1)
}
func TestSATSolverAssignmentCurrentMemberBorrowAndOwnerGuards(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	satOK(t, v.assignment.push(int8(lTrue)))
	view := v.clauseState()
	old := v.assignment.block
	satOK(t, v.assignment.push(int8(lUndef)))
	satOK(t, v.assignment.push(int8(lFalse)))
	satOK(t, v.assignment.write(0, int8(lFalse)))
	x, e := view.value(0)
	satOK(t, e)
	if x != lFalse {
		t.Fatal("stale member shadow", x)
	}
	if _, e = v.memory.Read(old, 8); e == nil {
		t.Fatal("old physical pointer survived")
	}
	copied := v.assignment
	if _, e = copied.read(0); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satAssignmentAuditBlocks(t, &v.assignment)
}
func TestSATSolverAssignmentSourceUnsignedGrowthException(t *testing.T) {
	for _, test := range []struct{ old, next, bytes uint32 }{{0, 2, 10}, {2, 3, 11}, {3, 5, 13}, {5, 8, 16}, {8, 12, 20}} {
		n, b, e := satAssignmentNext(test.old)
		satOK(t, e)
		if n != test.next || b != test.bytes {
			t.Fatal(test, n, b)
		}
	}
	_, _, e := satAssignmentNext(0x80000000)
	var d DefaultException
	if !errors.As(e, &d) || d.Error() != "Overflow encountered when expanding vector" {
		t.Fatal("source default_exception", e)
	}
}

func TestSATSolverAssignmentNonNullZeroCapacityActualExpandRoute(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	c := &v.assignment
	h, e := v.memory.Allocate(8)
	satOK(t, e)
	c.block = h
	satOK(t, satAssignmentWriteWord(v.memory, h, 0, 0))
	satOK(t, satAssignmentWriteWord(v.memory, h, 4, 0))
	before := v.memory.State()
	e = c.push(1)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal("nonnull source capacity assertion", e)
		}
	} else {
		var d DefaultException
		if !errors.As(e, &d) || d.Error() != "Overflow encountered when expanding vector" {
			t.Fatal("nonnull literal overflow", e)
		}
	}
	if c.block != h || v.memory.State() != before {
		t.Fatal("error precedes allocation", v.memory.State(), before)
	}
}

func TestSATSolverAssignmentRequiresSameActualCheckpointRuntime(t *testing.T) {
	s, v, b := satPropagationFixture(t, 1)
	foreign, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	satOK(t, e)
	before := b.Used()
	got, e := newSATSolverPropagationState(v, s.clauses, s.converter, s.limit, foreign, s.config, b)
	if got != nil || !errors.Is(e, ErrSATOwner) || b.Used() != before {
		t.Fatal("foreign checkpoint domain", got, e, b.Used(), before)
	}
	satOK(t, foreign.Close())
}
