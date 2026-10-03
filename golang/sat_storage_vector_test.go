package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"testing"
)

func TestSATConcreteVectorSourceCapacityAndCopyPolicies(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	v, e := NewSATWatchList(b)
	satOK(t, e)
	w, _ := newSATBinaryWatch(2, false)
	for i, c := range []uint64{2, 2, 3, 5, 5, 8} {
		satOK(t, v.Push(w))
		if v.Capacity() != c {
			t.Fatal(i, v.Capacity(), c)
		}
	}
	satOK(t, v.ResizeFill(10, w))
	if v.Capacity() != 12 {
		t.Fatal(v.Capacity())
	}
	copy, e := v.Clone(b)
	satOK(t, e)
	satOK(t, v.Reset())
	if copy.Size() != 10 || v.Capacity() != 12 {
		t.Fatal("deep copy/cap retention")
	}
	b.FailAfter(0)
	if e = v.Assign(copy); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if v.Size() != 0 || v.Capacity() != 0 {
		t.Fatal("source assignment finalizes destination first")
	}
	b.FailAfter(-1)
	satOK(t, v.Assign(copy))
	it := v.Iter()
	satOK(t, v.Push(w))
	if _, _, e = it.Next(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	if e = v.Append(v); !errors.Is(e, ErrSATSelfAppend) {
		t.Fatal(e)
	}
	copy.Close()
	v.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	a := satStore(t, nil)
	cv, e := NewSATClauseVector(a, nil)
	satOK(t, e)
	defer cv.Close()
	r := satMk(t, a, satLs(0, 2), false)
	satOK(t, cv.Push(r))
	other, e := cv.Clone(nil)
	satOK(t, e)
	defer other.Close()
	satOK(t, cv.MoveAssign(other))
	if other.Size() != 1 {
		t.Fatal("ptr_vector rvalue assignment is copy")
	}
	wv, e := NewSATClauseWrapperVector(a, nil)
	satOK(t, e)
	defer wv.Close()
	satOK(t, wv.Push(newSATBinaryWrapper(0, 2)))
	wc, e := wv.MoveClone(nil)
	satOK(t, e)
	defer wc.Close()
	if !wv.Empty() || wc.Size() != 1 {
		t.Fatal("move constructor")
	}
}
func TestSATBinaryWrapperReleaseIndexBody(t *testing.T) {
	w := newSATBinaryWrapper(4, 8)
	for _, i := range []uint32{2, ^uint32(0)} {
		l, e := w.Literal(nil, i)
		if satStorageSourceDebug {
			if !errors.Is(e, ErrSATAssertion) {
				t.Fatal(l, e)
			}
		} else if l != 8 || e != nil {
			t.Fatal("source release scalar extraction", l, e)
		}
	}
}
func TestSATBinClauseVectorRealSpecialization(t *testing.T) {
	v, e := NewSATBinClauseVector(nil)
	satOK(t, e)
	defer v.Close()
	x := newSATBinClause(2, 4, true)
	satOK(t, v.ResizeFill(3, x))
	copy, e := v.Clone(nil)
	satOK(t, e)
	defer copy.Close()
	satOK(t, v.MoveAssign(copy))
	if copy.Size() != 3 {
		t.Fatal("svector move assignment copies")
	}
	if has, e := v.Contains(newSATBinClause(9, 1, false)); !has || e != nil {
		t.Fatal("encoded bin equality", has, e)
	}
	satOK(t, v.Swap(copy))
	satOK(t, v.AppendArray([]satBinClause{x}))
	if v.Size() != 4 {
		t.Fatal(v.Size())
	}
}
func TestSATClauseVectorBorrowedForeignCopyLifetime(t *testing.T) {
	a := satStore(t, nil)
	b := satStore(t, nil)
	x := satMk(t, a, satLs(0, 2), false)
	y := satMk(t, b, satLs(4, 6), true)
	src, e := NewSATClauseVector(b, nil)
	satOK(t, e)
	defer src.Close()
	satOK(t, src.Push(y))
	dst, e := NewSATClauseVector(a, nil)
	satOK(t, e)
	defer dst.Close()
	satOK(t, dst.Push(x))
	satOK(t, dst.Assign(src))
	out, e := NewNumericTextStream(NumericTextStreamConfig{})
	satOK(t, e)
	defer out.Close()
	satOK(t, dst.DisplayInto(out))
	text, _ := out.String()
	if text != "(2 3)+*\n" {
		t.Fatal(text)
	}
	satOK(t, b.Close())
	copy, e := dst.Clone(nil)
	satOK(t, e)
	defer copy.Close()
	if got, e := copy.Get(0); got != y || e != nil {
		t.Fatal(got, e)
	}
	if e = copy.DisplayInto(out); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("borrowed closed owner dereference", e)
	}
}
func TestSATScalarVectorReleaseRetainedBodies(t *testing.T) {
	v, e := NewSATActivityDomain(nil)
	satOK(t, e)
	defer v.Close()
	satOK(t, v.Push(8))
	satOK(t, v.Push(9))
	satOK(t, v.Shrink(1))
	x, e := v.Get(1)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(x, e)
		}
	} else if x != 9 || e != nil {
		t.Fatal("retained scalar source body", x, e)
	}
	e = v.Shrink(2)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if e != nil || v.Len() != 2 {
		t.Fatal("release scalar shrink size write", e, v.Len())
	}
	if e = v.SetX(^uint32(0), 3, 0); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if v.Len() != 0 {
		t.Fatal("UINTMAX setx resize(0) publication prefix", v.Len())
	}
}
func TestSATConcreteSourceConstructorsAndFillRecursion(t *testing.T) {
	act, e := NewSATActivityDomainSized(7, nil)
	satOK(t, e)
	defer act.Close()
	if act.Len() != 7 || act.Capacity() != 7 {
		t.Fatal("default size ctor exact capacity", act.Len(), act.Capacity())
	}
	for i := uint32(0); i < 7; i++ {
		x, e := act.Get(i)
		if x != 0 || e != nil {
			t.Fatal(x, e)
		}
	}
	if e = act.FillSize(10, 99); !errors.Is(e, ErrSATNonterminating) {
		t.Fatal(e)
	}
	if act.Len() != 10 || act.Capacity() != 11 {
		t.Fatal("resize prefix before source recursive fill", act.Len(), act.Capacity())
	}
	a := satStore(t, nil)
	cv, e := NewSATClauseVectorSized(a, 3, nil)
	satOK(t, e)
	defer cv.Close()
	if cv.Size() != 3 || cv.Capacity() != 3 {
		t.Fatal(cv.Size(), cv.Capacity())
	}
	empty, e := cv.Get(0)
	if empty != (satStoredClauseRef{}) || e != nil {
		t.Fatal(empty, e)
	}
	w, _ := newSATBinaryWatch(2, false)
	wv, e := NewSATWatchListFill(7, w, nil)
	satOK(t, e)
	defer wv.Close()
	if wv.Capacity() != 8 {
		t.Fatal(wv.Capacity())
	}
	arr, e := NewSATWatchListArray([]satWatched{w, w, w}, nil)
	satOK(t, e)
	defer arr.Close()
	if arr.Capacity() != 3 {
		t.Fatal(arr.Capacity())
	}
	satOK(t, arr.Insert(w))
	if arr.Size() != 4 {
		t.Fatal(arr.Size())
	}
}
func TestSATConcreteVectorIndependentOperations(t *testing.T) {
	v, e := NewSATWatchList(nil)
	satOK(t, e)
	defer v.Close()
	var model []satWatched
	seed := uint32(99)
	for step := 0; step < 3000; step++ {
		seed = seed*1103515245 + 12345
		x, _ := newSATBinaryWatch(satLiteral(seed%32), seed&32 != 0)
		switch seed >> 29 {
		case 0:
			satOK(t, v.Push(x))
			model = append(model, x)
		case 1:
			if len(model) > 0 {
				i := uint32(seed) % uint32(len(model))
				satOK(t, v.Set(i, x))
				model[i] = x
			}
		case 2:
			if len(model) > 0 {
				got, e := v.Pop()
				satOK(t, e)
				if !got.Equal(model[len(model)-1]) {
					t.Fatal(step, got)
				}
				model = model[:len(model)-1]
			}
		case 3:
			n := uint32(seed % 15)
			satOK(t, v.ResizeFill(n, x))
			if int(n) < len(model) {
				model = model[:n]
			} else {
				for len(model) < int(n) {
					model = append(model, x)
				}
			}
		case 4:
			satOK(t, v.Reverse())
			for i, j := 0, len(model)-1; i < j; i, j = i+1, j-1 {
				model[i], model[j] = model[j], model[i]
			}
		case 5:
			if len(model) > 0 {
				chosen := model[int(seed)%len(model)]
				got, e := v.Erase(chosen)
				satOK(t, e)
				if !got {
					t.Fatal(step)
				}
				for i, w := range model {
					if w.Equal(chosen) {
						copy(model[i:], model[i+1:])
						model = model[:len(model)-1]
						break
					}
				}
			}
		case 6:
			satOK(t, v.Fill(x))
			for i := range model {
				model[i] = x
			}
		case 7:
			if len(model) < 10 {
				i := uint32(seed) % uint32(len(model)+1)
				satOK(t, v.InsertAt(i, x))
				model = append(model, x)
				copy(model[int(i)+1:], model[int(i):len(model)-1])
				model[i] = x
			}
		}
		if v.Size() != uint32(len(model)) {
			t.Fatal(step, v.Size(), len(model))
		}
		for i, w := range model {
			got, e := v.Get(uint32(i))
			if e != nil || !got.Equal(w) {
				t.Fatal(step, i, got, w, e)
			}
		}
	}
	it := v.ReverseIter()
	for i := len(model) - 1; i >= 0; i-- {
		got, ok, e := it.Next()
		if e != nil || !ok || !got.Equal(model[i]) {
			t.Fatal(i, got, ok, e)
		}
	}
	satOK(t, v.SetX(12, satWatched{}, satWatched{}))
	satOK(t, v.ReserveFill(20, satWatched{}))
	satOK(t, v.SetEnd(5))
	if x, e := v.GetDefault(10, satWatched{val1: 99}); x.val1 != 99 || e != nil {
		t.Fatal(x, e)
	}
}
func TestSATVectorCrossBudgetSwapAndMoveFailure(t *testing.T) {
	a := arena.NewBudget(1 << 20)
	b := arena.NewBudget(1 << 20)
	v, e := NewSATWatchList(a)
	satOK(t, e)
	w, e := NewSATWatchList(b)
	satOK(t, e)
	x, _ := newSATBinaryWatch(2, true)
	satOK(t, v.ResizeFill(3, x))
	satOK(t, w.ResizeFill(5, x))
	ua, ub := a.Used(), b.Used()
	b.FailAfter(0)
	if e = v.Swap(w); !errors.Is(e, arena.ErrInjectedFailure) || a.Used() != ua || b.Used() != ub || v.Size() != 3 || w.Size() != 5 {
		t.Fatal("second reserve refunds prepared charges", e, a.Used(), b.Used())
	}
	b.FailAfter(-1)
	satOK(t, v.Swap(w))
	if v.Size() != 5 || w.Size() != 3 {
		t.Fatal(v.Size(), w.Size())
	}
	satOK(t, v.MoveAssign(w))
	if w.Size() != 0 || v.Size() != 3 {
		t.Fatal(v.Size(), w.Size())
	}
	v.Close()
	w.Close()
	if a.Used() != 0 || b.Used() != 0 {
		t.Fatal(a.Used(), b.Used())
	}
}
func TestSATVectorIteratorEpochRetiresWithoutWrap(t *testing.T) {
	v, e := NewSATWatchList(nil)
	satOK(t, e)
	defer v.Close()
	v.epoch = ^uint64(0) - 1
	w, _ := newSATBinaryWatch(0, false)
	e = v.ResizeFill(10, w)
	if e != nil {
		t.Fatal(e)
	}
	if v.epoch != ^uint64(0) {
		t.Fatal("epoch must saturate", v.epoch)
	}
	if e = v.Reset(); !errors.Is(e, arena.ErrSlotLimit) {
		t.Fatal(e)
	}
}

