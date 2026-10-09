// Source: src/sat/sat_solver.h; src/sat/sat_solver.cpp; src/util/vector.h; src/util/memory_manager.cpp.
package sat

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/lbool"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverAssignmentAllocationReachesActualCheckpoint(t *testing.T) {
	s, v, _ := satPropagationFixture(t, 1)
	s.config.state.MaxMemory = 0
	for i := 0; i < 9; i++ {
		satOK(t, s.checkpoint())
	}
	if e := s.checkpoint(); !errors.Is(e, ErrSATSolverMaxMemory) {
		t.Fatal("actual assignment malloc absent from same Runtime", e, v.NumVars())
	}
}

func TestSATSolverAssignmentRuntimeHeadersRetainedBytesAndRealloc(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	satOK(t, v.assignment.Push(1))
	satOK(t, v.assignment.Push(-1))
	c := &v.assignment
	n, e := c.Size()
	satOK(t, e)
	cap, e := c.Capacity()
	satOK(t, e)
	if n != 2 || cap != 2 {
		t.Fatal(n, cap)
	}
	raw, e := v.memory.Size(c.Block)
	satOK(t, e)
	if raw != 10 {
		t.Fatal("source malloc request", raw)
	}
	amount, e := v.memory.AllocationSize()
	satOK(t, e)
	if amount != 18+24+576 {
		t.Fatal("actual malloc size_t header accounting", amount)
	}
	for i, want := range []byte{2, 0, 0, 0, 2, 0, 0, 0, 1, 255} {
		got, e := v.memory.Read(c.Block, uint64(i))
		satOK(t, e)
		if got != want {
			t.Fatal(i, got, want)
		}
	}
	old := c.Block
	satOK(t, c.Push(0))
	if _, e = v.memory.Read(old, 0); e == nil {
		t.Fatal("realloc invalidates captured pointer")
	}
	satOK(t, c.Shrink(1))
	got, e := c.Read(1)
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

func satAssignmentTestSize(t *testing.T, c *RuntimeVectorColumn) uint32 {
	t.Helper()
	n, e := c.Size()
	satOK(t, e)
	return n
}
func satAssignmentTestValue(t *testing.T, c *RuntimeVectorColumn, i uint32) int8 {
	t.Helper()
	v, e := c.Read(i)
	satOK(t, e)
	return v
}

func TestSATSolverAssignmentReleaseWriteAndShrinkSourceRoutes(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	c := &v.assignment
	e := c.Shrink(7)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal("null source assertion", e)
		}
	} else {
		satOK(t, e)
	}
	for i := 0; i < 4; i++ {
		satOK(t, c.Push(int8(i)))
	}
	_, e = c.Read(4)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, util.ErrRuntimeUninitialized) {
		t.Fatal("uninitialized spare read", e)
	}
	e = c.Write(4, 9)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else {
		satOK(t, e)
		x, e := c.Read(4)
		satOK(t, e)
		if x != 9 {
			t.Fatal(x)
		}
	}
	e = c.Shrink(99)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || satAssignmentTestSize(t, c) != 4 {
			t.Fatal(e)
		}
	} else {
		satOK(t, e)
		if satAssignmentTestSize(t, c) != 99 {
			t.Fatal("literal header publication")
		}
		x, e := c.Read(0)
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
	c.Block = h
	// Explicit raw-state boundary, not a normally constructed source vector:
	// data is initialized while the source size word has not been written.
	satOK(t, v.memory.Write(h, 8, 17))
	x, e := c.Read(0)
	if satStorageSourceDebug {
		if !errors.Is(e, util.ErrRuntimeUninitialized) {
			t.Fatal("actual debug size read", e)
		}
	} else {
		satOK(t, e)
		if x != 17 {
			t.Fatal(x)
		}
	}
}

func satAssignmentActualRefusal(r *util.Runtime, e error) bool {
	if errors.Is(e, arena.ErrInjectedFailure) {
		return true
	}
	var f util.Failure
	return errors.As(e, &f) && f.ErrorCode() == 101 && r.State().OutOfMemory
}

