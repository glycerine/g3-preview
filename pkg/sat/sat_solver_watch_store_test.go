package sat

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func satWatchStoreFixture(t *testing.T) (*satSolverWatchStore, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	s, e := newSATSolverWatchStore(arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	paramsMust(t, e)
	t.Cleanup(func() {
		paramsMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatalf("retained watch graph %d", b.Used())
		}
	})
	return s, b
}
func satWatchStoreValue(i uint32) satWatched {
	return satWatched{val1: i, val2: 2, locator: satClauseOffset{owner: 0xfedcba9876543210, slot: i, generation: 123, storageKind: 2}}
}
func TestSATSolverWatchStoreCodecAndCapacity(t *testing.T) {
	s, _ := satWatchStoreFixture(t)
	r, e := s.NewList()
	paramsMust(t, e)
	for i, c := range []uint32{2, 2, 3, 5, 5} {
		paramsMust(t, s.Push(r, satWatchStoreValue(uint32(i))))
		got, e := s.Capacity(r)
		paramsMust(t, e)
		if got != c {
			t.Fatalf("capacity %d/%d", got, c)
		}
		v, e := s.Read(r, uint32(i))
		paramsMust(t, e)
		if v != satWatchStoreValue(uint32(i)) {
			t.Fatal("codec loses locator bits")
		}
	}
	for _, x := range []reflect.Type{reflect.TypeOf(satSolverWatchListRef{}), reflect.TypeOf(satSolverWatchListRecord{})} {
		paramsMust(t, arena.ValidateRecordType(x))
	}
}
func TestSATSolverWatchStoreErasePopHolesAndGrowth(t *testing.T) {
	s, _ := satWatchStoreFixture(t)
	r, e := s.NewList()
	paramsMust(t, e)
	for i := uint32(0); i < 5; i++ {
		paramsMust(t, s.Push(r, satWatchStoreValue(i)))
	}
	paramsMust(t, s.EraseAt(r, 0))
	paramsMust(t, s.EraseAt(r, 0))
	_, e = s.Pop(r)
	paramsMust(t, e)
	// Erase leaves slot4 alive; Pop destroys slot2, so a high-water lifetime is wrong.
	v, e := s.readPhysical(r, 4)
	paramsMust(t, e)
	if v != satWatchStoreValue(4) {
		t.Fatal("erase tail lost")
	}
	if _, e = s.readPhysical(r, 2); !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("pop hole %v", e)
	}
	paramsMust(t, s.ReserveCapacity(r, 9))
	v, e = s.readPhysical(r, 4)
	paramsMust(t, e)
	if v != satWatchStoreValue(4) {
		t.Fatal("full capacity copy lost live tail")
	}
	if _, e = s.readPhysical(r, 2); !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("growth filled dead hole %v", e)
	}
	paramsMust(t, s.ResetList(r))
	_, e = s.readPhysical(r, 4)
	paramsMust(t, e)
	paramsMust(t, s.Finalize(r))
	if !s.IsLive(r) {
		t.Fatal("finalize destroyed list object")
	}
}
func TestSATSolverWatchStoreCopyLifetimeAndLedger(t *testing.T) {
	s, _ := satWatchStoreFixture(t)
	r, e := s.NewList()
	paramsMust(t, e)
	for i := uint32(0); i < 5; i++ {
		paramsMust(t, s.Push(r, satWatchStoreValue(i)))
	}
	paramsMust(t, s.EraseAt(r, 0))
	q, e := s.CloneList(r)
	paramsMust(t, e)
	c, e := s.Capacity(q)
	paramsMust(t, e)
	if c != 5 {
		t.Fatal(c)
	}
	if _, e = s.readPhysical(q, 4); !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("copy constructed tail %v", e)
	}
	paramsMust(t, s.Write(q, 0, satWatchStoreValue(99)))
	v, e := s.Read(r, 0)
	paramsMust(t, e)
	if v.val1 != 1 {
		t.Fatal("copy aliases")
	}
	paramsMust(t, s.Free(r))
	n, e := s.NewList()
	paramsMust(t, e)
	if n.slot != r.slot || n.generation == r.generation || s.IsLive(r) {
		t.Fatal("stale generation")
	}
}
func TestSATSolverWatchStoreEraseClauseDestroysTail(t *testing.T) {
	s, _ := satWatchStoreFixture(t)
	r, e := s.NewList()
	paramsMust(t, e)
	for i := uint32(0); i < 3; i++ {
		paramsMust(t, s.Push(r, satWatchStoreValue(i)))
	}
	found, e := s.EraseClauseWatch(r, satWatchStoreValue(1).locator)
	paramsMust(t, e)
	if !found {
		t.Fatal("not erased")
	}
	if _, e = s.readPhysical(r, 2); !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("set_end left tail live %v", e)
	}
}
func TestSATSolverWatchStoreFailureAndWarmValues(t *testing.T) {
	s, b := satWatchStoreFixture(t)
	r, e := s.NewList()
	paramsMust(t, e)
	paramsMust(t, s.Push(r, satWatchStoreValue(1)))
	b.FailAfter(0)
	e = s.ReserveCapacity(r, 1000)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatalf("armed growth %v", e)
	}
	n, e := s.Size(r)
	paramsMust(t, e)
	if n != 1 {
		t.Fatal("failed growth changed size")
	}
	v, e := s.Read(r, 0)
	paramsMust(t, e)
	if v != satWatchStoreValue(1) {
		t.Fatal("failed growth changed value")
	}
	paramsMust(t, s.ResetList(r))
	if a := testing.AllocsPerRun(100, func() {
		if e := s.Push(r, satWatchStoreValue(2)); e != nil {
			panic(e)
		}
		if _, e := s.Pop(r); e != nil {
			panic(e)
		}
	}); a != 0 {
		t.Fatalf("warm allocation %g", a)
	}
}