// Source vector/set/literal iterators are values. Returning pointers allocates
// an uncharged escaping wrapper even when traversal itself is allocation-free.
func TestSATIteratorConcreteValueContracts(t *testing.T) {
	var _ func(*satWatchList) satWatchListIterator = (*satWatchList).Iter
	var _ func(*satWatchList) satWatchListIterator = (*satWatchList).ReverseIter
	var _ func(*satClauseVector) satClauseVectorIterator = (*satClauseVector).Iter
	var _ func(*satClauseVector) satClauseVectorIterator = (*satClauseVector).ReverseIter
	var _ func(*satClauseWrapperVector) satClauseWrapperVectorIterator = (*satClauseWrapperVector).Iter
	var _ func(*satClauseWrapperVector) satClauseWrapperVectorIterator = (*satClauseWrapperVector).ReverseIter
	var _ func(*satActivityDomain) satActivityDomainIterator = (*satActivityDomain).Iter
	var _ func(*satActivityDomain) satActivityDomainIterator = (*satActivityDomain).ReverseIter
	var _ func(*satBinClauseVector) satBinClauseVectorIterator = (*satBinClauseVector).Iter
	var _ func(*satBinClauseVector) satBinClauseVectorIterator = (*satBinClauseVector).ReverseIter
	var _ func(*satClauseSet) satClauseSetIterator = (*satClauseSet).Iter
	var _ func(*satClauseAllocator, satStoredClauseRef) (satClauseLiteralIterator, error) = (*satClauseAllocator).LiteralIter
}

