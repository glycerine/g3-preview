// Source: src/sat/sat_var_queue.h:31-67; src/util/heap.h:24-257.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2011-05-21.
package g3

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
	"testing"
)

func TestSATCaseSplitHeapConstructorRealRuntimeAccounting(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	before := s.runtime.memory.Stats().LiveBlocks
	a, e := newSATSolverRuntimeActivity(s.budget, s.runtime.memory)
	satOK(t, e)
	q, e := newSATSolverRuntimeVarQueue(a, s.budget)
	satOK(t, e)
	// Actual heap ctor constructs source values {-1} and indices resized128.
	// These must be two real Runtime allocations, never just Budget events.
	after := s.runtime.memory.Stats().LiveBlocks
	t.Cleanup(func() { satOK(t, q.Close()); satOK(t, a.Close()) })
	if after != before+2 {
		t.Fatal("source heap two actual Runtime vector backings missing", before, after)
	}
}

func TestSATCaseSplitHeapSourceHeadersAndNoShadow(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	q := s.variables.queue
	if len(q.values) != 0 || cap(q.values) != 0 || len(q.indices) != 0 || cap(q.indices) != 0 {
		t.Fatal("actual solver queue still owns shadow slices")
	}
	n, e := q.runtimeValues.size()
	satOK(t, e)
	if n != 1 {
		t.Fatal(n)
	}
	n, e = q.runtimeValues.capacity()
	satOK(t, e)
	if n != 2 {
		t.Fatal(n)
	}
	x, e := q.runtimeValues.read(0)
	satOK(t, e)
	if int32(x) != -1 {
		t.Fatal(x)
	}
	n, e = q.runtimeIndices.size()
	satOK(t, e)
	if n != 128 {
		t.Fatal(n)
	}
	n, e = q.runtimeIndices.capacity()
	satOK(t, e)
	if n != 140 {
		t.Fatal(n)
	}
	for i := uint32(0); i < 128; i++ {
		x, e := q.runtimeIndices.read(i)
		satOK(t, e)
		if x != 0 {
			t.Fatal(i, x)
		}
	}
	empty, e := q.emptyChecked()
	satOK(t, e)
	if !empty {
		t.Fatal(empty)
	}
	size, e := q.sizeChecked()
	satOK(t, e)
	if size != 0 {
		t.Fatal(size)
	}
	present, e := q.containsChecked(0)
	satOK(t, e)
	if present {
		t.Fatal(present)
	}
	satFullSolverRuntimeAudit(t, s)
}

func TestSATCaseSplitHeapIndependentFiniteActivityOracle(t *testing.T) {
	for code := 0; code < 81; code++ {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			q := s.variables.queue
			var weight [4]uint32
			var active [4]bool
			n := code
			for i := 0; i < 4; i++ {
				_, e := s.MkVar(false, true)
				satOK(t, e)
				weight[i] = uint32(n % 3)
				n /= 3
				active[i] = true
				satOK(t, s.variables.activity.Set(uint32(i), weight[i]))
				satOK(t, q.ActivityIncreased(satBoolVar(i)))
			}
			for left := 4; left > 0; left-- {
				v, e := q.NextVar()
				satOK(t, e)
				if uint32(v) >= 4 || !active[v] {
					t.Fatal(v, active)
				}
				for i := 0; i < 4; i++ {
					if active[i] && weight[i] > weight[v] {
						t.Fatal("not maximal actual activity", v, weight, active)
					}
				}
				active[v] = false
				present, e := q.containsChecked(v)
				satOK(t, e)
				if present {
					t.Fatal("not erased", v)
				}
				satFullSolverRuntimeAudit(t, s)
			}
			empty, e := q.emptyChecked()
			satOK(t, e)
			if !empty {
				t.Fatal(empty)
			}
			satOK(t, q.UnassignVar(2))
			satOK(t, q.UnassignVar(0))
			satOK(t, q.DelVar(2))
			present, e := q.containsChecked(2)
			satOK(t, e)
			if present {
				t.Fatal(present)
			}
			satOK(t, q.Reset())
			empty, e = q.emptyChecked()
			satOK(t, e)
			if !empty {
				t.Fatal(empty)
			}
			satFullSolverRuntimeAudit(t, s)
		})
	}
}