var satWatchStoreEscapedIterator satSolverWatchIterator

func TestSATSolverWatchStoreIteratorAndMove(t *testing.T) {
	s, b := satWatchStoreFixture(t)
	r, e := s.NewList()
	paramsMust(t, e)
	paramsMust(t, s.Push(r, satWatchStoreValue(9)))
	paramsMust(t, s.Push(r, satWatchStoreValue(10)))
	it, e := s.Iter(r)
	paramsMust(t, e)
	paramsMust(t, s.ResetList(r))
	if _, _, e = it.Next(); !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("destroyed iterator slot %v", e)
	}
	paramsMust(t, s.Push(r, satWatchStoreValue(4)))
	it, e = s.Iter(r)
	paramsMust(t, e)
	q, e := s.MoveList(r)
	paramsMust(t, e)
	n, e := s.Size(r)
	paramsMust(t, e)
	if n != 0 {
		t.Fatal("move source not empty")
	}
	v, e := s.Read(q, 0)
	paramsMust(t, e)
	if v.val1 != 4 {
		t.Fatal("move payload")
	}
	v, ok, e := it.Next()
	paramsMust(t, e)
	if !ok || v.val1 != 4 {
		t.Fatalf("source iterator must follow moved backing: %v %v", v, ok)
	}
	b.FailAfter(0)
	if a := testing.AllocsPerRun(100, func() {
		var e error
		satWatchStoreEscapedIterator, e = s.Iter(q)
		if e != nil {
			panic(e)
		}
		if _, ok, e := satWatchStoreEscapedIterator.Next(); e != nil || !ok {
			panic("warm forward read")
		}
		satWatchStoreEscapedIterator, e = s.ReverseIter(q)
		if e != nil {
			panic(e)
		}
		if _, ok, e := satWatchStoreEscapedIterator.Next(); e != nil || !ok {
			panic("warm reverse read")
		}
	}); a != 0 {
		t.Fatalf("escaping value iterator %g", a)
	}
}
func TestSATSolverWatchStoreCrossCopyAndAssignment(t *testing.T) {
	a, _ := satWatchStoreFixture(t)
	b, _ := satWatchStoreFixture(t)
	r, e := a.NewList()
	paramsMust(t, e)
	paramsMust(t, a.ResizeFill(r, 5, satWatchStoreValue(17)))
	q, e := b.CopyList(a, r)
	paramsMust(t, e)
	paramsMust(t, b.Write(q, 0, satWatchStoreValue(18)))
	v, e := a.Read(r, 0)
	paramsMust(t, e)
	if v.val1 != 17 {
		t.Fatal("foreign copy aliases")
	}
	paramsMust(t, b.AssignList(q, a, r))
	v, e = b.Read(q, 0)
	paramsMust(t, e)
	if v.val1 != 17 {
		t.Fatal("assign failed")
	}
	paramsMust(t, b.AssignList(q, b, q))
}
func TestSATSolverWatchStoreSourceProfileBoundaries(t *testing.T) {
	s, _ := satWatchStoreFixture(t)
	r, e := s.NewList()
	paramsMust(t, e)
	paramsMust(t, s.Push(r, satWatchStoreValue(1)))
	paramsMust(t, s.EraseAt(r, 0))
	_, e = s.Read(r, 0)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatalf("source assertion %v", e)
		}
	} else {
		paramsMust(t, e)
	}
	if e = s.Shrink(r, 1); satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if e = s.SetEnd(r, 1); !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("unsafe destructor loop %v", e)
	}
	paramsMust(t, s.ResizeFill(r, 1, satWatchStoreValue(2)))
	if e = s.SetX(r, ^uint32(0), satWatchStoreValue(3), satWatchStoreValue(4)); !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("setx wrap %v", e)
	}
	n, e := s.Size(r)
	paramsMust(t, e)
	if n != 0 {
		t.Fatal("source wrapped resize prefix missing")
	}
}