// Concrete global sinks force the iterator value to escape the call without
// an interface box or a pointer allocation hidden by local inlining.
var satEscapedWatchIter satWatchListIterator
var satEscapedClauseIter satClauseVectorIterator
var satEscapedWrapperIter satClauseWrapperVectorIterator
var satEscapedActivityIter satActivityDomainIterator
var satEscapedBinIter satBinClauseVectorIterator
var satEscapedSetIter satClauseSetIterator
var satEscapedLiteralIter satClauseLiteralIterator

func TestSATIteratorEscapingValuesNoAllocations(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	a := satStore(t, b)
	r := satMk(t, a, satLs(2, 4), false)
	watch, e := NewSATWatchList(b)
	satOK(t, e)
	clauses, e := NewSATClauseVector(a, b)
	satOK(t, e)
	wrappers, e := NewSATClauseWrapperVector(a, b)
	satOK(t, e)
	activity, e := NewSATActivityDomain(b)
	satOK(t, e)
	bins, e := NewSATBinClauseVector(b)
	satOK(t, e)
	set, e := NewSATClauseSet(a, b)
	satOK(t, e)
	w, e := newSATBinaryWatch(2, false)
	satOK(t, e)
	bw := newSATBinaryWrapper(2, 4)
	bin := newSATBinClause(2, 4, false)
	satOK(t, watch.Push(w))
	satOK(t, clauses.Push(r))
	satOK(t, wrappers.Push(bw))
	satOK(t, activity.Push(7))
	satOK(t, bins.Push(bin))
	satOK(t, set.Insert(r))
	used := b.Used()
	b.FailAfter(0)
	cases := []struct {
		name string
		run  func()
	}{
		{"watch", func() { satEscapedWatchIter = watch.Iter() }},
		{"watch-reverse", func() { satEscapedWatchIter = watch.ReverseIter() }},
		{"clause", func() { satEscapedClauseIter = clauses.Iter() }},
		{"clause-reverse", func() { satEscapedClauseIter = clauses.ReverseIter() }},
		{"wrapper", func() { satEscapedWrapperIter = wrappers.Iter() }},
		{"wrapper-reverse", func() { satEscapedWrapperIter = wrappers.ReverseIter() }},
		{"activity", func() { satEscapedActivityIter = activity.Iter() }},
		{"activity-reverse", func() { satEscapedActivityIter = activity.ReverseIter() }},
		{"binary", func() { satEscapedBinIter = bins.Iter() }},
		{"binary-reverse", func() { satEscapedBinIter = bins.ReverseIter() }},
		{"set", func() { satEscapedSetIter = set.Iter() }},
		{"literal", func() {
			var err error
			satEscapedLiteralIter, err = a.LiteralIter(r)
			if err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, c := range cases {
		if allocations := testing.AllocsPerRun(100, c.run); allocations != 0 {
			t.Fatal(c.name, allocations)
		}
		if b.Used() != used {
			t.Fatal(c.name, "iterator charge changed", b.Used(), used)
		}
	}
	if x, ok, e := satEscapedWatchIter.Next(); e != nil || !ok || !x.Equal(w) {
		t.Fatal(x, ok, e)
	}
	if x, ok, e := satEscapedClauseIter.Next(); e != nil || !ok || x != r {
		t.Fatal(x, ok, e)
	}
	if x, ok, e := satEscapedWrapperIter.Next(); e != nil || !ok || x != bw {
		t.Fatal(x, ok, e)
	}
	if x, ok, e := satEscapedActivityIter.Next(); e != nil || !ok || x != 7 {
		t.Fatal(x, ok, e)
	}
	if x, ok, e := satEscapedBinIter.Next(); e != nil || !ok || !x.Equal(bin) {
		t.Fatal(x, ok, e)
	}
	if x, ok, e := satEscapedSetIter.Next(); e != nil || !ok || x != r {
		t.Fatal(x, ok, e)
	}
	if x, ok, e := satEscapedLiteralIter.Next(); e != nil || !ok || x != 2 {
		t.Fatal(x, ok, e)
	}
	// None of the factories consumes the armed fault prefix.
	if e = b.Reserve(1); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	satEscapedWatchIter = satWatchListIterator{}
	satEscapedClauseIter = satClauseVectorIterator{}
	satEscapedWrapperIter = satClauseWrapperVectorIterator{}
	satEscapedActivityIter = satActivityDomainIterator{}
	satEscapedBinIter = satBinClauseVectorIterator{}
	satEscapedSetIter = satClauseSetIterator{}
	satEscapedLiteralIter = satClauseLiteralIterator{}
	b.FailAfter(-1)
	satOK(t, set.Close())
	satOK(t, bins.Close())
	satOK(t, activity.Close())
	satOK(t, wrappers.Close())
	satOK(t, clauses.Close())
	satOK(t, watch.Close())
	satOK(t, a.Delete(r))
	a.Close()
	if b.Used() != 0 {
		t.Fatal("immediate owner cleanup", b.Used())
	}
}

func TestSATScalarSetEndSourcePublication(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	a := satStore(t, b)
	r := satMk(t, a, satLs(2, 4), false)
	c, e := NewSATClauseVectorFill(a, 3, r, b)
	satOK(t, e)
	w := newSATBinaryWrapper(2, 4)
	cv, e := NewSATClauseWrapperVectorFill(a, 3, w, b)
	satOK(t, e)
	act, e := NewSATActivityDomainFill(3, 7, b)
	satOK(t, e)
	bin := newSATBinClause(2, 4, false)
	bv, e := NewSATBinClauseVectorFill(3, bin, b)
	satOK(t, e)
	for _, tc := range []struct {
		name   string
		shrink func(uint32) error
		end    func(uint32) error
		size   func() uint32
	}{
		{"clause", c.Shrink, c.SetEnd, c.Size}, {"wrapper", cv.Shrink, cv.SetEnd, cv.Size},
		{"activity", act.Shrink, act.SetEnd, act.Size}, {"bin", bv.Shrink, bv.SetEnd, bv.Size},
	} {
		t.Run(tc.name, func(t *testing.T) {
			satOK(t, tc.shrink(1))
			used := b.Used()
			b.FailAfter(0)
			satOK(t, tc.end(2)) // set_end has no scalar-vector size assertion in either profile.
			if tc.size() != 2 || b.Used() != used {
				t.Fatal("source retained-capacity size publication", tc.size(), b.Used(), used)
			}
			b.FailAfter(-1)
		})
	}
	if x, e := c.Get(1); e != nil || x != r {
		t.Fatal(x, e)
	}
	if x, e := cv.Get(1); e != nil || x != w {
		t.Fatal(x, e)
	}
	if x, e := act.Get(1); e != nil || x != 7 {
		t.Fatal(x, e)
	}
	if x, e := bv.Get(1); e != nil || !x.Equal(bin) {
		t.Fatal(x, e)
	}
	satOK(t, bv.Close())
	satOK(t, act.Close())
	satOK(t, cv.Close())
	satOK(t, c.Close())
	satOK(t, a.Delete(r))
	a.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestSATScalarExpansionRetainsBackingPrefix(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	a := satStore(t, b)
	r := satMk(t, a, satLs(2, 4), false)
	c, e := NewSATClauseVectorFill(a, 3, r, b)
	satOK(t, e)
	w := newSATBinaryWrapper(2, 4)
	cv, e := NewSATClauseWrapperVectorFill(a, 3, w, b)
	satOK(t, e)
	act, e := NewSATActivityDomainFill(3, 7, b)
	satOK(t, e)
	bin := newSATBinClause(2, 4, false)
	bv, e := NewSATBinClauseVectorFill(3, bin, b)
	satOK(t, e)
	for _, tc := range []struct {
		name     string
		shrink   func(uint32) error
		resize   func() error
		size     func() uint32
		capacity func() uint64
		stats    func() satStorageVectorStats
	}{
		{"clause", c.Shrink, func() error { return c.ResizeFill(8, satStoredClauseRef{}) }, c.Size, c.Capacity, c.Stats},
		{"wrapper", cv.Shrink, func() error { return cv.ResizeFill(8, newSATBinaryWrapper(6, 8)) }, cv.Size, cv.Capacity, cv.Stats},
		{"activity", act.Shrink, func() error { return act.ResizeFill(8, 0) }, act.Size, act.Capacity, act.Stats},
		{"bin", bv.Shrink, func() error { return bv.ResizeFill(8, newSATBinClause(6, 8, false)) }, bv.Size, bv.Capacity, bv.Stats},
	} {
		t.Run(tc.name, func(t *testing.T) {
			satOK(t, tc.shrink(1))
			used := b.Used()
			old := tc.stats().BackingBytes
			b.FailAfter(1)
			e := tc.resize()
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
			if tc.size() != 1 || tc.capacity() != 5 || b.Used() != used-old+tc.stats().BackingBytes {
				t.Fatal("completed growth/source size prefix", tc.size(), tc.capacity(), b.Used())
			}
			b.FailAfter(-1)
		})
	}
	// Get remains source debug guarded; inspect physical backing independently
	// before teardown so no subsequent operation repairs the lost prefix.
	if c.data[:cap(c.data)][2] != r {
		t.Error("clause retained tail lost")
	}
	if cv.data[:cap(cv.data)][2] != w {
		t.Error("wrapper retained tail lost")
	}
	if act.data[:cap(act.data)][2] != 7 {
		t.Error("activity retained tail lost")
	}
	if !bv.data[:cap(bv.data)][2].Equal(bin) {
		t.Error("binary retained tail lost")
	}
	satOK(t, bv.Close())
	satOK(t, act.Close())
	satOK(t, cv.Close())
	satOK(t, c.Close())
	satOK(t, a.Delete(r))
	a.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestSATWatchEraseRetainedLifetimesAndGrowthPrefix(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	v, e := NewSATWatchList(b)
	satOK(t, e)
	for _, n := range []uint32{2, 4, 6, 8, 10} {
		w, e := newSATBinaryWatch(satLiteral(n), false)
		satOK(t, e)
		satOK(t, v.Push(w))
	}
	satOK(t, v.EraseAt(0))
	satOK(t, v.EraseAt(0))
	last, _ := newSATBinaryWatch(10, false)
	got, e := v.Get(4)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(got, e)
		}
	} else if e != nil || !got.Equal(last) {
		t.Fatal("erase did not destroy trailing object", got, e)
	}
	changed, _ := newSATBinaryWatch(12, true)
	e = v.Set(4, changed)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else {
		satOK(t, e)
		last = changed
	}
	used := b.Used()
	old := v.Stats().BackingBytes
	b.FailAfter(1)
	e = v.ResizeFill(9, last)
	if !errors.Is(e, arena.ErrInjectedFailure) || v.Size() != 3 || v.Capacity() != 8 || b.Used() != used-old+v.Stats().BackingBytes {
		t.Fatal("completed first growth prefix", e, v.Size(), v.Capacity(), b.Used())
	}
	if !v.data[:cap(v.data)][4].Equal(last) {
		t.Fatal("trivial reallocate lost live erase tail")
	}
	b.FailAfter(-1)
	// Erase retains high tails, then pop destroys only the current last slot.
	_, e = v.Pop()
	satOK(t, e)
	_, e = v.Get(2)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("pop must end lifetime at the hole", e)
	}
	got, e = v.Get(4)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if e != nil || !got.Equal(last) {
		t.Fatal("higher erase tail remains alive", got, e)
	}
	// A copy constructs only logical size, while move and swap retain lifetimes.
	clone, e := v.Clone(b)
	satOK(t, e)
	_, e = clone.Get(4)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("copy must not construct spare tail", e)
	}
	moved, e := v.MoveClone(b)
	satOK(t, e)
	satOK(t, moved.Swap(clone))
	got, e = clone.Get(4)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if e != nil || !got.Equal(last) {
		t.Fatal("move/swap must transfer retained lifetime", got, e)
	}
	satOK(t, clone.Close())
	satOK(t, moved.Close())
	satOK(t, v.Close())
	if b.Used() != 0 {
		t.Fatal("complete data/lifetime/headers cleanup", b.Used())
	}
}

func TestSATWatchLifetimeMetadataBudget(t *testing.T) {
	b := arena.NewBudget(4096)
	v, e := NewSATWatchList(b)
	satOK(t, e)
	w, _ := newSATBinaryWatch(2, false)
	satOK(t, v.Push(w))
	satOK(t, v.Push(w))
	owner := uint64(reflect.TypeOf(satWatchList{}).Size())
	stride := uint64(reflect.TypeOf(satWatched{}).Size())
	life := uint64(reflect.TypeOf(false).Size())
	if b.Used() != owner+2*(stride+life) || v.Stats().BackingBytes != 2*(stride+life) {
		t.Fatal("actual two backings/header charge", b.Used(), v.Stats())
	}
	// Enough remaining for new watched array alone, insufficient for its bool
	// metadata while old arrays remain owned: reject before either publication.
	filler := b.Limit() - b.Used() - 3*stride
	satOK(t, b.Reserve(filler))
	used := b.Used()
	if e = v.Push(w); !errors.Is(e, arena.ErrBudget) || b.Used() != used || v.Size() != 2 || v.Capacity() != 2 {
		t.Fatal("missing life old/new peak charge", e, b.Used(), v.Size(), v.Capacity())
	}
	b.Release(filler)
	satOK(t, v.Push(w))
	if b.Used() != owner+3*(stride+life) {
		t.Fatal("growth actual accounting", b.Used())
	}
	satOK(t, v.EraseAt(0))
	_, e = v.Pop()
	satOK(t, e)
	if !v.initialized[2] || v.initialized[1] || !v.initialized[0] {
		t.Fatal("erase/pop lifetime hole", v.initialized)
	}
	before := b.Used()
	b.FailAfter(1)
	if clone, e := v.Clone(b); !errors.Is(e, arena.ErrInjectedFailure) || clone != nil || b.Used() != before {
		t.Fatal("copy header unwind", clone, e, b.Used(), before)
	}
	b.FailAfter(-1)
	satOK(t, v.Reset())
	if v.initialized[0] || v.initialized[1] || !v.initialized[2] {
		t.Fatal("reset destroys only logical objects", v.initialized)
	}
	satOK(t, v.Finalize())
	if b.Used() != owner {
		t.Fatal("finalize releases both backings", b.Used())
	}
	satOK(t, v.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