// Source insert publishes the reverse index before a refusing values.push.
func TestSATCaseSplitHeapReverseIndexBeforeValueRefusal(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	q := s.variables.queue
	_, e := s.MkVar(false, true)
	satOK(t, e)
	// Source values now contains sentinel and var0; next push must grow.
	before, e := q.runtimeValues.size()
	satOK(t, e)
	if before != 2 {
		t.Fatal(before)
	}
	s.budget.FailAfter(0)
	e = q.insertRuntime(1)
	var failure memory.Failure
	if !errors.As(e, &failure) || failure.ErrorCode() != 101 || !s.runtime.memory.State().OutOfMemory {
		t.Fatal("real raw refusal", e, s.runtime.memory.State())
	}
	n, readErr := q.runtimeValues.size()
	satOK(t, readErr)
	index, readErr := q.runtimeIndices.read(1)
	satOK(t, readErr)
	if n != before || index != before {
		t.Fatal("indices-before-values source prefix", n, index, before)
	}
	satFullSolverRuntimeAudit(t, s)
	s.budget.FailAfter(-1)
}

func TestSATCaseSplitHeapCloneMoveRealBackingLifetime(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 4; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	q := s.variables.queue
	clone, e := q.Clone(s.budget)
	satOK(t, e)
	original := q.runtimeValues.raw.block
	if clone.runtimeValues.raw.block == original {
		t.Fatal("copy must own actual distinct backing")
	}
	satOK(t, clone.Close())
	if _, e = s.runtime.memory.Read(original, 0); e != nil {
		t.Fatal("copy close freed source", e)
	}
	moved, e := q.MoveClone(s.budget)
	satOK(t, e)
	if moved.runtimeValues.raw.block != original {
		t.Fatal("move cloned source allocation")
	}
	empty, e := q.emptyChecked()
	satOK(t, e)
	// A moved source vector has null data. It is not source heap {-1} empty.
	if empty {
		t.Fatal("move manufactured sentinel")
	}
	satOK(t, moved.Close())
	if _, e = s.runtime.memory.Read(original, 0); e == nil {
		t.Fatal("moved owner did not release backing")
	}
	satFullSolverRuntimeAudit(t, s)
}

// This is the guarded source signed-accounting overflow domain, not ordinary
// exhaustion. Only real public allocations/reallocations/frees prepare it.
func satHeapTLSNearMinimum(t *testing.T, r *memory.Runtime) {
	t.Helper()
	first, e := r.Allocate(8)
	satOK(t, e)
	second, e := r.Allocate(8)
	satOK(t, e)
	delta := int64(-1000000)
	if e = r.Reallocate(&first, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	satOK(t, r.Deallocate(&first))
	state := r.State()
	if state.LocalSize != 0 || state.GlobalSize >= 0 {
		t.Fatal(state)
	}
	target := int64(math.MinInt64) + 1000 - state.GlobalSize + 16
	if e = r.Reallocate(&second, uint64(target)+16-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	satOK(t, r.Deallocate(&second))
	if state = r.State(); state.GlobalSize != math.MinInt64+1000 || state.LocalSize != 0 {
		t.Fatal(state)
	}
}

func TestSATCaseSplitHeapSourceDestructorPostPhysicalTLSCursor(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true})
	satOK(t, e)
	a, e := newSATSolverRuntimeActivity(b, r)
	satOK(t, e)
	q, e := newSATSolverRuntimeVarQueue(a, b)
	satOK(t, e)
	satOK(t, q.reserveRuntime(50000))
	values := q.runtimeValues.raw.block
	indices := q.runtimeIndices.raw.block
	satHeapTLSNearMinimum(t, r)
	e = q.Close()
	if !errors.Is(e, arena.ErrBounds) || q.runtimeSourceCursor != 0 || q.runtimeIndices.raw.block != (memory.Block{}) || q.runtimeValues.raw.block != values || q.closed {
		t.Fatal("actual indices destruction prefix", e, q.runtimeSourceCursor, q.closed)
	}
	if _, e = r.Read(indices, 0); e == nil {
		t.Fatal("first source member must be physically consumed")
	}
	if _, e = r.Read(values, 0); e != nil {
		t.Fatal("older values member prematurely destroyed", e)
	}
	// Ordinary source retry retains the exact stopped boundary. Explicit owning
	// acknowledgment alone proceeds to the older values member and cold refund.
	cause := q.runtimeSourceStop
	if e = q.Close(); e != cause {
		t.Fatal("ordinary close repaired source cause", e, cause)
	}
	if r.Stats().LiveBlocks != 1 {
		t.Fatal("named older values only", r.Stats())
	}
	e = q.CleanupStopped()
	if !errors.Is(e, arena.ErrBounds) || q.runtimeSourceCursor != 1 || q.runtimeValues.raw.block != (memory.Block{}) || q.closed || r.Stats().LiveBlocks != 0 {
		t.Fatal("older member postphysical stopped prefix", e, q.runtimeSourceCursor, r.Stats())
	}
	satOK(t, q.CleanupStopped())
	if q.runtimeSourceCursor != 2 || !q.closed || q.runtimeSourceStop != cause || r.Stats().LiveBlocks != 0 {
		t.Fatal("owning acknowledgment", q.runtimeSourceCursor, q.closed, r.Stats())
	}
	satOK(t, a.Close())
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("actual graph not refunded", b.Used())
	}
}

