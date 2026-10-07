// Licensed under the MIT license.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestPrimeGeneratorSparseReadAndReversedLoopPrefixes(t *testing.T) {
	for _, f := range []struct {
		idx, count uint32
		want       error
	}{{500, 191, arena.ErrBounds}, {191, 2016, ErrPrimeNontermination}} {
		budget := arena.NewBudget(1 << 24)
		g, e := NewPrimeGenerator(budget)
		if e != nil {
			t.Fatal(e)
		}
		n := g.Stats().Count
		if n != 32 {
			t.Fatal("eager128 source batch", n)
		}
		if _, e = g.At(f.idx); !errors.Is(e, f.want) {
			t.Fatal(f.idx, e)
		}
		if n = g.Stats().Count; n != f.count {
			t.Fatal("source completed growth prefix", f.idx, n, f.count)
		}
		if e = g.Close(); e != nil {
			t.Fatal(e)
		}
		if budget.Used() != 0 {
			t.Fatal(budget.Used())
		}
	}
}
func TestPrimeIteratorIndexBeforeRefusalWrapAndColdOwner(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	g, e := NewPrimeGenerator(budget)
	if e != nil {
		t.Fatal(e)
	}
	it, e := NewPrimeIterator(g)
	if e != nil {
		t.Fatal(e)
	}
	it.Index = ^uint32(0)
	if _, e = it.Next(); !errors.Is(e, ErrPrimeCapacity) || it.Index != 0 {
		t.Fatal("source increment/wrap BEFORE capacity exception", it.Index, e)
	}
	p, e := it.Next()
	if e != nil || p != 2 || it.Index != 1 {
		t.Fatal(p, it.Index, e)
	}
	budget.FailAfter(0)
	it.Index = 32
	if _, e = it.Next(); !errors.Is(e, arena.ErrInjectedFailure) || it.Index != 33 {
		t.Fatal("increment BEFORE growth refusal", it.Index, e)
	}
	budget.FailAfter(-1)
	copyOwner := *g
	if _, e = copyOwner.At(0); !errors.Is(e, arena.ErrConfig) {
		t.Fatal("copied cold owner", e)
	}
	if e = g.Finalize(); e != nil {
		t.Fatal(e)
	}
	if g.Stats().Count != 0 || g.Stats().Capacity != 0 {
		t.Fatal("source finalization frees array", g.Stats())
	}
	if e = g.Close(); e != nil {
		t.Fatal(e)
	}
	if budget.Used() != 0 {
		t.Fatal(budget.Used())
	}
}

func TestPrimeGeneratorConstructorAndSparseFailureAccounting(t *testing.T) {
	refused, success := 0, 0
	for ordinal := int64(0); ordinal < 80; ordinal++ {
		b := arena.NewBudget(1 << 24)
		b.FailAfter(ordinal)
		g, e := NewPrimeGenerator(b)
		if e != nil {
			refused++
			if !errors.Is(e, arena.ErrInjectedFailure) || g != nil || b.Used() != 0 {
				t.Fatal(ordinal, g, e, b.Used())
			}
		} else {
			success++
			if e = g.Close(); e != nil {
				t.Fatal(e)
			}
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
			break
		}
	}
	if refused == 0 || success != 1 {
		t.Fatal(refused, success)
	}
	late, finished := 0, 0
	for ordinal := int64(0); ordinal < 160; ordinal++ {
		b := arena.NewBudget(1 << 24)
		g, e := NewPrimeGenerator(b)
		if e != nil {
			t.Fatal(e)
		}
		b.FailAfter(ordinal)
		_, e = g.At(191)
		st := g.Stats()
		if b.Used() != st.TotalBytes || st.TotalBytes != st.OwnerBytes+st.BackingBytes {
			t.Fatal("temporary todo refunded immediately", ordinal, e, st, b.Used())
		}
		if e != nil && !errors.Is(e, ErrPrimeNontermination) && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(ordinal, e)
		}
		if st.Count > 191 && errors.Is(e, arena.ErrInjectedFailure) {
			late++
		}
		if errors.Is(e, ErrPrimeNontermination) {
			finished++
		}
		// Independent primality oracle checks every actually published prefix.
		for i := uint32(0); i < st.Count; i++ {
			p, err := g.At(i)
			if err != nil {
				t.Fatal(err)
			}
			for d := uint64(2); d <= p/d; d++ {
				if p%d == 0 {
					t.Fatal("published composite", ordinal, i, p)
				}
			}
		}
		b.FailAfter(-1)
		if e = g.Close(); e != nil {
			t.Fatal(e)
		}
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
		if finished != 0 {
			break
		}
	}
	if late == 0 || finished != 1 {
		t.Fatal("actual late reversed-loop refusals", late, finished)
	}
}
func TestPrimeGeneratorWarmCachedReadAndFinalizeDomain(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	g, e := NewPrimeGenerator(b)
	if e != nil {
		t.Fatal(e)
	}
	prefix := []uint64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37}
	for i, p := range prefix {
		q, e := g.At(uint32(i))
		if e != nil || q != p {
			t.Fatal(i, q, e)
		}
	}
	b.FailAfter(0)
	alloc := testing.AllocsPerRun(100, func() {
		p, e := g.At(31)
		if e != nil || p != 131 {
			panic("warmed cached endpoint")
		}
	})
	if alloc != 0 {
		t.Fatal("cached reads allocate", alloc)
	}
	b.FailAfter(-1)
	if e = g.Finalize(); e != nil {
		t.Fatal(e)
	}
	// Source finalized empty back() is undefined. Go checks its actual read.
	if _, e = g.At(0); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if e = g.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = g.At(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	var zero PrimeGenerator
	if e = zero.Close(); !errors.Is(e, arena.ErrConfig) {
		t.Fatal(e)
	}
}
func TestPrimeGeneratorUnsignedSourceExtent(t *testing.T) {
	n, oldBytes, newBytes, e := primeSourceCapacity(536870912)
	if e != nil || n != 805306368 || oldBytes != 8 || newBytes != 2147483656 {
		t.Fatal(n, oldBytes, newBytes, e)
	}
	if _, _, e = primeNextCapacity(536870912, ^uint64(0)); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual full array versus wrapped source allocation", e)
	}
	n, oldBytes, newBytes, e = primeSourceCapacity(357913941)
	if !errors.Is(e, arena.ErrBounds) || n != 536870912 || newBytes > oldBytes {
		t.Fatal(n, oldBytes, newBytes, e)
	}
}
