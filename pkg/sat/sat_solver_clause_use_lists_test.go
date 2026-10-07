// Licensed under the MIT license.
package sat

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func satOuterUseFixture(t *testing.T, vars uint32) (*ClauseAllocator, *satClauseUseLists, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	a := satStore(t, b)
	u, e := newSATClauseUseLists(a, b, arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{Budget: b}})
	satOK(t, e)
	t.Cleanup(func() {
		satOK(t, u.Close())
		satOK(t, a.Close())
		if b.Used() != 0 {
			t.Fatal("outer ownership", b.Used())
		}
	})
	satOK(t, u.Init(vars))
	return a, u, b
}
func TestSATClauseUseListsLiteralRoutingAndEraseExcept(t *testing.T) {
	a, u, _ := satOuterUseFixture(t, 4)
	c := satMk(t, a, satLs(0, 3, 6), false)
	satOK(t, u.Insert(c))
	for i := uint32(0); i < 8; i++ {
		v, e := u.Get(Literal(i))
		satOK(t, e)
		n, e := v.Size()
		satOK(t, e)
		want := uint32(0)
		if i == 0 || i == 3 || i == 6 {
			want = 1
		}
		if n != want {
			t.Fatal(i, n, want)
		}
	}
	satOK(t, a.SetRemoved(c, true))
	satOK(t, u.EraseExcept(c, Literal(3)))
	for _, i := range []uint32{0, 6} {
		v, e := u.Get(Literal(i))
		satOK(t, e)
		n, e := v.Size()
		satOK(t, e)
		if n != 0 {
			t.Fatal(i, n)
		}
	}
	v, e := u.Get(Literal(3))
	satOK(t, e)
	n, e := v.Size()
	satOK(t, e)
	if n != 1 {
		t.Fatal(n)
	}
}
func TestSATClauseUseListsSameAddressReconstruction(t *testing.T) {
	a, u, _ := satOuterUseFixture(t, 2)
	c := satMk(t, a, satLs(0, 1, 2), false)
	satOK(t, u.Insert(c))
	v, e := u.Get(0)
	satOK(t, e)
	it, e := v.Begin()
	satOK(t, e)
	before := u.backingID
	satOK(t, u.Init(2))
	if u.backingID != before {
		t.Fatal("unexpected outer growth")
	}
	d := satMk(t, a, satLs(0, 1, 3), false)
	satOK(t, u.Insert(d))
	n, e := v.Size()
	satOK(t, e)
	if n != 1 {
		t.Fatal("same-address replacement", n)
	}
	r, e := it.Current()
	satOK(t, e)
	if r != d {
		t.Fatal("iterator should follow reconstructed inline member", r, d)
	}
	satOK(t, it.Close())
}
func TestSATClauseUseListsOuterGrowthInvalidatesChild(t *testing.T) {
	a, u, _ := satOuterUseFixture(t, 1)
	c := satMk(t, a, satLs(0, 1, 0), false)
	v, e := u.Get(0)
	satOK(t, e)
	satOK(t, v.Insert(c))
	it, e := v.Begin()
	satOK(t, e)
	satOK(t, u.Init(32))
	if _, e := v.Size(); !errors.Is(e, ErrSATOwner) {
		t.Fatal("dangling old outer address", e)
	}
	if _, e := it.Current(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	if it.AtEnd() {
		t.Fatal("AtEnd must remain captured scalar")
	}
}
func TestSATClauseUseListsLaterLiteralFailurePreservesPrefix(t *testing.T) {
	a, u, _ := satOuterUseFixture(t, 2)
	c := satMk(t, a, satLs(0, 1, 99), false)
	e := u.Insert(c)
	want := arena.ErrBounds
	if satStorageSourceDebug {
		want = ErrSATAssertion
	}
	if !errors.Is(e, want) {
		t.Fatal(e)
	}
	for _, i := range []Literal{0, 1} {
		v, e := u.Get(i)
		satOK(t, e)
		n, e := v.Size()
		satOK(t, e)
		if n != 1 {
			t.Fatal(i, n)
		}
	}
}
func TestSATClauseUseListsInitUnsignedWrapAndFinalize(t *testing.T) {
	a, u, b := satOuterUseFixture(t, 2)
	c := satMk(t, a, satLs(0, 1, 2), false)
	satOK(t, u.Insert(c))
	old := u.capacity
	satOK(t, u.Init(1<<31))
	if u.length != 0 || u.capacity != old {
		t.Fatal(u.length, u.capacity, old)
	}
	if !a.IsLive(c) {
		t.Fatal("list owns clause")
	}
	retained := b.Used()
	satOK(t, u.Finalize())
	if u.capacity != 0 || u.length != 0 || b.Used() >= retained {
		t.Fatal("outer finalize")
	}
	satOK(t, u.Init(1))
}

func TestSATClauseUseListsIteratorCurrentInnerBackingAndDestructor(t *testing.T) {
	a, u, _ := satOuterUseFixture(t, 4)
	v, e := u.Get(0)
	satOK(t, e)
	refs := make([]StoredClauseRef, 5)
	for i := range refs {
		refs[i] = satMk(t, a, satLs(0, uint32(i+1), 7), false)
		satOK(t, v.Insert(refs[i]))
	}
	it, e := v.Begin()
	satOK(t, e)
	r, e := v.record()
	satOK(t, e)
	old := r.capacity
	for i := 0; i < 8; i++ {
		c := satMk(t, a, satLs(0, 1, 2), false)
		satOK(t, v.Insert(c))
	}
	if r.capacity <= old {
		t.Fatal("no inner growth")
	}
	satOK(t, v.write(0, refs[4]))
	c, e := it.Current()
	satOK(t, e)
	if c != refs[4] {
		t.Fatal("cached inner payload")
	}
	satOK(t, it.Close())
	if r.length != 5 || r.liveSize != 13 {
		t.Fatal("captured-end compaction publication", r.length, r.liveSize)
	}
}
func TestSATClauseUseListsRemovedMasksIndependent(t *testing.T) {
	for mask := 0; mask < 64; mask++ {
		a, u, _ := satOuterUseFixture(t, 3)
		v, e := u.Get(0)
		satOK(t, e)
		var refs [6]StoredClauseRef
		var want []StoredClauseRef
		for i := range refs {
			refs[i] = satMk(t, a, satLs(0, 1, 2), false)
			satOK(t, v.Insert(refs[i]))
			if mask&(1<<i) != 0 {
				satOK(t, a.SetRemoved(refs[i], true))
				satOK(t, v.Erase(refs[i]))
			} else {
				want = append(want, refs[i])
			}
		}
		it, e := v.Begin()
		satOK(t, e)
		var got []StoredClauseRef
		for !it.AtEnd() {
			c, e := it.Current()
			satOK(t, e)
			got = append(got, c)
			satOK(t, it.Next())
		}
		satOK(t, it.Close())
		if len(got) != len(want) {
			t.Fatal(mask, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatal(mask, i)
			}
		}
		r, e := v.record()
		satOK(t, e)
		if r.length != uint32(len(want)) || r.liveSize != uint32(len(want)) {
			t.Fatal(mask, r)
		}
		if ok, e := v.CheckInvariant(); !ok || e != nil {
			t.Fatal(mask, ok, e)
		}
	}
}
func TestSATClauseUseListsDuplicateAndRepeatedErase(t *testing.T) {
	a, u, _ := satOuterUseFixture(t, 2)
	c := satMk(t, a, satLs(0, 0, 1), false)
	e := u.Insert(c)
	v, x := u.Get(0)
	satOK(t, x)
	n, x := v.Size()
	satOK(t, x)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || n != 1 {
			t.Fatal(e, n)
		}
		return
	}
	satOK(t, e)
	if n != 2 {
		t.Fatal(n)
	}
	satOK(t, a.SetRemoved(c, true))
	satOK(t, u.Erase(c))
	satOK(t, u.Erase(c))
	n, e = v.Size()
	satOK(t, e)
	if n != ^uint32(0)-1 {
		t.Fatal("unsigned double duplicated erase", n)
	}
}
func TestSATClauseUseListsConstructorFailureGraph(t *testing.T) {
	done := false
	for fail := int64(0); fail < 64; fail++ {
		b := arena.NewBudget(^uint64(0))
		a := satStore(t, b)
		before := b.Used()
		b.FailAfter(fail)
		u, e := newSATClauseUseLists(a, b, arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{Budget: b}})
		b.FailAfter(-1)
		if e == nil {
			satOK(t, u.Close())
			done = true
		} else if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(fail, e)
		}
		if b.Used() != before {
			t.Fatal("constructor ownership", fail, b.Used(), before)
		}
		satOK(t, a.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
		if done {
			break
		}
	}
	if !done {
		t.Fatal("constructor never completed")
	}
}
func TestSATClauseUseListsInitRefusalCompletedGrowth(t *testing.T) {
	a, u, b := satOuterUseFixture(t, 2)
	c := satMk(t, a, satLs(0, 1, 2), false)
	satOK(t, u.Insert(c))
	v, e := u.Get(0)
	satOK(t, e)
	old := u.capacity
	b.FailAfter(2)
	e = u.Init(100)
	b.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	c1, _, e := satOuterUseCapacity(old, uint64(^uint(0)>>1))
	satOK(t, e)
	c2, _, e := satOuterUseCapacity(c1, uint64(^uint(0)>>1))
	satOK(t, e)
	if u.capacity != c2 || u.length != 0 || u.words.Stats().Live != 0 {
		t.Fatal("completed reset/growth prefix", u.capacity, c2, u.length, u.words.Stats())
	}
	if _, e = v.Size(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	if !a.IsLive(c) {
		t.Fatal("clause freed")
	}
	satOK(t, u.Init(1))
}
func TestSATClauseUseListsInsertionFaultRootsAndPrefixes(t *testing.T) {
	done := false
	for fail := int64(0); fail < 80; fail++ {
		b := arena.NewBudget(^uint64(0))
		a := satStore(t, b)
		c := satMk(t, a, satLs(0, 1, 2), false)
		before := b.Used()
		u, e := newSATClauseUseLists(a, b, arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{Budget: b}})
		satOK(t, e)
		satOK(t, u.Init(2))
		b.FailAfter(fail)
		e = u.Insert(c)
		b.FailAfter(-1)
		if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(fail, e)
		}
		var count uint32
		ended := false
		for i := uint32(0); i < 3; i++ {
			v, x := u.Get(Literal(i))
			satOK(t, x)
			n, x := v.Size()
			satOK(t, x)
			if n == 0 {
				ended = true
			} else if n != 1 || ended {
				t.Fatal("nonprefix", fail, i, n)
			}
			count += n
		}
		if u.words.Stats().Live != count {
			t.Fatal("immediate lost descriptor", fail, count, u.words.Stats())
		}
		if !a.IsLive(c) {
			t.Fatal("lost clause")
		}
		satOK(t, u.Close())
		if b.Used() != before {
			t.Fatal("close graph", fail, b.Used(), before)
		}
		satOK(t, a.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
		if e == nil {
			done = true
			break
		}
	}
	if !done {
		t.Fatal("fault sweep never completed")
	}
}
func TestSATClauseUseListsPhysicalRecordAndHostBounds(t *testing.T) {
	if e := arena.ValidateRecordType(reflect.TypeOf(satClauseUseListRecord{})); e != nil {
		t.Fatal(e)
	}
	if n := reflect.TypeOf(satClauseUseListRecord{}).Size(); n != 32 {
		t.Fatal(n)
	}
	if _, _, e := satOuterUseCapacity((1<<28)-1, ^uint64(0)); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("source unsigned extent", e)
	}
	if _, _, e := satOuterUseCapacity(1<<26, (1<<31)-1); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual simulated32 host extent", e)
	}
}
func TestSATClauseUseListsWarmZeroAllocationAndNoClauseOwnership(t *testing.T) {
	a, u, b := satOuterUseFixture(t, 2)
	c := satMk(t, a, satLs(0, 1, 2), false)
	run := func() {
		if e := u.Insert(c); e != nil {
			panic(e)
		}
		v, e := u.Get(0)
		if e != nil {
			panic(e)
		}
		it, e := v.Begin()
		if e != nil {
			panic(e)
		}
		if e = it.Close(); e != nil {
			panic(e)
		}
		if e = u.Init(2); e != nil {
			panic(e)
		}
	}
	run()
	before := b.Used()
	if n := testing.AllocsPerRun(100, run); n != 0 {
		t.Fatal("warm allocation", n)
	}
	if b.Used() != before || u.words.Stats().Live != 0 || !a.IsLive(c) {
		t.Fatal("warm ownership", b.Used(), before)
	}
}