func satHeapRuntimeAudit(t *testing.T, q *satVarQueue, a *satSolverRuntimeActivity, r *memory.Runtime, b *arena.Budget, extra ...*satVarQueue) {
	t.Helper()
	roots := []memory.Block{}
	var header uint64
	for _, q := range append([]*satVarQueue{q}, extra...) {
		if q == nil {
			continue
		}
		header += q.ownerBytes
		for _, c := range []*satSolverAssignmentColumn{&q.runtimeValues.raw, &q.runtimeIndices.raw} {
			if !c.pendingReplaces && c.block != (memory.Block{}) {
				roots = append(roots, c.block)
			}
			if c.pending != (memory.Block{}) {
				roots = append(roots, c.pending)
			}
		}
	}
	// An empty synthetic *reader* binds the real Runtime solely for the census;
	// no invented source owner or allocation enters the expected set.
	reader := satSolverAssignmentColumn{runtime: r}
	satAssignmentAuditBlocks(t, &reader, roots...)
	if want := r.Stats().TotalBytes + a.ownerBytes + header; want != b.Used() {
		t.Fatal("once-only actual heap cohort", want, b.Used())
	}
}

func TestSATCaseSplitHeapConstructorEveryActualRefusal(t *testing.T) {
	failures, success := 0, 0
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			b := arena.NewBudget(^uint64(0))
			r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
			satOK(t, e)
			a, e := newSATSolverRuntimeActivity(b, r)
			satOK(t, e)
			b.FailAfter(ordinal)
			q, e := newSATSolverRuntimeVarQueue(a, b)
			if e != nil {
				failures++
				if !satAssignmentActualRefusal(r, e) {
					t.Fatal(e)
				}
			} else {
				success++
			}
			satHeapRuntimeAudit(t, q, a, r, b) // BEFORE disabling refusal or owning cleanup.
			b.FailAfter(-1)
			if q != nil {
				satOK(t, q.Close())
			}
			satOK(t, a.Close())
			satOK(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
		})
		if success != 0 {
			break
		}
	}
	if failures == 0 || success != 1 {
		t.Fatal(failures, success)
	}
	t.Log("actual constructor refusals", failures, "first-success", success)
}

func TestSATCaseSplitHeapLargeReserveTriggersSameRuntimeCheckpoint(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	before, e := s.runtime.memory.AllocationSize()
	satOK(t, e)
	satOK(t, s.variables.queue.reserveRuntime(50000))
	after, e := s.runtime.memory.AllocationSize()
	satOK(t, e)
	if after <= before || s.variables.queue.runtimeIndices.raw.block == (memory.Block{}) {
		t.Fatal("real source storage not accounted", before, after)
	}
	s.config.state.MaxMemory = 0
	p := s.propagation
	// The sole source checkpoint modulo10 gate reads the exact same Runtime.
	for i := 0; i < 9; i++ {
		satOK(t, p.checkpoint())
	}
	if e = p.checkpoint(); !errors.Is(e, ErrSATSolverMaxMemory) {
		t.Fatal(e)
	}
	satFullSolverRuntimeAudit(t, s)
}

