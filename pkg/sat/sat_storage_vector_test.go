package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

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

func TestSATNativeClauseVectorIndependentOperations(t *testing.T) {
	a := satStore(t, nil)
	var refs []StoredClauseRef
	for i := uint32(0); i < 17; i++ {
		refs = append(refs, satMk(t, a, satLs(i*2, i*2+3), false))
	}
	b := arena.NewBudget(1 << 20)
	v, e := NewSATClauseVector(a, b)
	satOK(t, e)
	defer v.Close()
	other, e := NewSATClauseVector(a, b)
	satOK(t, e)
	defer other.Close()
	for _, r := range refs {
		satOK(t, other.Push(r))
	}
	before := b.Used()
	b.FailAfter(0)
	for i := 0; i < 4097; i++ {
		satOK(t, v.Push(refs[i%len(refs)]))
	}
	if v.Size() != 4097 {
		t.Fatal(v.Size())
	}
	for i := uint32(0); i < v.Size(); i++ {
		got, e := v.Get(i)
		satOK(t, e)
		if got != refs[int(i)%len(refs)] {
			t.Fatal(i, got)
		}
	}
	satOK(t, v.Reset())
	var model []StoredClauseRef
	seed := uint32(5831)
	for step := 0; step < 4096; step++ {
		seed = seed*1664525 + 1013904223
		r := refs[int(seed>>8)%len(refs)]
		switch seed % 11 {
		case 0:
			satOK(t, v.Reset())
			model = nil
		case 1:
			satOK(t, v.Assign(other))
			model = append(model[:0], refs...)
		case 2:
			if len(model) > 0 {
				i := int(seed>>16) % len(model)
				satOK(t, v.Set(uint32(i), r))
				model[i] = r
			}
		case 3:
			n := 0
			if len(model) > 0 {
				n = int(seed>>16) % (len(model) + 1)
			}
			satOK(t, v.Shrink(uint32(n)))
			model = model[:n]
		case 4:
			yes, e := v.Erase(r)
			satOK(t, e)
			found := false
			for i, x := range model {
				if x == r {
					model = append(model[:i], model[i+1:]...)
					found = true
					break
				}
			}
			if yes != found {
				t.Fatal("erase oracle", step, yes, found)
			}
		default:
			satOK(t, v.Push(r))
			model = append(model, r)
		}
		if int(v.Size()) != len(model) {
			t.Fatal("length oracle", step, v.Size(), len(model))
		}
		for i, want := range model {
			got, e := v.Get(uint32(i))
			satOK(t, e)
			if got != want {
				t.Fatal("content oracle", step, i, got, want)
			}
		}
	}
	satOK(t, v.Assign(v))
	satOK(t, v.Finalize())
	if v.data != nil || b.Used() != before || v.Stats().BackingBytes != 0 || v.Stats().BindingBytes != 0 {
		t.Fatal("native backing charged or retained", b.Used(), before)
	}
	for _, r := range refs {
		if !a.IsLive(r) {
			t.Fatal("vector owned a clause", r)
		}
	}
	b.FailAfter(-1)
	satOK(t, other.Close())
	satOK(t, v.Close())
	if b.Used() != 0 {
		t.Fatal("cold headers leaked", b.Used())
	}
}

func TestSATNativeClauseVectorBorrowedCopyLifetime(t *testing.T) {
	a, foreign := satStore(t, nil), satStore(t, nil)
	x := satMk(t, a, satLs(0, 2), false)
	y := satMk(t, foreign, satLs(4, 6), true)
	b := arena.NewBudget(1 << 20)
	dst, e := NewSATClauseVector(a, b)
	satOK(t, e)
	defer dst.Close()
	src, e := NewSATClauseVector(foreign, b)
	satOK(t, e)
	defer src.Close()
	satOK(t, dst.Push(x))
	satOK(t, src.Push(y))
	before := b.Used()
	b.FailAfter(0)
	satOK(t, dst.BindAllocator(foreign))
	satOK(t, dst.Assign(src))
	copy, e := dst.Clone(b)
	if !errors.Is(e, arena.ErrInjectedFailure) || copy != nil || b.Used() != before {
		t.Fatal("actual clone header refusal", copy, e, b.Used(), before)
	}
	b.FailAfter(1)
	copy, e = dst.Clone(b)
	satOK(t, e)
	defer copy.Close()
	b.FailAfter(0)
	satOK(t, dst.Set(0, x))
	got, e := copy.Get(0)
	satOK(t, e)
	if got != y {
		t.Fatal("native copy shared backing", got)
	}
	satOK(t, src.Close())
	if !a.IsLive(x) || !foreign.IsLive(y) {
		t.Fatal("native owner adopted borrowed clauses")
	}
	satOK(t, foreign.Close())
	bound, e := copy.resolve(y.owner)
	satOK(t, e)
	if _, e = bound.Removed(y); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("actual borrowed owner guard", e)
	}
	got, e = copy.Get(0)
	satOK(t, e)
	if got != y {
		t.Fatal("scalar reference copy dereferenced closed clause", got)
	}
	satOK(t, copy.Close())
	satOK(t, dst.Close())
	if !a.IsLive(x) || b.Used() != 0 {
		t.Fatal("native copy cleanup", a.IsLive(x), b.Used())
	}
	b.FailAfter(-1)
}

