package sat

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATStoragePointerFreeOwnershipAndConstructorFailure(t *testing.T) {
	for _, v := range []reflect.Type{reflect.TypeOf(StoredClauseRef{}), reflect.TypeOf(satClauseOffset{}), reflect.TypeOf(satClauseHotHeader{}), reflect.TypeOf(satWatched{}), reflect.TypeOf(satJustification{}), reflect.TypeOf(satBinClause{}), reflect.TypeOf(satClauseWrapper{})} {
		satOK(t, arena.ValidateRecordType(v))
	}
	success := false
	for i := int64(0); i < 80; i++ {
		b := arena.NewBudget(1 << 20)
		b.FailAfter(i)
		a, e := NewSATClauseAllocator(satClauseAllocatorConfig{Storage: arena.SmallBytesConfig{Descriptors: arena.Config{Budget: b}}})
		if e == nil {
			satOK(t, a.Close())
			success = true
		} else if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(i, e)
		}
		if b.Used() != 0 {
			t.Fatal("constructor unwind", i, b.Used())
		}
		if success {
			break
		}
	}
	if !success {
		t.Fatal("sweep did not complete")
	}
	b := arena.NewBudget(1 << 20)
	a := satStore(t, b)
	r, e := a.MkClauseN(3, satLs(2), true)
	if !errors.Is(e, arena.ErrBounds) || !a.IsLive(r) {
		t.Fatal(r, e)
	}
	if id, _ := a.ID(r); id != 0 {
		t.Fatal(id)
	}
	if size, _ := a.Size(r); size != 3 {
		t.Fatal(size)
	}
	if cap, _ := a.Capacity(r); cap != 3 {
		t.Fatal(cap)
	}
	if learned, _ := a.Learned(r); !learned {
		t.Fatal("header published before copy")
	}
	satOK(t, a.Delete(r))
	next := satMk(t, a, satLs(0, 2), false)
	if id, _ := a.ID(next); id != 0 {
		t.Fatal("consumed then explicitly recycled source ID", id)
	}
	satOK(t, a.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestSATLargeClauseBackendLedgerBudgetAndFaults(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	a := satStore(t, b)
	base := b.Used()
	b.FailAfter(1)
	if _, e := a.allocateLarge(28); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if b.Used() != base || len(a.large) != 0 || a.largeDirectory != 0 {
		t.Fatal("unpublished prepared directory refunded", b.Used(), base)
	}
	b.FailAfter(-1)
	r, e := a.allocateLarge(28)
	satOK(t, e)
	satOK(t, a.construct(r, 88, 2, satLs(2, 4), true))
	stride := uint64(reflect.TypeOf(satLargeClauseEntry{}).Size())
	if b.Used() != base+28+stride {
		t.Fatal(b.Used(), base, stride)
	}
	satOK(t, a.releaseLarge(r))
	if a.IsLive(r) || b.Used() != base+stride {
		t.Fatal("ledger retained without payload")
	}
	next, e := a.allocateLarge(28)
	satOK(t, e)
	if next.slot != r.slot || next.generation != r.generation+1 {
		t.Fatal(next, r)
	}
	a.large[next.slot].generation = ^uint32(0)
	next.generation = ^uint32(0)
	satOK(t, a.releaseLarge(next))
	retired, e := a.allocateLarge(28)
	satOK(t, e)
	if retired.slot == next.slot || !a.large[next.slot].retired {
		t.Fatal("exhaustion retires identity", retired, next)
	}
	satOK(t, a.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestSATStorageVectorFailurePrefixesAndBudgetPeaks(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	v, e := NewSATWatchList(b)
	satOK(t, e)
	header := b.Used()
	x, _ := newSATBinaryWatch(2, true)
	stride := uint64(reflect.TypeOf(x).Size()) + uint64(reflect.TypeOf(false).Size())
	b.FailAfter(2)
	e = v.ResizeFill(10, x)
	if !errors.Is(e, arena.ErrInjectedFailure) || v.Size() != 0 || v.Capacity() != 3 || b.Used() != header+3*stride {
		t.Fatal("completed growth prefix", v.Size(), v.Capacity(), b.Used(), e)
	}
	b.FailAfter(-1)
	satOK(t, v.ResizeFill(3, x))
	peakBudget := arena.NewBudget(header + 7*stride)
	peak, e := NewSATWatchList(peakBudget)
	satOK(t, e)
	satOK(t, peak.ResizeFill(3, x))
	e = peak.Push(x)
	if !errors.Is(e, arena.ErrBudget) || peak.Size() != 3 || peak.Capacity() != 3 {
		t.Fatal("old plus new backing peak", e)
	}
	peak.Close()
	if peakBudget.Used() != 0 {
		t.Fatal(peakBudget.Used())
	}
	satOK(t, v.Push(x))
	dst, e := NewSATWatchList(b)
	satOK(t, e)
	satOK(t, dst.Push(x))
	b.FailAfter(0)
	e = dst.Assign(v)
	if !errors.Is(e, arena.ErrInjectedFailure) || dst.Size() != 0 || dst.Capacity() != 0 {
		t.Fatal("copy assignment destroys before allocation", e)
	}
	b.FailAfter(-1)
	v.Close()
	dst.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestSATVarQueueIndependentMutableActivityModel(t *testing.T) {
	a, e := NewSATActivityDomain(nil)
	satOK(t, e)
	defer a.Close()
	satOK(t, a.Resize(64))
	q, e := NewSATVarQueue(a, nil)
	satOK(t, e)
	defer q.Close()
	var live [64]bool
	seed := uint32(7)
	for step := 0; step < 12000; step++ {
		seed = seed*1664525 + 1013904223
		v := seed % 64
		switch seed >> 28 {
		case 0, 1, 2:
			if !live[v] {
				satOK(t, q.MkVar(BoolVar(v)))
				live[v] = true
			}
		case 3, 4:
			satOK(t, q.DelVar(BoolVar(v)))
			live[v] = false
		case 5, 6, 7:
			old, _ := a.Get(v)
			satOK(t, a.Set(v, old+1))
			satOK(t, q.ActivityIncreased(BoolVar(v)))
		default:
			if !q.Empty() {
				got, e := q.NextVar()
				satOK(t, e)
				if !live[got] {
					t.Fatal("absent pop", step, got)
				}
				priority, _ := a.Get(uint32(got))
				for i, present := range live {
					p, _ := a.Get(uint32(i))
					if present && p > priority {
						t.Fatal("not maximal", step, got, i)
					}
				}
				live[got] = false
			}
		}
		count := uint32(0)
		for i, present := range live {
			if q.Contains(BoolVar(i)) != present {
				t.Fatal("membership", step, i)
			}
			if present {
				count++
			}
		}
		if q.Size() != count {
			t.Fatal(step, q.Size(), count)
		}
		if ok, e := q.CheckInvariant(); !ok || e != nil {
			t.Fatal(step, ok, e)
		}
	}
}
func TestSATStorageSourceTextAndWidthActions(t *testing.T) {
	a := satStore(t, nil)
	r := satMk(t, a, satLs(0, 3), true)
	satOK(t, a.SetRemoved(r, true))
	s, e := util.NewNumericTextStream(util.NumericTextStreamConfig{})
	satOK(t, e)
	defer s.Close()
	satOK(t, a.DisplayInto(r, s))
	satOK(t, s.AppendCString("|"))
	satOK(t, newSATClauseWrapper(r).DisplayInto(a, s))
	text, _ := s.String()
	if text != "(0 -1)x+*|(0 -1)" {
		t.Fatal(text)
	}
	list, e := NewSATWatchList(nil)
	satOK(t, e)
	defer list.Close()
	bin, _ := newSATBinaryWatch(2, true)
	ter, _ := newSATTernaryWatch(2, 4)
	off, _ := a.Offset(r)
	cls, _ := newSATClauseWatch(6, off)
	ext, _ := newSATExternalWatch(3)
	for _, w := range []satWatched{ext, cls, ter, bin, ter} {
		satOK(t, list.Push(w))
	}
	satOK(t, list.SortByKind())
	satOK(t, s.AppendCString("|"))
	satOK(t, list.DisplayInto(a, s))
	text, _ = s.String()
	if text != "(0 -1)x+*|(0 -1)|1* (1 2) (1 2) 3 (3 (0 -1)x+*)" {
		t.Fatal(text)
	}
	satOK(t, s.AppendCString("|"))
	satOK(t, s.SetWidth(4))
	satOK(t, satLiteralInto(s, 0))
	satOK(t, s.SetWidth(4))
	satOK(t, satLiteralInto(s, 3))
	text, _ = s.String()
	if text != "(0 -1)x+*|(0 -1)|1* (1 2) (1 2) 3 (3 (0 -1)x+*)|    0   -1" {
		t.Fatal(text)
	}
	if s.State().Width != 0 {
		t.Fatal(s.State())
	}
}
func TestSATStorageWarmNoAllocationsAndClosedBorrow(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	a, e := NewSATClauseAllocator(satClauseAllocatorConfig{Storage: arena.SmallBytesConfig{Descriptors: arena.Config{Budget: b}}, PooledOverride: true})
	satOK(t, e)
	ls := [2]Literal{0, 2}
	r, e := a.MkClause(ls[:], false)
	satOK(t, e)
	satOK(t, a.Delete(r))
	if n := testing.AllocsPerRun(100, func() {
		r, e := a.MkClause(ls[:], false)
		if e != nil {
			panic(e)
		}
		if e = a.Delete(r); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal(n)
	}
	a.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	act, e := NewSATActivityDomain(nil)
	satOK(t, e)
	satOK(t, act.Resize(4))
	q, e := NewSATVarQueue(act, nil)
	satOK(t, e)
	defer q.Close()
	satOK(t, q.MkVar(0))
	satOK(t, q.MkVar(1))
	satOK(t, q.Reset())
	if n := testing.AllocsPerRun(100, func() {
		if e = q.MkVar(0); e != nil {
			panic(e)
		}
		if e = q.MkVar(1); e != nil {
			panic(e)
		}
		if _, e = q.NextVar(); e != nil {
			panic(e)
		}
		if e = q.Reset(); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal(n)
	}
	satOK(t, act.Close())
	satOK(t, q.MkVar(0))
	if e = q.MkVar(1); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("comparator actual borrowed owner", e)
	}
}
func TestSATVarQueueConstructorCloneFailureAndMovedReset(t *testing.T) {
	act, e := NewSATActivityDomain(nil)
	satOK(t, e)
	defer act.Close()
	satOK(t, act.Resize(4))
	success := false
	for k := int64(0); k < 32; k++ {
		b := arena.NewBudget(1 << 20)
		b.FailAfter(k)
		q, e := NewSATVarQueue(act, b)
		if e == nil {
			satOK(t, q.Close())
			success = true
		} else if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(k, e)
		}
		if b.Used() != 0 {
			t.Fatal("constructor unwinds every completed growth", k, b.Used())
		}
		if success {
			break
		}
	}
	if !success {
		t.Fatal("no success")
	}
	q, e := NewSATVarQueue(act, nil)
	satOK(t, e)
	defer q.Close()
	satOK(t, q.MkVar(0))
	for k := int64(0); k < 4; k++ {
		b := arena.NewBudget(1 << 20)
		b.FailAfter(k)
		copy, e := q.Clone(b)
		if e == nil {
			copy.Close()
		} else if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(k, e)
		}
		if b.Used() != 0 || q.Size() != 1 {
			t.Fatal("copy constructor cleanup/source unchanged", k, b.Used())
		}
	}
	m, e := q.MoveClone(nil)
	satOK(t, e)
	defer m.Close()
	if q.Empty() {
		t.Fatal("source moved vector size0 is not size1 sentinel empty")
	}
	if _, e = q.NextVar(); !errors.Is(e, arena.ErrBounds) && !errors.Is(e, ErrSATAssertion) {
		t.Fatal(e)
	}
	satOK(t, q.Reset())
	if !q.Empty() || len(q.values) != 1 || q.values[0] != -1 {
		t.Fatal("source reset recreates moved sentinel")
	}
	satOK(t, q.MkVar(3))
	if q.Size() != 1 {
		t.Fatal(q.Size())
	}
}
func TestSATStorageOwnerIdentityExhaustionAndOOMRecycle(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	saved := satStorageNextOwner.Load()
	satStorageNextOwner.Store(^uint64(0))
	a, e := NewSATClauseAllocator(satClauseAllocatorConfig{Storage: arena.SmallBytesConfig{Descriptors: arena.Config{Budget: b}}})
	satStorageNextOwner.Store(saved)
	if a != nil || !errors.Is(e, arena.ErrSlotLimit) || b.Used() != 0 {
		t.Fatal(a, e, b.Used())
	}
	oom := true
	a, e = NewSATClauseAllocator(satClauseAllocatorConfig{Storage: arena.SmallBytesConfig{Descriptors: arena.Config{Budget: b}}, OutOfMemory: func() bool { return oom }})
	satOK(t, e)
	defer a.Close()
	r := satMk(t, a, satLs(0, 2), false)
	satOK(t, a.Delete(r))
	if a.IsLive(r) || len(a.ids.Free) != 0 {
		t.Fatal("OOM skips only source ID recycle, not physical release")
	}
	next := satMk(t, a, satLs(2, 4), false)
	if id, _ := a.ID(next); id != 1 {
		t.Fatal(id)
	}
	copied := *a
	if _, e = copied.MkClause(satLs(0, 2), false); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	oom = false
	satOK(t, a.Delete(next))
	satOK(t, a.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