func TestSATCaseSplitHeapReserveAllCapacitiesBeforeSizePublication(t *testing.T) {
	failures, success := 0, 0
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			b := arena.NewBudget(^uint64(0))
			r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
			satOK(t, e)
			a, e := newSATSolverRuntimeActivity(b, r)
			satOK(t, e)
			q, e := newSATSolverRuntimeVarQueue(a, b)
			satOK(t, e)
			b.FailAfter(ordinal)
			e = q.reserveRuntime(1000)
			n, readErr := q.runtimeIndices.size()
			satOK(t, readErr)
			capacity, readErr := q.runtimeIndices.capacity()
			satOK(t, readErr)
			if e != nil {
				failures++
				if !satAssignmentActualRefusal(r, e) || n != 128 || capacity < 140 {
					t.Fatal("all capacity prefix, old logicalsize", e, n, capacity)
				}
			} else {
				success++
				if n != 1000 || capacity < 1000 {
					t.Fatal(n, capacity)
				}
			}
			for i := uint32(0); i < n; i++ {
				x, e := q.runtimeIndices.read(i)
				satOK(t, e)
				if x != 0 {
					t.Fatal(i, x)
				}
			}
			satHeapRuntimeAudit(t, q, a, r, b)
			b.FailAfter(-1)
			satOK(t, q.Close())
			satOK(t, a.Close())
			satOK(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
		})
		if success != 0 {
			break
		}
	}
	if failures == 0 || success != 1 {
		t.Fatal(failures, success)
	}
	t.Log("actual resize refusals", failures)
}

func TestSATCaseSplitHeapWarmPhysicalPlateauAndOwnerGuards(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 4; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	q := s.variables.queue
	copyOwner := *q
	if _, e := copyOwner.emptyChecked(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	if e := copyOwner.Close(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	// Warm resets reuse actual existing member capacities. No source allocator
	// event or Go physical allocation may be fabricated on this hot route.
	run := func() {
		satOK(t, q.Reset())
		for i := 0; i < 4; i++ {
			satOK(t, q.UnassignVar(satBoolVar(i)))
		}
		for i := 0; i < 4; i++ {
			_, e := q.NextVar()
			satOK(t, e)
		}
	}
	run()
	used := s.budget.Used()
	before := s.runtime.memory.Stats()
	s.budget.FailAfter(0)
	if allocations := testing.AllocsPerRun(20, run); allocations != 0 {
		t.Fatal("warmed heap host allocation", allocations)
	}
	satFullSolverRuntimeAudit(t, s)
	if after := s.runtime.memory.Stats(); after != before || s.budget.Used() != used {
		t.Fatal("warmed actual graph growth", before, after, used, s.budget.Used())
	}
	s.budget.FailAfter(-1)
}

func TestSATCaseSplitHeapReachedRawHeaderGuards(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	q := s.variables.queue
	original := q.runtimeValues.raw.block
	fresh, e := s.runtime.memory.Allocate(16)
	satOK(t, e)
	q.runtimeValues.raw.block = fresh
	// Source heap.empty reads the current values size; a reached uninitialized
	// source header must not be replaced by a shadow logical-size answer.
	if _, e = q.emptyChecked(); !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
	satScalarAuditRoots(t, s.variables, append(satFullSolverRuntimeRoots(s), original)...)
	q.runtimeValues.raw.block = original
	satOK(t, s.runtime.memory.Deallocate(&fresh))
	indices := q.runtimeIndices.raw.block
	satOK(t, satAssignmentWriteWord(s.runtime.memory, indices, 0, 0))
	e = q.reserveRuntime(129)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else {
		var source DefaultException
		if !errors.As(e, &source) || source.Error() != "Overflow encountered when expanding vector" {
			t.Fatal(e)
		}
	}
	if n, e := q.runtimeIndices.size(); e != nil || n != 128 {
		t.Fatal("guard retains source size", n, e)
	}
	satFullSolverRuntimeAudit(t, s)
	// Defined owning destruction uses malloc's extent, not the corrupted vector
	// capacity. No header repair is needed to release this diagnostic fixture.
}

func TestSATCaseSplitHeapCopyEveryActualRefusal(t *testing.T) {
	failures, success := 0, 0
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			b := arena.NewBudget(^uint64(0))
			r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
			satOK(t, e)
			a, e := newSATSolverRuntimeActivity(b, r)
			satOK(t, e)
			q, e := newSATSolverRuntimeVarQueue(a, b)
			satOK(t, e)
			sourceValues, sourceIndices := q.runtimeValues.raw.block, q.runtimeIndices.raw.block
			b.FailAfter(ordinal)
			copyOwner, e := q.Clone(b)
			if e != nil {
				failures++
				if !satAssignmentActualRefusal(r, e) {
					t.Fatal(e)
				}
			} else {
				success++
			}
			if q.runtimeValues.raw.block != sourceValues || q.runtimeIndices.raw.block != sourceIndices {
				t.Fatal("copy mutated source identity")
			}
			satHeapRuntimeAudit(t, q, a, r, b, copyOwner)
			b.FailAfter(-1)
			if copyOwner != nil {
				satOK(t, copyOwner.Close())
			}
			satOK(t, q.Close())
			satOK(t, a.Close())
			satOK(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
		})
		if success != 0 {
			break
		}
	}
	if failures == 0 || success != 1 {
		t.Fatal(failures, success)
	}
	t.Log("actual copy refusals", failures)
}