// Enumerate only declared owning roots; never infer ownership from live entries.
func satAssignmentAuditBlocks(t *testing.T, c *RuntimeVectorColumn, external ...util.Block) {
	t.Helper()
	roots := append([]util.Block{}, external...)
	if !c.PendingReplaces && c.Block != (util.Block{}) {
		roots = append(roots, c.Block)
	}
	if c.Pending != (util.Block{}) {
		roots = append(roots, c.Pending)
	}
	type identity struct{ slot, generation uint32 }
	expected := map[identity]bool{}
	rv := reflect.ValueOf(c.Runtime).Elem()
	origin := testFieldBySourceName(rv, "origin").Uint()
	for _, h := range roots {
		hv := reflect.ValueOf(h)
		if testFieldBySourceName(hv, "origin").Uint() != origin {
			t.Fatal("foreign declared root")
		}
		id := identity{uint32(testFieldBySourceName(hv, "slot").Uint()), uint32(testFieldBySourceName(hv, "generation").Uint())}
		if expected[id] {
			t.Fatal("duplicate owning root", id)
		}
		expected[id] = true
		_, e := c.Runtime.Size(h)
		satOK(t, e)
	}
	records := testFieldBySourceName(rv, "blocks")
	var physical, init uint64
	live := 0
	for i := 0; i < records.Len(); i++ {
		b := records.Index(i)
		if !testFieldBySourceName(b, "live").Bool() {
			continue
		}
		id := identity{uint32(i + 1), uint32(testFieldBySourceName(b, "generation").Uint())}
		if !expected[id] {
			t.Fatal("anonymous raw allocation", id)
		}
		delete(expected, id)
		live++
		physical += uint64(testFieldBySourceName(b, "data").Len())
		init += uint64(testFieldBySourceName(b, "initialized").Len()) * 8
	}
	st := c.Runtime.Stats()
	if len(expected) != 0 || uint32(live) != st.LiveBlocks || physical != st.PhysicalBytes || init != st.InitializationBytes {
		t.Fatal("actual raw graph", expected, st, physical, init)
	}
}
func TestSATSolverAssignmentTLSAllocationFailureRetainsUnpublishedRoot(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	todo, e := newSATSolverUintSet(b)
	satOK(t, e)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true})
	satOK(t, e)
	v, e := newSATSolverVariableState(todo, b, arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{Budget: b}}, r)
	satOK(t, e)
	prefix, e := r.Allocate(99390)
	satOK(t, e)
	satOK(t, r.SetMaxSize(1))
	_, e = v.mkVar(false, true)
	if !satAssignmentActualRefusal(r, e) || v.assignment.Block != (util.Block{}) || v.assignment.Pending == (util.Block{}) {
		t.Fatal("post-malloc/pre-m_data prefix", e, r.State())
	}
	satScalarAuditRoots(t, v, prefix)
	if _, e = r.Read(v.assignment.Pending, 0); !errors.Is(e, util.ErrRuntimeUninitialized) {
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
	satOK(t, c.Push(7))
	satOK(t, c.Push(-3))
	old := c.Block
	before := v.memory.State()
	used := b.Used()
	b.FailAfter(0)
	e := c.Push(9)
	if !satAssignmentActualRefusal(v.memory, e) || c.Block != old || c.Pending != (util.Block{}) {
		t.Fatal(e, c, v.memory.State())
	}
	satScalarAuditRoots(t, v)
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
	satOK(t, v.assignment.Push(int8(lbool.True)))
	view := v.clauseState()
	old := v.assignment.Block
	satOK(t, v.assignment.Push(int8(lbool.Undef)))
	satOK(t, v.assignment.Push(int8(lbool.False)))
	satOK(t, v.assignment.Write(0, int8(lbool.False)))
	x, e := view.value(0)
	satOK(t, e)
	if x != lbool.False {
		t.Fatal("stale member shadow", x)
	}
	if _, e = v.memory.Read(old, 8); e == nil {
		t.Fatal("old physical pointer survived")
	}
	copied := v.assignment
	if _, e = copied.Read(0); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satScalarAuditRoots(t, v)
}
func TestSATSolverAssignmentSourceUnsignedGrowthException(t *testing.T) {
	for _, test := range []struct{ old, next, bytes uint32 }{{0, 2, 10}, {2, 3, 11}, {3, 5, 13}, {5, 8, 16}, {8, 12, 20}} {
		n, b, e := RuntimeByteColumnNext(test.old)
		satOK(t, e)
		if n != test.next || b != test.bytes {
			t.Fatal(test, n, b)
		}
	}
	_, _, e := RuntimeByteColumnNext(0x80000000)
	var d util.DefaultException
	if !errors.As(e, &d) || d.Error() != "Overflow encountered when expanding vector" {
		t.Fatal("source default_exception", e)
	}
}

func TestSATSolverAssignmentNonNullZeroCapacityActualExpandRoute(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	c := &v.assignment
	h, e := v.memory.Allocate(8)
	satOK(t, e)
	c.Block = h
	satOK(t, WriteRuntimeWord(v.memory, h, 0, 0))
	satOK(t, WriteRuntimeWord(v.memory, h, 4, 0))
	before := v.memory.State()
	e = c.Push(1)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal("nonnull source capacity assertion", e)
		}
	} else {
		var d util.DefaultException
		if !errors.As(e, &d) || d.Error() != "Overflow encountered when expanding vector" {
			t.Fatal("nonnull literal overflow", e)
		}
	}
	if c.Block != h || v.memory.State() != before {
		t.Fatal("error precedes allocation", v.memory.State(), before)
	}
}

func TestSATSolverAssignmentRequiresSameActualCheckpointRuntime(t *testing.T) {
	s, v, b := satPropagationFixture(t, 1)
	foreign, e := util.NewRuntime(util.RuntimeConfig{Budget: b})
	satOK(t, e)
	before := b.Used()
	got, e := newSATSolverPropagationState(v, s.clauses, s.converter, s.limit, foreign, s.config, b)
	if got != nil || !errors.Is(e, ErrSATOwner) || b.Used() != before {
		t.Fatal("foreign checkpoint domain", got, e, b.Used(), before)
	}
	satOK(t, foreign.Close())
}