func TestSATClauseUseListsEndedAndNeverConstructedSlots(t *testing.T) {
	_, u, _ := satOuterUseFixture(t, 2)
	v, e := u.Get(3)
	satOK(t, e)
	cap := u.capacity
	satOK(t, u.Init(1))
	if u.capacity != cap {
		t.Fatal("growth")
	}
	if _, e = v.Size(); !errors.Is(e, ErrSATOwner) {
		t.Fatal("ended nontrivial object", e)
	}
	if _, e = u.Get(3); satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satOK(t, u.Init(2))
	if n, e := v.Size(); e != nil || n != 0 {
		t.Fatal("same-address transparent replacement", n, e)
	}
	v, e = u.Get(0)
	satOK(t, e)
	if _, e = v.read(0); satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("never initialized primitive slot", e)
	}
}
func TestSATClauseUseListsRawCurrentAndPayloadFailurePrefix(t *testing.T) {
	a, u, _ := satOuterUseFixture(t, 2)
	v, e := u.Get(0)
	satOK(t, e)
	c := satMk(t, a, satLs(0, 1, 2), false)
	d := satMk(t, a, satLs(0, 1, 2), false)
	satOK(t, v.Insert(c))
	satOK(t, v.Insert(d))
	it, e := v.Begin()
	satOK(t, e)
	satOK(t, a.Delete(d))
	e = it.Next()
	if !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	if it.i != 1 || it.j != 1 {
		t.Fatal("completed Next cursor prefix", it)
	}
	if got, e := it.Current(); e != nil || got != d {
		t.Fatal("Current invents payload callback", got, e)
	}
	// Close cannot safely complete invalid-source callback; reset destroys only
	// primitive references, and does not dereference the freed clause.
	satOK(t, v.Reset())
	if n, e := v.Size(); e != nil || n != 0 {
		t.Fatal(n, e)
	}
}
func TestSATClauseUseListsEraseNotRemovedAndReleaseMissing(t *testing.T) {
	a, u, _ := satOuterUseFixture(t, 2)
	v, e := u.Get(0)
	satOK(t, e)
	var refs [3]StoredClauseRef
	for i := range refs {
		refs[i] = satMk(t, a, satLs(0, 1, 2), false)
		satOK(t, v.Insert(refs[i]))
	}
	satOK(t, v.EraseNotRemoved(refs[1]))
	r, e := v.record()
	satOK(t, e)
	if r.length != 2 || r.liveSize != 2 {
		t.Fatal(r)
	}
	c, e := v.read(1)
	satOK(t, e)
	if c != refs[2] {
		t.Fatal(c)
	}
	e = v.EraseNotRemoved(refs[1])
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || r.liveSize != 2 {
			t.Fatal(e, r)
		}
	} else {
		satOK(t, e)
		if r.length != 2 || r.liveSize != 1 {
			t.Fatal(r)
		}
	}
}