// A reached uninitialized spare coefficient is a guarded undefined-domain
// copy fixture; it is not a valid-world C++ allocation-exhaustion claim.
func TestSATCaseSplitHeapFailedCopyCompletedMemberBeforeOwningAbort(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	satOK(t, e)
	a, e := newSATSolverRuntimeActivity(b, r)
	satOK(t, e)
	q, e := newSATSolverRuntimeVarQueue(a, b)
	satOK(t, e)
	satOK(t, satAssignmentWriteWord(r, q.runtimeIndices.raw.block, 4, 129))
	copied, e := q.Clone(b)
	if !errors.Is(e, memory.ErrRuntimeUninitialized) || copied != nil {
		t.Fatal("reached unfinished second copy", copied, e)
	}
	satHeapRuntimeAudit(t, q, a, r, b)
	records := reflect.ValueOf(r).Elem().FieldByName("blocks")
	var releases []uint64
	for i := 0; i < records.Len(); i++ {
		record := records.Index(i)
		if record.FieldByName("live").Bool() {
			continue
		}
		serial := record.FieldByName("release").FieldByName("serial").Uint()
		if serial != 0 {
			releases = append(releases, serial)
		}
	}
	// Actual allocations are in construction order values then indices. Only
	// completed values has a source destructor; unfinished indices is freed
	// afterwards by the separately labelled Go owning-abort adapter.
	if len(releases) != 2 || releases[0] != 1 || releases[1] != 2 {
		t.Fatal("completed member before failed-ctor owning abort", releases)
	}
	satOK(t, q.Close())
	satOK(t, a.Close())
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestSATCaseSplitHeapNullPopReachedSourceDereference(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	satOK(t, e)
	a, e := newSATSolverRuntimeActivity(b, r)
	satOK(t, e)
	q, e := newSATSolverRuntimeVarQueue(a, b)
	satOK(t, e)
	moved, e := q.MoveClone(b)
	satOK(t, e)
	e = q.popRuntime()
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, memory.ErrRuntimeOwner) {
		t.Fatal("source null size-word decrement must reach guarded dereference", e)
	}
	satHeapRuntimeAudit(t, q, a, r, b, moved)
	satOK(t, q.Close())
	satOK(t, moved.Close())
	satOK(t, a.Close())
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestSATCaseSplitHeapDirectEraseSourcePreconditionAndWrapperGuard(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	q := s.variables.queue
	satOK(t, q.DelVar(0)) // var_queue guard: absent remains unchanged.
	if n, e := q.runtimeValues.size(); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	e := q.eraseRuntime(0) // actual concrete heap::erase, not var_queue guard.
	n, readErr := q.runtimeValues.size()
	satOK(t, readErr)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || n != 1 {
			t.Fatal("source SASSERT before absent index access", e, n)
		}
	} else {
		satOK(t, e)
		// idx0 equals size1-1, so source release clears index0 and pops sentinel.
		if n != 0 {
			t.Fatal("literal finite release absent-erase prefix", n)
		}
		empty, e := q.emptyChecked()
		satOK(t, e)
		if empty {
			t.Fatal("source empty compares size==1, not size0")
		}
	}
	if x, e := q.runtimeIndices.read(0); e != nil || x != 0 {
		t.Fatal(x, e)
	}
	satFullSolverRuntimeAudit(t, s)
}

