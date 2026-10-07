package sat

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func satUseFixture(t *testing.T, n int) (*ClauseAllocator, *satClauseUseList, []StoredClauseRef, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	a := satStore(t, b)
	u, e := newSATClauseUseList(a, b)
	satOK(t, e)
	t.Cleanup(func() {
		satOK(t, u.Close())
		satOK(t, a.Close())
		if b.Used() != 0 {
			t.Fatal("use-list ownership", b.Used())
		}
	})
	refs := make([]StoredClauseRef, n)
	for i := range refs {
		refs[i] = satMk(t, a, satLs(uint32(i*2), uint32(i*2+3), uint32(i*2+4)), false)
		satOK(t, u.Insert(refs[i]))
	}
	return a, u, refs, b
}
func TestSATClauseUseListInsertEraseNotRemovedReset(t *testing.T) {
	a, u, r, b := satUseFixture(t, 3)
	if u.Size() != 3 || u.Empty() {
		t.Fatal(u.Size())
	}
	satOK(t, u.EraseNotRemoved(r[1]))
	if u.Size() != 2 || u.vector.Size() != 2 {
		t.Fatal(u.Size(), u.vector.Size())
	}
	x, e := u.vector.Get(1)
	satOK(t, e)
	if x != r[2] {
		t.Fatal("physical erase order", x)
	}
	if ok, e := u.CheckInvariant(); !ok || e != nil {
		t.Fatal(ok, e)
	}
	before := b.Used()
	backing := u.vector.Stats().BackingBytes
	satOK(t, u.Reset())
	if !u.Empty() || u.vector.Size() != 0 || u.vector.Capacity() != 0 || b.Used() != before-backing {
		t.Fatal("source finalize before size0", u.Size(), b.Used(), before, backing)
	}
	for _, x := range r {
		if !a.IsLive(x) {
			t.Fatal("list does not own clauses", x)
		}
	}
}
func TestSATClauseUseListLazyEraseAndUnsignedUnderflow(t *testing.T) {
	a, u, r, _ := satUseFixture(t, 1)
	satOK(t, a.SetRemoved(r[0], true))
	satOK(t, u.Erase(r[0]))
	if !u.Empty() || u.vector.Size() != 1 {
		t.Fatal("lazy erase must retain pointer")
	}
	satOK(t, u.Erase(r[0]))
	if u.Size() != ^uint32(0) {
		t.Fatal("source unsigned repeated erase", u.Size())
	}
	ok, e := u.CheckInvariant()
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(ok, e)
		}
	} else if !ok || e != nil {
		t.Fatal(ok, e)
	}
	satOK(t, u.Reset())
}
func TestSATClauseUseListBeginConsumeAndDestructorCompletion(t *testing.T) {
	a, u, r, _ := satUseFixture(t, 5)
	for _, i := range []int{0, 2, 4} {
		satOK(t, a.SetRemoved(r[i], true))
		satOK(t, u.Erase(r[i]))
	}
	it, e := u.Begin()
	satOK(t, e)
	if it.i != 1 || it.j != 0 || it.size != 5 || u.vector.Size() != 5 {
		t.Fatal("source begin consume timing", it)
	}
	x, e := u.vector.Get(0)
	satOK(t, e)
	if x != r[1] {
		t.Fatal("begin already publishes compacted current")
	}
	if x, e := it.Current(); e != nil || x != r[1] {
		t.Fatal(x, e)
	}
	satOK(t, it.Close())
	satOK(t, it.Close())
	if u.vector.Size() != 2 || u.Size() != 2 {
		t.Fatal(u.Size(), u.vector.Size())
	}
	for i, want := range []StoredClauseRef{r[1], r[3]} {
		x, e := u.vector.Get(uint32(i))
		satOK(t, e)
		if x != want {
			t.Fatal(i, x, want)
		}
	}
	if ok, e := u.CheckInvariant(); !ok || e != nil {
		t.Fatal(ok, e)
	}
	for _, x := range r {
		if !a.IsLive(x) {
			t.Fatal("iterator destroyed clause", x)
		}
	}
}
func TestSATClauseUseListPartialIterationFinishesAndAllRemoved(t *testing.T) {
	a, u, r, _ := satUseFixture(t, 4)
	satOK(t, a.SetRemoved(r[1], true))
	satOK(t, u.Erase(r[1]))
	it, e := u.Begin()
	satOK(t, e)
	satOK(t, it.Next())
	if it.i != 2 || it.j != 1 {
		t.Fatal(it)
	}
	satOK(t, it.Close())
	if u.vector.Size() != 3 {
		t.Fatal(u.vector.Size())
	}
	for _, x := range []StoredClauseRef{r[0], r[2], r[3]} {
		satOK(t, a.SetRemoved(x, true))
		satOK(t, u.Erase(x))
	}
	it, e = u.Begin()
	satOK(t, e)
	if !it.AtEnd() || it.i != 3 || it.j != 0 || u.vector.Size() != 3 {
		t.Fatal(it)
	}
	satOK(t, it.Close())
	if u.vector.Size() != 0 || !u.Empty() {
		t.Fatal(u.Size())
	}
}
func TestSATClauseUseListIteratorCurrentBackingAfterGrowth(t *testing.T) {
	_, u, r, _ := satUseFixture(t, 3)
	it, e := u.Begin()
	satOK(t, e)
	satOK(t, u.vector.Set(0, r[2]))
	x, e := it.Current()
	satOK(t, e)
	if x != r[2] {
		t.Fatal("current cached original buffer/value", x)
	}
	old := u.vector.Capacity()
	satOK(t, u.Insert(r[0]))
	if satStorageSourceDebug { // r[0] no longer occurs after the direct Set above.
		found, e := u.vector.Contains(r[0])
		satOK(t, e)
		if !found {
			t.Fatal("fixture")
		}
	}
	if u.vector.Capacity() <= old {
		t.Fatal("fixture did not actually grow")
	}
	if x, e = it.Current(); e != nil || x != r[2] {
		t.Fatal("growth must follow vector reference", x, e)
	}
	satOK(t, it.Close())
	if u.vector.Size() != 3 || u.Size() != 4 {
		t.Fatal("captured-end destructor drops appended tail, does not repair size", u.Size(), u.vector.Size())
	}
}
func TestSATClauseUseListIteratorFollowsVectorSwap(t *testing.T) {
	a, u, r, b := satUseFixture(t, 3)
	it, e := u.Begin()
	satOK(t, e)
	other, e := NewSATClauseVector(a, b)
	satOK(t, e)
	defer func() { satOK(t, other.Close()) }()
	for _, x := range []StoredClauseRef{r[2], r[1], r[0]} {
		satOK(t, other.Push(x))
	}
	satOK(t, u.vector.Swap(other))
	x, e := it.Current()
	satOK(t, e)
	if x != r[2] {
		t.Fatal("iterator pinned backing")
	}
	satOK(t, it.Next())
	satOK(t, it.Close())
	x, e = u.vector.Get(2)
	satOK(t, e)
	if x != r[0] {
		t.Fatal(x)
	}
}
func TestSATClauseUseListIteratorCurrentLogicalDebugRetainedRelease(t *testing.T) {
	_, u, r, _ := satUseFixture(t, 3)
	it, e := u.Begin()
	satOK(t, e)
	satOK(t, u.vector.Shrink(1))
	e = it.Next()
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || it.i != 1 || it.j != 1 {
			t.Fatal("actual operator[] guard after index increments", it, e)
		}
	} else {
		satOK(t, e)
		x, e := it.Current()
		satOK(t, e)
		if x != r[1] {
			t.Fatal(x)
		}
	}
	satOK(t, u.vector.SetEnd(3))
	satOK(t, it.Close())
	if u.vector.Size() != 3 {
		t.Fatal("source destructor uses captured count", u.vector.Size())
	}
}
func TestSATClauseUseListAtEndScalarAfterVectorClose(t *testing.T) {
	_, u, _, _ := satUseFixture(t, 0)
	it, e := u.Begin()
	satOK(t, e)
	satOK(t, u.Close())
	if !it.AtEnd() {
		t.Fatal("at_end must not dereference dead vector")
	}
	if e = it.Close(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("destructor shrink still accesses vector", e)
	}
}
func TestSATClauseUseListSourceDebugPreconditionsAndReleaseBodies(t *testing.T) {
	a, u, r, _ := satUseFixture(t, 2)
	e := u.Insert(r[0])
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || u.Size() != 2 {
			t.Fatal(e, u.Size())
		}
	} else {
		satOK(t, e)
		if u.Size() != 3 {
			t.Fatal(u.Size())
		}
	}
	satOK(t, a.SetRemoved(r[1], true))
	e = u.Insert(r[1])
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else {
		satOK(t, e)
	}
	missing := satMk(t, a, satLs(12, 14), false)
	before := u.Size()
	e = u.EraseNotRemoved(missing)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || u.Size() != before {
			t.Fatal(e)
		}
	} else {
		satOK(t, e)
		if u.Size() != before-1 {
			t.Fatal("release missing erase decrements", u.Size())
		}
	}
	satOK(t, u.Reset())
}
func TestSATClauseUseListNextRemovedAssertionAndCurrentCopy(t *testing.T) {
	a, u, r, _ := satUseFixture(t, 1)
	it, e := u.Begin()
	satOK(t, e)
	satOK(t, a.SetRemoved(r[0], true))
	satOK(t, u.Erase(r[0]))
	x, e := it.Current()
	satOK(t, e)
	if x != r[0] {
		t.Fatal("curr must not invent was_removed guard")
	}
	e = it.Next()
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || it.i != 0 || it.j != 0 {
			t.Fatal(e, it)
		}
		satOK(t, a.SetRemoved(r[0], false))
		satOK(t, it.Close())
	} else {
		satOK(t, e)
		satOK(t, it.Close())
	}
}
func TestSATClauseUseListPayloadFailurePrefixAndReleaseInvariantTraversal(t *testing.T) {
	a, u, r, _ := satUseFixture(t, 3)
	satOK(t, a.SetRemoved(r[0], true))
	satOK(t, u.Erase(r[0]))
	satOK(t, a.Delete(r[1]))
	it, e := u.Begin()
	if e == nil || it.i != 1 || it.j != 0 || u.vector.Size() != 3 {
		t.Fatal("consume skips prefix then stops at actual dead payload", it, e)
	}
	if ok, e := u.CheckInvariant(); e == nil {
		t.Fatal("release check_invariant still executes was_removed reads", ok, e)
	}
	// curr only returns the retained scalar reference; a later actual clause
	// access performs the lifetime guard. This is not a was_removed callback.
	x, e := it.Current()
	satOK(t, e)
	if x != r[1] {
		t.Fatal(x)
	}
	if _, e = a.Removed(x); e == nil {
		t.Fatal("fixture dead scalar payload")
	}
}
func TestSATClauseUseListConstructCloneAssignFailureAccounting(t *testing.T) {
	a, u, r, b := satUseFixture(t, 3)
	before := b.Used()
	for _, fail := range []int64{0, 1} {
		b.FailAfter(fail)
		n, e := newSATClauseUseList(a, b)
		b.FailAfter(-1)
		if !errors.Is(e, arena.ErrInjectedFailure) || n != nil || b.Used() != before {
			t.Fatal(fail, n, e, b.Used(), before)
		}
	}
	n, e := u.Clone(b)
	satOK(t, e)
	defer func() { satOK(t, n.Close()) }()
	if n.Size() != u.Size() || n.vector.Capacity() != u.vector.Capacity() {
		t.Fatal("implicit field copy")
	}
	satOK(t, n.vector.Set(0, r[2]))
	x, e := u.vector.Get(0)
	satOK(t, e)
	if x != r[0] {
		t.Fatal("copy aliases backing")
	}
	satOK(t, n.Assign(n))
	oldCount := n.Size()
	b.FailAfter(0)
	e = n.Assign(u)
	b.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || n.vector.Size() != 0 || n.Size() != oldCount {
		t.Fatal("member assignment finalizes vector before allocation, count assigned last", e, n.Size(), n.vector.Size())
	}
	satOK(t, n.Assign(u))
	if n.Size() != u.Size() {
		t.Fatal(n.Size())
	}
	for _, x := range r {
		if !a.IsLive(x) {
			t.Fatal("copy lost clause roots")
		}
	}
}
func TestSATClauseUseListInsertGrowthFailureAndWarmIteratorNoAllocation(t *testing.T) {
	a, u, r, b := satUseFixture(t, 2)
	x := satMk(t, a, satLs(20, 22), false)
	oldSize := u.Size()
	oldCap := u.vector.Capacity()
	before := b.Used()
	b.FailAfter(0)
	e := u.Insert(x)
	b.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || u.Size() != oldSize || u.vector.Size() != oldSize || u.vector.Capacity() != oldCap || b.Used() != before {
		t.Fatal("push failure before size increment", e, u.Size(), u.vector.Size())
	}
	if allocs := testing.AllocsPerRun(50, func() {
		it, e := u.Begin()
		if e != nil {
			panic(e)
		}
		if e = it.Close(); e != nil {
			panic(e)
		}
	}); allocs != 0 {
		t.Fatal("by-value iterator metadata allocation", allocs)
	}
	if b.Used() != before {
		t.Fatal("iterator hidden budget charge")
	}
	satOK(t, u.Close())
	if !a.IsLive(r[0]) || !a.IsLive(r[1]) || !a.IsLive(x) {
		t.Fatal("Close does not own clauses")
	}
}
func TestSATClauseUseListIteratorPhysicalBoundAndClosedOwner(t *testing.T) {
	a, u, _, b := satUseFixture(t, 2)
	it, e := u.Begin()
	satOK(t, e)
	other, e := NewSATClauseVector(a, b)
	satOK(t, e)
	defer func() { satOK(t, other.Close()) }()
	satOK(t, u.vector.Swap(other))
	if _, e = it.Current(); satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	clonedHeader := *u
	if _, e = clonedHeader.Begin(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satOK(t, u.Close())
	if e = u.Insert(StoredClauseRef{}); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestSATClauseUseListActualColdAndGrowthPeakCharges(t *testing.T) {
	a := satStore(t, nil)
	header := uint64(reflect.TypeOf(satClauseUseList{}).Size()) + uint64(reflect.TypeOf(satClauseVector{}).Size())
	stride := uint64(reflect.TypeOf(StoredClauseRef{}).Size())
	// Source capacity2 then3: both backing arrays physically coexist during the
	// next reserve. Refuse one byte below that real peak, rather than source-width
	// phantom accounting or charging only the final net backing difference.
	b := arena.NewBudget(header + 5*stride - 1)
	u, e := newSATClauseUseList(a, b)
	satOK(t, e)
	defer func() {
		satOK(t, u.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	if b.Used() != header {
		t.Fatal("actual cold hierarchy", b.Used(), header)
	}
	for i := uint32(0); i < 2; i++ {
		r := satMk(t, a, satLs(i*2, i*2+3), false)
		satOK(t, u.Insert(r))
	}
	if b.Used() != header+2*stride {
		t.Fatal("full physical vector backing charge", b.Used())
	}
	r := satMk(t, a, satLs(20, 22), false)
	e = u.Insert(r)
	if !errors.Is(e, arena.ErrBudget) || u.Size() != 2 || u.vector.Size() != 2 || b.Used() != header+2*stride {
		t.Fatal("growth peak refusal publication", e, b.Used(), u.Size())
	}
	if !a.IsLive(r) {
		t.Fatal("failed list insertion released caller clause")
	}
}

func TestSATClauseUseListImplicitAssignmentCurrentVectorAndRealBindings(t *testing.T) {
	a, u, original, b := satUseFixture(t, 3)
	foreign, other, r, _ := satUseFixture(t, 3)
	satOK(t, foreign.SetRemoved(r[1], true))
	satOK(t, other.Erase(r[1]))
	it, e := u.Begin()
	satOK(t, e)
	satOK(t, u.Assign(other))
	current, e := it.Current()
	satOK(t, e)
	if current != r[0] {
		t.Fatal("implicit assignment changed vector member rather than its backing", current)
	}
	satOK(t, it.Next())
	if it.i != 2 || it.j != 1 {
		t.Fatal("was_removed must use actual copied allocator binding", it)
	}
	satOK(t, it.Close())
	if u.Size() != 2 || u.vector.Size() != 2 {
		t.Fatal(u.Size(), u.vector.Size())
	}
	if ok, e := u.CheckInvariant(); !ok || e != nil {
		t.Fatal(ok, e)
	}
	copy, e := u.Clone(b)
	satOK(t, e)
	defer func() { satOK(t, copy.Close()) }()
	copied, e := copy.Begin()
	satOK(t, e)
	satOK(t, copied.Close())
	for _, x := range original {
		if !a.IsLive(x) {
			t.Fatal("assignment destroyed prior borrowed clauses", x)
		}
	}
	for _, x := range r {
		if !foreign.IsLive(x) {
			t.Fatal("copy/iterator destroyed foreign borrowed clauses", x)
		}
	}
}

func TestSATClauseUseListFiniteIndependentStableCompaction(t *testing.T) {
	for mask := uint32(0); mask < 64; mask++ {
		func() {
			a, u, r, _ := satUseFixture(t, 6)
			var want []StoredClauseRef
			for i, x := range r {
				if mask&(1<<uint(i)) != 0 {
					satOK(t, a.SetRemoved(x, true))
					satOK(t, u.Erase(x))
				} else {
					want = append(want, x)
				}
			}
			it, e := u.Begin()
			satOK(t, e)
			// Vary lexical early destruction independently from the expected complete
			// stable subsequence; destructor must finish the remaining source actions.
			reads := int(mask % 7)
			if reads > len(want) {
				reads = len(want)
			}
			for i := 0; i < reads; i++ {
				if it.AtEnd() {
					t.Fatal(mask, i)
				}
				x, e := it.Current()
				satOK(t, e)
				if x != want[i] {
					t.Fatal(mask, i, x, want[i])
				}
				satOK(t, it.Next())
			}
			satOK(t, it.Close())
			if u.Size() != uint32(len(want)) || u.vector.Size() != uint32(len(want)) {
				t.Fatal(mask, u.Size(), u.vector.Size(), len(want))
			}
			for i, x := range want {
				got, e := u.vector.Get(uint32(i))
				satOK(t, e)
				if got != x {
					t.Fatal(mask, i, got, x)
				}
			}
			if ok, e := u.CheckInvariant(); !ok || e != nil {
				t.Fatal(mask, ok, e)
			}
			satOK(t, u.Close())
			satOK(t, a.Close())
		}()
	}
}

func TestSATClauseUseListEverySourceAssertionSite(t *testing.T) {
	a, u, r, _ := satUseFixture(t, 1)
	// Second Insert assertion, with a genuinely absent removed clause.
	absent := satMk(t, a, satLs(30, 32), false)
	satOK(t, a.SetRemoved(absent, true))
	e := u.Insert(absent)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || u.Size() != 1 {
			t.Fatal(e, u.Size())
		}
	} else {
		satOK(t, e)
		satOK(t, u.Erase(absent))
	}
	// Both callbacks use a contained scalar, reaching their second assertions.
	satOK(t, a.SetRemoved(r[0], true))
	before := u.Size()
	e = u.EraseNotRemoved(r[0])
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || u.Size() != before {
			t.Fatal(e)
		}
		satOK(t, a.SetRemoved(r[0], false))
		e = u.Erase(r[0])
		if !errors.Is(e, ErrSATAssertion) || u.Size() != before {
			t.Fatal(e)
		}
	} else {
		satOK(t, e)
	}
	// A missing scalar reaches Erase's contains assertion before any payload read.
	missing := satMk(t, a, satLs(40, 42), false)
	e = u.Erase(missing)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else {
		satOK(t, e)
	}
	satOK(t, u.Reset())
	it, e := u.Begin()
	satOK(t, e)
	_, e = it.Current()
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	e = it.Next()
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || it.i != 0 || it.j != 0 {
			t.Fatal(e, it)
		}
	} else if !errors.Is(e, arena.ErrBounds) || it.i != 1 || it.j != 1 {
		t.Fatal("release source increments before actual bounds guard", e, it)
	}
}

func TestSATClauseUseListSourceReadThenWriteAndResetRefusalPrefixes(t *testing.T) {
	_, u, _, _ := satUseFixture(t, 2)
	it, e := u.Begin()
	satOK(t, e)
	oldEpoch := u.vector.epoch
	u.vector.epoch = ^uint64(0)
	e = it.Next()
	if !errors.Is(e, arena.ErrSlotLimit) || it.i != 1 || it.j != 1 || u.vector.Size() != 2 {
		t.Fatal("increment/read prefix before write refusal", e, it)
	}
	oldCount := u.Size()
	e = u.Reset()
	if !errors.Is(e, arena.ErrSlotLimit) || u.Size() != oldCount || u.vector.Size() != 2 {
		t.Fatal("reset publishes count0 only after finalize", e)
	}
	u.vector.epoch = oldEpoch
	satOK(t, it.Close())
}

func TestSATClauseUseListCloneReservationSweep(t *testing.T) {
	a, u, r, b := satUseFixture(t, 3)
	before := b.Used()
	complete := false
	for fail := int64(0); fail < 16 && !complete; fail++ {
		b.FailAfter(fail)
		n, e := u.Clone(b)
		b.FailAfter(-1)
		if e == nil {
			complete = true
			satOK(t, n.Close())
		} else if n != nil || !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(fail, n, e)
		}
		if b.Used() != before || u.Size() != 3 || u.vector.Size() != 3 {
			t.Fatal("partial constructor storage leak", fail, b.Used(), before)
		}
		for _, x := range r {
			if !a.IsLive(x) {
				t.Fatal("lost actual caller clause roots", fail, x)
			}
		}
	}
	if !complete {
		t.Fatal("sweep never reached actual success")
	}
}