func TestSATSolverWatchStoreSourceHelpersAndReverse(t *testing.T) {
	s, _ := satWatchStoreFixture(t)
	a, e := s.NewList()
	paramsMust(t, e)
	b, e := s.NewList()
	paramsMust(t, e)
	paramsMust(t, s.AppendArray(a, []satWatched{satWatchStoreValue(1), satWatchStoreValue(2), satWatchStoreValue(3)}))
	paramsMust(t, s.Reverse(a))
	it, e := s.ReverseIter(a)
	paramsMust(t, e)
	for j := uint32(1); j <= 3; j++ {
		w, ok, e := it.Next()
		paramsMust(t, e)
		if !ok || w.val1 != j {
			t.Fatal(w, ok)
		}
	}
	paramsMust(t, s.Append(b, s, a))
	paramsMust(t, s.Fill(a, satWatchStoreValue(8)))
	ok, e := s.Contains(b, satWatchStoreValue(2))
	paramsMust(t, e)
	if !ok {
		t.Fatal("contains")
	}
	paramsMust(t, s.SwapLists(a, b))
	w, e := s.Read(a, 0)
	paramsMust(t, e)
	if w.val1 != 3 {
		t.Fatal("swap")
	}
	if e = s.Append(a, s, a); !errors.Is(e, ErrSATSelfAppend) {
		t.Fatal(e)
	}
	paramsMust(t, s.MoveAssignList(b, a))
	n, e := s.Size(a)
	paramsMust(t, e)
	if n != 0 {
		t.Fatal("move assign")
	}
}
func TestSATSolverWatchStoreFailureSweepAndPhysicalAccounting(t *testing.T) {
	sawPrefix := false
	for fail := int64(0); fail < 24; fail++ {
		b := arena.NewBudget(^uint64(0))
		s, e := newSATSolverWatchStore(arena.VectorConfig{PageSize: 32, Descriptors: arena.Config{PageSize: 2, Budget: b}})
		paramsMust(t, e)
		r, e := s.NewList()
		paramsMust(t, e)
		for i := uint32(0); i < 5; i++ {
			paramsMust(t, s.Push(r, satWatchStoreValue(i)))
		}
		paramsMust(t, s.EraseAt(r, 0))
		paramsMust(t, s.EraseAt(r, 0))
		b.FailAfter(fail)
		e = s.ReserveCapacity(r, 9)
		c, ce := s.Capacity(r)
		paramsMust(t, ce)
		if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(e)
		}
		if e != nil && c == 8 {
			sawPrefix = true
		}
		v, ve := s.readPhysical(r, 4)
		paramsMust(t, ve)
		if v.val1 != 4 {
			t.Fatal("failure lost tail")
		}
		stats := s.Stats()
		if stats.ChargedBytes != b.Used() {
			t.Fatalf("immediate owner graph %d/%d", stats.ChargedBytes, b.Used())
		}
		if stats.Lists != 1 {
			t.Fatal("lost root")
		}
		paramsMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatal("failure leaked")
		}
	}
	if !sawPrefix {
		t.Fatal("did not exercise completed first expansion then next failure")
	}
}
func TestSATSolverWatchStoreConstructorUnwind(t *testing.T) {
	sawSuccess := false
	for fail := int64(0); fail < 15; fail++ {
		b := arena.NewBudget(^uint64(0))
		b.FailAfter(fail)
		s, e := newSATSolverWatchStore(arena.VectorConfig{PageSize: 32, Descriptors: arena.Config{PageSize: 2, Budget: b}})
		if e == nil {
			sawSuccess = true
			paramsMust(t, s.Close())
		} else if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(e)
		}
		if b.Used() != 0 {
			t.Fatalf("constructor prefix %d leaks %d", fail, b.Used())
		}
	}
	if !sawSuccess {
		t.Fatal("no successful constructor")
	}
}

