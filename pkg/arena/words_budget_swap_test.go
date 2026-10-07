package arena

import (
	"errors"
	"testing"
)

func wordsOwnedBytes(v *Words) uint64 {
	s := v.Stats()
	return s.OwnerBytes + s.PayloadBytes + s.BlockMetadataBytes + s.DirectoryBytes + s.Descriptors.OwnerBytes + s.Descriptors.RecordBytes + s.Descriptors.MetadataBytes + s.Descriptors.DirectoryBytes
}
func budgetSwapWords(t *testing.T, b *Budget, n uint32, value uint32) (*Words, VectorRef) {
	t.Helper()
	w, e := NewWords(VectorConfig{PageSize: 4, Descriptors: Config{PageSize: 2, Budget: b}})
	if e != nil {
		t.Fatal(e)
	}
	r, e := w.Allocate(n, n)
	if e != nil {
		t.Fatal(e)
	}
	if n > 0 {
		if e = w.Write(r, 0, value); e != nil {
			t.Fatal(e)
		}
	}
	// Retain a free payload extent and descriptor generation ledger in each graph.
	spare, e := w.Allocate(2, 2)
	if e != nil {
		t.Fatal(e)
	}
	if e = w.Free(spare); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(w.Close)
	return w, r
}
func TestWordsSwapBudgetsAtomicFailures(t *testing.T) {
	for _, second := range []bool{false, true} {
		t.Run(map[bool]string{false: "first", true: "second"}[second], func(t *testing.T) {
			ab, bb := NewBudget(1<<20), NewBudget(1<<20)
			ab.Reserve(17)
			bb.Reserve(23)
			a, ar := budgetSwapWords(t, ab, 1, 41)
			b, br := budgetSwapWords(t, bb, 9, 91)
			beforeA, beforeB := a.Stats(), b.Stats()
			usedA, usedB := ab.Used(), bb.Used()
			if second {
				bb.FailAfter(3)
				ab.FailAfter(0)
			} else {
				bb.FailAfter(0)
				ab.FailAfter(4)
			}
			remA, remB := ab.remaining, bb.remaining
			if second {
				remB--
			} // successful reservation consumes the injector prefix even when refunded.
			if e := a.SwapBudgets(b); !errors.Is(e, ErrInjectedFailure) {
				t.Fatal("missing reserve failure", e)
			}
			if ab.Used() != usedA || bb.Used() != usedB || ab.remaining != remA || bb.remaining != remB {
				t.Fatal("failed swap changed budget state")
			}
			if a.config.Descriptors.Budget != ab || a.descriptors.config.Budget != ab || b.config.Descriptors.Budget != bb || b.descriptors.config.Budget != bb {
				t.Fatal("failed swap rebound owner")
			}
			if a.Stats() != beforeA || b.Stats() != beforeB {
				t.Fatal("failed swap changed storage")
			}
			if v, e := a.Read(ar, 0); e != nil || v != 41 {
				t.Fatal("failed swap lost a")
			}
			if v, e := b.Read(br, 0); e != nil || v != 91 {
				t.Fatal("failed swap lost b")
			}
			ab.FailAfter(-1)
			bb.FailAfter(-1)
			a.Close()
			b.Close()
			if ab.Used() != 17 || bb.Used() != 23 {
				t.Fatal("failed swap close accounting")
			}
		})
	}
}
func TestWordsSwapBudgetsNestedOwnersAndOperations(t *testing.T) {
	ab, bb := NewBudget(1<<20), NewBudget(1<<20)
	ab.Reserve(17)
	bb.Reserve(23)
	a, ar := budgetSwapWords(t, ab, 1, 41)
	b, br := budgetSwapWords(t, bb, 9, 91)
	as, bs := a.Stats(), b.Stats()
	ag, bg := wordsOwnedBytes(a), wordsOwnedBytes(b)
	if ag+17 != ab.Used() || bg+23 != bb.Used() {
		t.Fatal("fixture omitted nested owners")
	}
	if e := a.SwapBudgets(b); e != nil {
		t.Fatal(e)
	}
	if ab.Used() != bg+17 || bb.Used() != ag+23 {
		t.Fatal("opposite graph budget total")
	}
	if a.Stats() != as || b.Stats() != bs || !a.IsLive(ar) || !b.IsLive(br) {
		t.Fatal("swap moved storage or identity")
	}
	if a.config.Descriptors.Budget != bb || a.descriptors.config.Budget != bb || b.config.Descriptors.Budget != ab || b.descriptors.config.Budget != ab {
		t.Fatal("nested budget rebinding")
	}
	if v, e := a.Read(ar, 0); e != nil || v != 41 {
		t.Fatal("a value")
	}
	if v, e := b.Read(br, 0); e != nil || v != 91 {
		t.Fatal("b value")
	}
	bb.FailAfter(0)
	before := ab.Used()
	if _, e := a.Allocate(500, 500); !errors.Is(e, ErrInjectedFailure) {
		t.Fatal("growth routed to old budget", e)
	}
	if ab.Used() != before {
		t.Fatal("a growth touched old budget")
	}
	bb.FailAfter(-1)
	next, e := a.Allocate(500, 500)
	if e != nil {
		t.Fatal(e)
	}
	a.Free(next)
	a.Reset()
	b.Reset()
	a.Close()
	b.Close()
	if ab.Used() != 17 || bb.Used() != 23 {
		t.Fatal("transferred graph close leaked")
	}
}
func TestWordsSwapBudgetsNilAndSame(t *testing.T) {
	for _, bothNil := range []bool{false, true} {
		var bb *Budget
		if !bothNil {
			bb = NewBudget(1 << 20)
			bb.Reserve(23)
		}
		a, ar := budgetSwapWords(t, nil, 1, 31)
		b, br := budgetSwapWords(t, bb, 9, 79)
		ag := wordsOwnedBytes(a)
		if e := a.SwapBudgets(b); e != nil {
			t.Fatal(e)
		}
		if !bothNil && bb.Used() != ag+23 {
			t.Fatal("nil/unlimited transfer charge")
		}
		if v, e := a.Read(ar, 0); e != nil || v != 31 {
			t.Fatal("nil a")
		}
		if v, e := b.Read(br, 0); e != nil || v != 79 {
			t.Fatal("nil b")
		}
		a.Close()
		b.Close()
		if !bothNil && bb.Used() != 23 {
			t.Fatal("nil transfer close")
		}
	}
	shared := NewBudget(1 << 20)
	a, _ := budgetSwapWords(t, shared, 1, 21)
	b, _ := budgetSwapWords(t, shared, 9, 31)
	used := shared.Used()
	shared.FailAfter(0)
	if e := a.SwapBudgets(b); e != nil || shared.Used() != used || shared.remaining != 0 {
		t.Fatal("same budget must skip reservations", e)
	}
	if e := a.SwapBudgets(a); e != nil {
		t.Fatal("self swap")
	}
	a.Close()
	if e := a.SwapBudgets(b); !errors.Is(e, ErrClosed) {
		t.Fatal("closed a")
	}
	if e := b.SwapBudgets(a); !errors.Is(e, ErrClosed) {
		t.Fatal("closed b")
	}
	b.Close()
	if shared.Used() != 0 {
		t.Fatal("same budget cleanup")
	}
}
func TestWordsSwapBudgetsDestinationLimitAndWarmAllocation(t *testing.T) {
	ab, bb := NewBudget(1<<20), NewBudget(1<<20)
	a, _ := budgetSwapWords(t, ab, 1, 31)
	b, _ := budgetSwapWords(t, bb, 9, 79)
	usedA, usedB := ab.Used(), bb.Used()
	bb.limit = usedB + wordsOwnedBytes(a) - 1
	if e := a.SwapBudgets(b); !errors.Is(e, ErrBudget) {
		t.Fatal("must account destination old+incoming peak", e)
	}
	if ab.Used() != usedA || bb.Used() != usedB {
		t.Fatal("limit failure changed charge")
	}
	bb.limit = 1 << 20
	if n := testing.AllocsPerRun(1000, func() {
		if e := a.SwapBudgets(b); e != nil {
			panic(e)
		}
		if e := a.SwapBudgets(b); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatalf("warm swap allocations%v", n)
	}
	a.Close()
	b.Close()
	if ab.Used() != 0 || bb.Used() != 0 {
		t.Fatal("warm swap leaked")
	}
}

func TestWordsSwapBudgetsOverflowBeforeReservations(t *testing.T) {
	for _, opposite := range []bool{false, true} {
		ab, bb := NewBudget(1<<20), NewBudget(1<<20)
		a, _ := budgetSwapWords(t, ab, 1, 31)
		b, _ := budgetSwapWords(t, bb, 9, 79)
		usedA, usedB := ab.Used(), bb.Used()
		bad := a
		if opposite {
			bad = b
		}
		previous := bad.directoryBytes
		bad.directoryBytes = ^uint64(0)
		if opposite {
			ab.FailAfter(0)
			bb.FailAfter(3)
		} else {
			ab.FailAfter(3)
			bb.FailAfter(0)
		}
		remA, remB := ab.remaining, bb.remaining
		e := a.SwapBudgets(b)
		bad.directoryBytes = previous
		if !errors.Is(e, ErrBounds) {
			t.Fatal("overflow classification", e)
		}
		if ab.Used() != usedA || bb.Used() != usedB || ab.remaining != remA || bb.remaining != remB {
			t.Fatal("overflow changed charges or reserved before validation")
		}
		if a.config.Descriptors.Budget != ab || b.config.Descriptors.Budget != bb {
			t.Fatal("overflow rebound")
		}
		ab.FailAfter(-1)
		bb.FailAfter(-1)
		a.Close()
		b.Close()
	}
}