func satOuterUseOwnedBytes(u *satClauseUseLists) uint64 {
	s := u.words.Stats()
	d := s.Descriptors
	return u.ownerBytes + u.backingBytes + s.OwnerBytes + s.PayloadBytes + s.BlockMetadataBytes + s.DirectoryBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
}
func TestSATClauseUseListsExactOwnedGraphAndColdGuards(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	a := satStore(t, b)
	c := satMk(t, a, satLs(0, 1, 2), false)
	base := b.Used()
	u, e := newSATClauseUseLists(a, b, arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{Budget: b}})
	satOK(t, e)
	check := func() {
		if n := satOuterUseOwnedBytes(u); b.Used()-base != n {
			t.Fatal("actual complete graph", b.Used()-base, n)
		}
	}
	check()
	satOK(t, u.Init(3))
	check()
	satOK(t, u.Insert(c))
	check()
	copyOwner := *u
	if _, e = copyOwner.Get(0); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	v, e := u.Get(0)
	satOK(t, e)
	satOK(t, v.Reset())
	check()
	satOK(t, u.Finalize())
	check()
	satOK(t, u.Close())
	satOK(t, u.Close())
	if b.Used() != base {
		t.Fatal(b.Used(), base)
	}
	if _, e = v.Size(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	satOK(t, a.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestSATClauseUseListsDestructorBeyondCapturedEnd(t *testing.T) {
	a, u, _ := satOuterUseFixture(t, 2)
	v, e := u.Get(0)
	satOK(t, e)
	var refs [3]StoredClauseRef
	for i := range refs {
		refs[i] = satMk(t, a, satLs(0, 1, 2), false)
		satOK(t, v.Insert(refs[i]))
	}
	satOK(t, v.EraseNotRemoved(refs[0]))
	satOK(t, v.EraseNotRemoved(refs[1]))
	it, e := v.Begin()
	satOK(t, e)
	satOK(t, it.Next())
	e = it.Next()
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || it.i != 1 || it.j != 1 {
			t.Fatal(e, it)
		}
	} else {
		satOK(t, e)
		if it.i != 2 || it.j != 2 {
			t.Fatal(it)
		}
	}
	// Source destructor uses i<size, not i!=size. Release i=2,size=1 skips
	// completion and publishes initialized retained j=2 via vector::shrink.
	satOK(t, it.Close())
	r, e := v.record()
	satOK(t, e)
	want := uint32(2)
	if satStorageSourceDebug {
		want = 1
	}
	if r.length != want {
		t.Fatal(r.length, want)
	}
}