func TestSATNativeWrapperAndBinarySliceGrowth(t *testing.T) {
	a := satStore(t, nil)
	r := satMk(t, a, satLs(0, 2), false)
	b := arena.NewBudget(1 << 20)
	wrappers, e := NewSATClauseWrapperVector(a, b)
	satOK(t, e)
	bins, e := NewSATBinClauseVector(b)
	satOK(t, e)
	before := b.Used()
	b.FailAfter(0)
	for i := uint32(0); i < 4097; i++ {
		satOK(t, wrappers.Push(newSATBinaryWrapper(Literal(i*2), Literal(i*2+2))))
		satOK(t, bins.Push(newSATBinClause(Literal(i*2), Literal(i*2+2), i&1 != 0)))
	}
	for i := uint32(0); i < 4097; i++ {
		w, e := wrappers.Get(i)
		satOK(t, e)
		if w != newSATBinaryWrapper(Literal(i*2), Literal(i*2+2)) {
			t.Fatal("wrapper order", i, w)
		}
		x, e := bins.Get(i)
		satOK(t, e)
		if x != newSATBinClause(Literal(i*2), Literal(i*2+2), i&1 != 0) {
			t.Fatal("binary order", i, x)
		}
	}
	w, e := wrappers.Back()
	satOK(t, e)
	if w != newSATBinaryWrapper(8192, 8194) {
		t.Fatal(w)
	}
	x, e := bins.Pop()
	satOK(t, e)
	if x != newSATBinClause(8192, 8194, false) || bins.Size() != 4096 {
		t.Fatal(x, bins.Size())
	}
	satOK(t, wrappers.Reset())
	satOK(t, bins.Reset())
	satOK(t, wrappers.Push(newSATClauseWrapper(r)))
	satOK(t, bins.Push(newSATBinClause(2, 4, true)))
	erased, e := bins.Erase(newSATBinClause(2, 4, true))
	satOK(t, e)
	if !erased || bins.Size() != 0 {
		t.Fatal("binary erase", erased, bins.Size())
	}
	satOK(t, wrappers.Finalize())
	satOK(t, bins.Finalize())
	if !a.IsLive(r) || wrappers.data != nil || bins.data != nil || b.Used() != before {
		t.Fatal("native backing owned clause or charged memory")
	}
	copied := *wrappers
	if e = copied.Push(newSATClauseWrapper(r)); !errors.Is(e, ErrSATOwner) {
		t.Fatal("shallow owner copy accepted", e)
	}
	satOK(t, wrappers.Close())
	satOK(t, bins.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	b.FailAfter(-1)
}

func TestSATNativeActivityGrowthAndLogicalBounds(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	v, e := NewSATActivityDomain(b)
	satOK(t, e)
	before := b.Used()
	b.FailAfter(0)
	for i := uint32(0); i < 4097; i++ {
		satOK(t, v.Push(i*i))
	}
	for i := uint32(0); i < 4097; i++ {
		got, e := v.Get(i)
		satOK(t, e)
		if got != i*i {
			t.Fatal(i, got)
		}
	}
	satOK(t, v.Resize(5))
	if _, e = v.Get(5); satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	satOK(t, v.Resize(4097))
	for i := uint32(5); i < 4097; i++ {
		got, e := v.Get(i)
		satOK(t, e)
		if got != 0 {
			t.Fatal("regrown native values must start at zero", i, got)
		}
	}
	satOK(t, v.Set(0, 71))
	got, e := v.Get(0)
	satOK(t, e)
	if got != 71 || b.Used() != before {
		t.Fatal(got, b.Used(), before)
	}
	satOK(t, v.Close())
	satOK(t, v.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	b.FailAfter(-1)
}