func TestSATCaseSplitHeapForeignBudgetRejectedBeforeColdReserve(t *testing.T) {
	for _, op := range []string{"construct", "copy", "move"} {
		t.Run(op, func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			q := s.variables.queue
			foreign := arena.NewBudget(^uint64(0))
			foreign.FailAfter(0)
			before := s.runtime.memory.Stats()
			var created *satVarQueue
			var e error
			switch op {
			case "construct":
				created, e = newSATSolverRuntimeVarQueue(s.variables.activity, foreign)
			case "copy":
				created, e = q.Clone(foreign)
			case "move":
				created, e = q.MoveClone(foreign)
			}
			if created != nil {
				t.Cleanup(func() { satOK(t, created.Close()) })
			}
			if !errors.Is(e, ErrSATOwner) || created != nil || foreign.Used() != 0 || s.runtime.memory.Stats() != before {
				t.Fatal("actual same-cohort guard before reserve", op, e, created, foreign.Used(), before, s.runtime.memory.Stats())
			}
			foreign.FailAfter(-1)
			satFullSolverRuntimeAudit(t, s)
		})
	}
}

func TestSATCaseSplitHeapLegacySignaturesExposeActualTypedReadFailure(t *testing.T) {
	for _, op := range []string{"Size", "Empty", "Contains", "closed"} {
		t.Run(op, func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			q := s.variables.queue
			if op == "closed" {
				satOK(t, q.Close())
				recovered := func() (r any) { defer func() { r = recover() }(); q.Empty(); return nil }()
				e, ok := recovered.(error)
				if !ok || !errors.Is(e, arena.ErrClosed) {
					t.Fatal("closed Runtime facade silent default", recovered)
				}
				return
			}
			c := &q.runtimeValues.raw
			if op == "Contains" {
				c = &q.runtimeIndices.raw
			}
			original := c.block
			fresh, e := s.runtime.memory.Allocate(16)
			satOK(t, e)
			c.block = fresh
			t.Cleanup(func() { c.block = original; satOK(t, s.runtime.memory.Deallocate(&fresh)) })
			recovered := func() (r any) {
				defer func() { r = recover() }()
				switch op {
				case "Size":
					q.Size()
				case "Empty":
					q.Empty()
				case "Contains":
					q.Contains(0)
				}
				return nil
			}()
			e, ok := recovered.(error)
			if !ok || !errors.Is(e, memory.ErrRuntimeUninitialized) {
				t.Fatal("legacy signature silently discarded actual reached read guard", op, recovered)
			}
			satScalarAuditRoots(t, s.variables, append(satFullSolverRuntimeRoots(s), original)...)
		})
	}
}

func TestSATCaseSplitHeapBorrowedRuntimeBlockedUntilAllCopiedBackingsReleased(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	satOK(t, e)
	a, e := newSATSolverRuntimeActivity(b, r)
	satOK(t, e)
	q, e := newSATSolverRuntimeVarQueue(a, b)
	satOK(t, e)
	clone, e := q.Clone(b)
	satOK(t, e)
	if e = r.Close(); !errors.Is(e, memory.ErrRuntimeLive) {
		t.Fatal("caller Runtime cannot free actual heap raw ownership", e)
	}
	satHeapRuntimeAudit(t, q, a, r, b, clone)
	satOK(t, q.Close())
	if e = r.Close(); !errors.Is(e, memory.ErrRuntimeLive) {
		t.Fatal("copy retains independent actual backing", e)
	}
	satHeapRuntimeAudit(t, q, a, r, b, clone)
	satOK(t, clone.Close())
	satOK(t, a.Close())
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