func TestSATSolverWatchStoreIndependentLifetimeModel(t *testing.T) {
	s, _ := satWatchStoreFixture(t)
	r, e := s.NewList()
	paramsMust(t, e)
	var data [64]satWatched
	var life [64]bool
	var size, capacity uint32
	seed := uint32(17)
	grow := func(n uint32) {
		for n > capacity {
			if capacity == 0 {
				capacity = 2
			} else {
				capacity = (3*capacity + 1) / 2
			}
		}
	}
	for step := uint32(0); step < 1500; step++ {
		seed = 1664525*seed + 1013904223
		switch seed % 6 {
		case 0:
			if size < 24 {
				w := satWatchStoreValue(step)
				paramsMust(t, s.Push(r, w))
				grow(size + 1)
				data[size] = w
				life[size] = true
				size++
			}
		case 1:
			if size > 0 {
				w, e := s.Pop(r)
				paramsMust(t, e)
				if w != data[size-1] {
					t.Fatal("oracle pop")
				}
				size--
				life[size] = false
			}
		case 2:
			if size > 0 {
				i := (seed >> 8) % size
				paramsMust(t, s.EraseAt(r, i))
				for j := i + 1; j < size; j++ {
					data[j-1] = data[j]
				}
				size--
			}
		case 3:
			n := (seed >> 8) % 25
			w := satWatchStoreValue(step)
			paramsMust(t, s.ResizeFill(r, n, w))
			if n > size {
				grow(n)
				for i := size; i < n; i++ {
					data[i] = w
					life[i] = true
				}
			} else {
				for i := n; i < size; i++ {
					life[i] = false
				}
			}
			size = n
		case 4:
			paramsMust(t, s.ResetList(r))
			for i := uint32(0); i < size; i++ {
				life[i] = false
			}
			size = 0
		case 5:
			n := (seed >> 8) % 25
			paramsMust(t, s.ReserveCapacity(r, n))
			grow(n)
		}
		actual, e := s.Size(r)
		paramsMust(t, e)
		c, e := s.Capacity(r)
		paramsMust(t, e)
		if actual != size || c != capacity {
			t.Fatalf("model step%d size/cap %d/%d expected%d/%d", step, actual, c, size, capacity)
		}
		for i := uint32(0); i < capacity; i++ {
			w, e := s.readPhysical(r, i)
			if life[i] {
				paramsMust(t, e)
				if w != data[i] {
					t.Fatalf("model slot%d at step%d", i, step)
				}
			} else if !errors.Is(e, arena.ErrBounds) {
				t.Fatalf("model dead slot%d at step%d: %v", i, step, e)
			}
		}
	}
}
func TestSATSolverWatchStoreOwnerLedgerRetirement(t *testing.T) {
	s, b := satWatchStoreFixture(t)
	other, _ := satWatchStoreFixture(t)
	r, e := s.NewList()
	paramsMust(t, e)
	if _, e = other.Size(r); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	paramsMust(t, s.Reset())
	if s.IsLive(r) {
		t.Fatal("reset reused generation")
	}
	q, e := s.NewList()
	paramsMust(t, e)
	x, e := s.record(q)
	paramsMust(t, e)
	x.generation = ^uint32(0)
	q.generation = x.generation
	paramsMust(t, s.Free(q))
	n, e := s.NewList()
	paramsMust(t, e)
	if n.slot == q.slot {
		t.Fatal("exhausted slot recycled")
	}
	stats := s.Stats()
	if stats.Retired != 1 || stats.ChargedBytes != b.Used() {
		t.Fatal(stats)
	}
	copied := *s
	if _, e = copied.NewList(); !errors.Is(e, ErrSATOwner) {
		t.Fatal("cold owner copy accepted")
	}
}

