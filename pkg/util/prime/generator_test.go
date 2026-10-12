package prime

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestPrimeGeneratorSparseReadAndReversedLoopPrefixes(t *testing.T) {
	for _, f := range []struct {
		idx, count uint32
		want       error
	}{{500, 191, arena.ErrBounds}, {191, 2016, ErrNontermination}} {
		budget := arena.NewBudget(1 << 24)
		g, e := NewGenerator(budget)
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
func TestPrimeIteratorIndexBeforeNativeGrowthWrapAndColdOwner(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	g, e := NewGenerator(budget)
	if e != nil {
		t.Fatal(e)
	}
	it, e := NewIterator(g)
	if e != nil {
		t.Fatal(e)
	}
	it.Index = ^uint32(0)
	if _, e = it.Next(); !errors.Is(e, ErrCapacity) || it.Index != 0 {
		t.Fatal("source increment/wrap BEFORE capacity exception", it.Index, e)
	}
	p, e := it.Next()
	if e != nil || p != 2 || it.Index != 1 {
		t.Fatal(p, it.Index, e)
	}
	budget.FailAfter(0)
	it.Index = 32
	before := budget.Used()
	if p, e = it.Next(); e != nil || p != 137 || it.Index != 33 || budget.Used() != before {
		t.Fatal("native cache growth charged arena", p, it.Index, e, budget.Used(), before)
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

func TestPrimeGeneratorConstructorAndSparseNativeStorage(t *testing.T) {
	for ordinal := int64(0); ordinal < 2; ordinal++ {
		b := arena.NewBudget(1 << 24)
		b.FailAfter(ordinal)
		g, e := NewGenerator(b)
		if ordinal == 0 {
			if !errors.Is(e, arena.ErrInjectedFailure) || g != nil || b.Used() != 0 {
				t.Fatal(ordinal, g, e, b.Used())
			}
		} else {
			if e != nil || g.Stats().BackingBytes != 0 || b.Used() != g.Stats().OwnerBytes {
				t.Fatal(ordinal, g, e, b.Used())
			}
			if e = g.Close(); e != nil {
				t.Fatal(e)
			}
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
		}
	}
	b := arena.NewBudget(1 << 24)
	g, e := NewGenerator(b)
	if e != nil {
		t.Fatal(e)
	}
	b.FailAfter(0)
	_, e = g.At(191)
	st := g.Stats()
	if !errors.Is(e, ErrNontermination) || st.Count != 2016 || st.BackingBytes != 0 || b.Used() != st.OwnerBytes {
		t.Fatal("native cache and candidate storage", e, st, b.Used())
	}
	// Independent primality oracle checks every actually published prefix.
	for i := uint32(0); i < st.Count; i++ {
		p, err := g.At(i)
		if err != nil {
			t.Fatal(err)
		}
		for d := uint64(2); d <= p/d; d++ {
			if p%d == 0 {
				t.Fatal("published composite", i, p)
			}
		}
	}
	b.FailAfter(-1)
	if e = g.Close(); e != nil || b.Used() != 0 {
		t.Fatal(e, b.Used())
	}
}
func TestPrimeGeneratorWarmCachedReadAndFinalizeDomain(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	g, e := NewGenerator(b)
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
	var zero Generator
	if e = zero.Close(); !errors.Is(e, arena.ErrConfig) {
		t.Fatal(e)
	}
}