func TestSATSolverWatchStoreActualWatchDisplayAndStableSort(t *testing.T) {
	s, _ := satWatchStoreFixture(t)
	r, e := s.NewList()
	paramsMust(t, e)
	a, e := newSATBinaryWatch(Literal(2), true)
	paramsMust(t, e)
	b, e := newSATTernaryWatch(4, 7)
	paramsMust(t, e)
	c, e := newSATExternalWatch(3)
	paramsMust(t, e)
	paramsMust(t, s.AppendArray(r, []satWatched{c, b, a, c, a}))
	paramsMust(t, s.SortByKind(r))
	want := []satWatched{a, a, b, c, c}
	for i, w := range want {
		v, e := s.Read(r, uint32(i))
		paramsMust(t, e)
		if v != w {
			t.Fatal("sort kind/stability")
		}
	}
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{})
	paramsMust(t, e)
	defer out.Close()
	paramsMust(t, s.DisplayInto(r, nil, out))
	text, e := out.String()
	paramsMust(t, e)
	if text != "1* 1* (2 -3) 3 3" {
		t.Fatal(text)
	}
	ca := satStore(t, nil)
	cl := satMk(t, ca, satLs(0, 2), false)
	cw, e := newSATClauseWatch(4, satClauseOffset(cl))
	paramsMust(t, e)
	paramsMust(t, s.ResetList(r))
	paramsMust(t, s.Push(r, cw))
	out2, e := util.NewNumericTextStream(util.NumericTextStreamConfig{})
	paramsMust(t, e)
	defer out2.Close()
	paramsMust(t, s.DisplayInto(r, ca, out2))
	text, e = out2.String()
	paramsMust(t, e)
	if text != "(2 (0 1)+)" {
		t.Fatal(text)
	}
}

func TestSATSolverWatchStoreConstructorRoutesAndBounds(t *testing.T) {
	s, b := satWatchStoreFixture(t)
	r, e := s.NewListFill(4, satWatchStoreValue(7))
	paramsMust(t, e)
	n, e := s.Size(r)
	paramsMust(t, e)
	c, e := s.Capacity(r)
	paramsMust(t, e)
	if n != 4 || c != 5 {
		t.Fatalf("source fill ctor resize policy %d/%d", n, c)
	}
	q, e := s.NewListArray([]satWatched{satWatchStoreValue(1), satWatchStoreValue(2), satWatchStoreValue(3)})
	paramsMust(t, e)
	c, e = s.Capacity(q)
	paramsMust(t, e)
	if c != 3 {
		t.Fatal(c)
	}
	words, bits, e := satWatchPayloadExtent((^uint32(0) - 8) / 8)
	paramsMust(t, e)
	if words != 2147483640 || bits != 16777216 {
		t.Fatalf("source maximum growth extent %d/%d", words, bits)
	}
	if _, _, e = satWatchPayloadExtent(^uint32(0)); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	n64, e := satWatchDirectoryCapacity(70000000)
	if uint64(^uint(0)>>1)/uint64(reflect.TypeOf(satSolverWatchListRecord{}).Size()) >= 105000000 {
		paramsMust(t, e)
		if n64 != 105000000 {
			t.Fatal(n64)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	before := s.Stats().Lists
	b.FailAfter(0)
	_, e = s.NewListFill(^uint32(0), satWatchStoreValue(8))
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if s.Stats().Lists != before {
		t.Fatal("unpublished constructor root retained")
	}
	if s.Stats().ChargedBytes != b.Used() {
		t.Fatal("unpublished constructor graph uncharged")
	}
}

func TestSATSolverWatchStoreBitmapWordBoundaryAndRawClauseBorrow(t *testing.T) {
	s, _ := satWatchStoreFixture(t)
	r, e := s.NewList()
	paramsMust(t, e)
	for i := uint32(0); i < 40; i++ {
		paramsMust(t, s.Push(r, satWatchStoreValue(i)))
	}
	paramsMust(t, s.EraseAt(r, 0))
	paramsMust(t, s.EraseAt(r, 0))
	_, e = s.Pop(r)
	paramsMust(t, e)
	paramsMust(t, s.ReserveCapacity(r, 62))
	for _, i := range []uint32{31, 32, 36, 38, 39} {
		_, e = s.readPhysical(r, i)
		paramsMust(t, e)
	}
	if _, e = s.readPhysical(r, 37); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("cross-word destroyed hole", e)
	}
	paramsMust(t, s.Shrink(r, 20))
	for _, i := range []uint32{20, 31, 32, 36, 37} {
		if _, e = s.readPhysical(r, i); !errors.Is(e, arena.ErrBounds) {
			t.Fatal("cross-word shrink", i, e)
		}
	}
	for _, i := range []uint32{38, 39} {
		w, e := s.readPhysical(r, i)
		paramsMust(t, e)
		if w.val1 != 39 {
			t.Fatal(w)
		}
	}
	raw := satWatched{val1: ^uint32(0), val2: ^uint32(0), locator: satClauseOffset{owner: ^uint64(0), slot: ^uint32(0), generation: ^uint32(0), storageKind: ^uint32(0)}}
	if satWatchCodecDecode(satWatchCodecEncode(raw)) != raw {
		t.Fatal("codec high bits")
	}
	a := satStore(t, nil)
	cl := satMk(t, a, satLs(0, 2), false)
	w, e := newSATClauseWatch(0, satClauseOffset(cl))
	paramsMust(t, e)
	paramsMust(t, s.ResetList(r))
	paramsMust(t, s.Push(r, w))
	paramsMust(t, a.Close())
	q, e := s.CloneList(r)
	paramsMust(t, e)
	got, e := s.Read(q, 0)
	paramsMust(t, e)
	if got != w {
		t.Fatal("raw copy dereferenced dead clause spelling")
	}
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{})
	paramsMust(t, e)
	defer out.Close()
	if e = s.DisplayInto(q, a, out); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("actual clause borrow lifetime guard", e)
	}
}

func TestSATSolverWatchStoreDefaultBackAndTernaryRoutes(t *testing.T) {
	s, _ := satWatchStoreFixture(t)
	r, e := s.NewList()
	paramsMust(t, e)
	d := satWatchStoreValue(44)
	v, e := s.GetDefault(r, ^uint32(0), d)
	paramsMust(t, e)
	if v != d {
		t.Fatal(v)
	}
	if _, e = s.Back(r); satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	w, e := newSATTernaryWatch(2, 4)
	paramsMust(t, e)
	paramsMust(t, s.Push(r, w))
	paramsMust(t, s.Push(r, d))
	paramsMust(t, s.EraseTernaryWatch(r, 2, 4))
	v, e = s.Back(r)
	paramsMust(t, e)
	if v != d {
		t.Fatal(v)
	}
	v, e = s.readPhysical(r, 1)
	paramsMust(t, e)
	if v != d {
		t.Fatal("ternary erase incorrectly destroyed tail")
	}
}

func TestSATSolverWatchStoreAssignmentConstructionGuardPrefix(t *testing.T) {
	s, b := satWatchStoreFixture(t)
	src, e := s.NewListFill(2, satWatchStoreValue(77))
	paramsMust(t, e)
	dst, e := s.NewListFill(1, satWatchStoreValue(99))
	paramsMust(t, e)
	// Explicit unsafe-source lifetime fixture: an element is destroyed while
	// the vector header still includes it. Guard only at its copy access.
	x, e := s.record(src)
	paramsMust(t, e)
	paramsMust(t, s.setInitialized(x, 1, false))
	e = s.AssignList(dst, s, src)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	n, e := s.Size(dst)
	paramsMust(t, e)
	if n != 2 {
		t.Fatalf("copy_core must publish complete size before constructing prefix: %d", n)
	}
	w, e := s.Read(dst, 0)
	paramsMust(t, e)
	if w.val1 != 77 {
		t.Fatal("first constructed copy lost")
	}
	if _, e = s.Read(dst, 1); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("unconstructed suffix not guarded", e)
	}
	if s.Stats().ChargedBytes != b.Used() {
		t.Fatal("construction prefix uncharged")
	}
}

func TestSATSolverWatchStoreNullBackingShrinkAndSetEnd(t *testing.T) {
	s, _ := satWatchStoreFixture(t)
	r, e := s.NewList()
	paramsMust(t, e)
	for _, n := range []uint32{1, ^uint32(0)} {
		e = s.Shrink(r, n)
		if satStorageSourceDebug {
			if !errors.Is(e, ErrSATAssertion) {
				t.Fatal("null source shrink assertion", e)
			}
		} else {
			paramsMust(t, e)
		}
		e = s.SetEnd(r, n)
		if satStorageSourceDebug {
			if !errors.Is(e, ErrSATAssertion) {
				t.Fatal("null source iterator assertion", e)
			}
		} else {
			paramsMust(t, e)
		}
		size, e := s.Size(r)
		paramsMust(t, e)
		cap, e := s.Capacity(r)
		paramsMust(t, e)
		if size != 0 || cap != 0 {
			t.Fatal("null source body changed header")
		}
	}
	paramsMust(t, s.Push(r, satWatchStoreValue(1)))
	paramsMust(t, s.Finalize(r))
	e = s.Shrink(r, 7)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else {
		paramsMust(t, e)
	}
}

func TestSATSolverWatchStoreIteratorsFollowMoveAndSwapBacking(t *testing.T) {
	s, _ := satWatchStoreFixture(t)
	a, e := s.NewListFill(2, satWatchStoreValue(91))
	paramsMust(t, e)
	it, e := s.Iter(a)
	paramsMust(t, e)
	b, e := s.MoveList(a)
	paramsMust(t, e)
	paramsMust(t, s.Free(a))
	w, ok, e := it.Next()
	paramsMust(t, e)
	if !ok || w.val1 != 91 {
		t.Fatal("move constructor invalidated actual live backing")
	}
	_, e = s.Pop(b)
	paramsMust(t, e)
	if _, _, e = it.Next(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("transferred destination pop did not destroy captured slot", e)
	}
	c, e := s.NewListFill(2, satWatchStoreValue(11))
	paramsMust(t, e)
	d, e := s.NewListFill(1, satWatchStoreValue(22))
	paramsMust(t, e)
	ci, e := s.Iter(c)
	paramsMust(t, e)
	di, e := s.Iter(d)
	paramsMust(t, e)
	paramsMust(t, s.MoveAssignList(d, c))
	paramsMust(t, s.Free(c))
	w, ok, e = ci.Next()
	paramsMust(t, e)
	if !ok || w.val1 != 11 {
		t.Fatal("move assignment lost source iterator")
	}
	if _, _, e = di.Next(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("move assignment kept freed destination iterator", e)
	}
	p, e := s.NewListFill(2, satWatchStoreValue(3))
	paramsMust(t, e)
	q, e := s.NewListFill(2, satWatchStoreValue(4))
	paramsMust(t, e)
	pi, e := s.Iter(p)
	paramsMust(t, e)
	qi, e := s.Iter(q)
	paramsMust(t, e)
	paramsMust(t, s.SwapLists(p, q))
	_, e = s.Pop(p)
	paramsMust(t, e)
	w, ok, e = qi.Next()
	paramsMust(t, e)
	if !ok || w.val1 != 4 {
		t.Fatal("swap source data")
	}
	if _, _, e = qi.Next(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("swap destination destruction", e)
	}
	for j := 0; j < 2; j++ {
		w, ok, e = pi.Next()
		paramsMust(t, e)
		if !ok || w.val1 != 3 {
			t.Fatal("swap lost still-live original backing")
		}
	}
	gi, e := s.Iter(q)
	paramsMust(t, e)
	paramsMust(t, s.ReserveCapacity(q, 3))
	if _, _, e = gi.Next(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("actual reallocation did not invalidate backing", e)
	}
}

func TestSATSolverWatchStoreFiniteBudgetRefusal(t *testing.T) {
	empty := arena.NewBudget(0)
	if _, e := newSATSolverWatchStore(arena.VectorConfig{Descriptors: arena.Config{Budget: empty}}); !errors.Is(e, arena.ErrBudget) || empty.Used() != 0 {
		t.Fatal("zero-budget constructor", e, empty.Used())
	}
	b := arena.NewBudget(1 << 20)
	s, e := newSATSolverWatchStore(arena.VectorConfig{PageSize: 32, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	paramsMust(t, e)
	r, e := s.NewListFill(2, satWatchStoreValue(1))
	paramsMust(t, e)
	base := b.Used()
	hold := b.Limit() - base
	paramsMust(t, b.Reserve(hold))
	e = s.ReserveCapacity(r, 1000)
	if !errors.Is(e, arena.ErrBudget) {
		t.Fatal("real finite budget", e)
	}
	if s.Stats().ChargedBytes != base || b.Used() != base+hold {
		t.Fatal("refusal changed actual graph")
	}
	w, e := s.Read(r, 1)
	paramsMust(t, e)
	if w.val1 != 1 {
		t.Fatal("refusal lost prior root")
	}
	b.Release(hold)
	paramsMust(t, s.Close())
	if b.Used() != 0 {
		t.Fatal("finite budget cleanup")
	}
}
