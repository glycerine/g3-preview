package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"
)

type lpRationalIteratorTerm struct {
	index uint32
	value *big.Rat
}

func TestLPRationalIteratorsIndependentCursorOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	h := newLPPairHarness(t)
	xtext := strings.Repeat("1234567890", 30) + "/7"
	ytext := strings.Repeat("9876543210", 30) + "/13"
	p := h.pair(xtext, ytext)
	z := h.pair("0", "-7/11")
	out := h.pair("0", "0")
	m := NewRationalStaticMatrix(h.m, 4, 5)
	v := NewIndexedRationalVector(h.m, 6)
	h.must(m.Set(2, 3, &p.Y))
	h.must(m.Set(0, 3, &p.X))
	h.must(m.AddNewElement(3, 3, &z.X))
	h.must(m.Set(0, 1, &z.Y))
	h.must(m.AddNewElement(0, 4, &z.X))
	h.must(h.m.SetInt64(&z.X, 3))
	h.must(m.Set(0, 3, &z.X))
	h.must(h.m.Copy(&v.Data[4], &p.X))
	h.must(h.m.Copy(&v.Data[1], &z.Y))
	v.Index = []uint32{4, 1, 4, 2}
	row, e := NewRationalRowIterator(m, 0)
	h.must(e)
	col, e := NewRationalColumnIterator(m, 3)
	h.must(e)
	indexed := NewRationalIndexedIterator(v)
	pivot := NewRationalPivotRowIterator(v, 4)
	iterators := []RationalLinearIterator{row, col, indexed, pivot}
	t.Cleanup(func() {
		for _, it := range iterators {
			h.must(it.Close())
		}
		h.must(v.Close())
		h.must(m.Close())
	})
	x, _ := new(big.Rat).SetString(xtext)
	y, _ := new(big.Rat).SetString(ytext)
	expected := [][]lpRationalIteratorTerm{
		{{3, x}, {1, big.NewRat(-7, 11)}, {4, new(big.Rat)}, {3, big.NewRat(3, 1)}},
		{{2, y}, {0, x}, {3, new(big.Rat)}, {0, big.NewRat(3, 1)}},
		{{4, x}, {1, big.NewRat(-7, 11)}, {4, x}, {2, new(big.Rat)}},
		{{4, big.NewRat(1, 1)}, {4, x}, {1, big.NewRat(-7, 11)}, {4, x}, {2, new(big.Rat)}},
	}
	r := rand.New(rand.NewPCG(1931, 541))
	for n, it := range iterators {
		pos := 0
		for step := 0; step < 1200; step++ {
			switch r.IntN(4) {
			case 0:
				h.must(it.Reset())
				pos = 0
			case 1:
				index, ok, e := it.Next(&out.X)
				h.must(e)
				if ok != (pos < len(expected[n])) {
					t.Fatal("Next end", n, pos, ok)
				}
				if ok {
					if index != expected[n][pos].index {
						t.Fatal("Next index")
					}
					lpSparseRationalCheck(t, h, &out.X, expected[n][pos].value)
					pos++
				}
			case 2:
				before, e := h.m.String(&out.X)
				h.must(e)
				index, ok, e := it.NextIndex()
				h.must(e)
				if ok != (pos < len(expected[n])) {
					t.Fatal("NextIndex end")
				}
				if ok {
					if index != expected[n][pos].index {
						t.Fatal("NextIndex value")
					}
					pos++
				}
				after, e := h.m.String(&out.X)
				h.must(e)
				if after != before {
					t.Fatal("index-only copy")
				}
			case 3:
				size := len(expected[n])
				if n == 3 {
					size--
				}
				if it.Size() != uint32(size) {
					t.Fatal("literal size", n, it.Size(), size)
				}
			}
		}
		h.must(it.Reset())
		for _, term := range expected[n] {
			index, ok, e := it.Next(&out.X)
			h.must(e)
			if !ok || index != term.index {
				t.Fatal("drain")
			}
			lpSparseRationalCheck(t, h, &out.X, term.value)
		}
		for k := 0; k < 3; k++ {
			_, ok, e := it.Next(&out.X)
			h.must(e)
			if ok {
				t.Fatal("repeat end")
			}
		}
	}
	if row.offset != 4 || indexed.offset != 4 || col.offset != 6 || pivot.inner.offset != 4 {
		t.Fatal("cursor exhaustion", row.offset, indexed.offset, col.offset, pivot.inner.offset)
	}
	// Existing container objects remain borrowed, rather than snapshotted.
	h.must(h.m.SetInt64(&v.Data[5], 19))
	v.Index[0] = 5
	h.must(indexed.Reset())
	index, ok, e := indexed.Next(&out.X)
	h.must(e)
	if !ok || index != 5 {
		t.Fatal("borrowed index")
	}
	lpSparseRationalCheck(t, h, &out.X, big.NewRat(19, 1))
	for _, it := range iterators {
		h.must(it.Close())
	}
	h.must(v.Close())
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPRationalIteratorsCopyFailureOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, kind := range []string{"row", "column", "indexed", "pivot"} {
		failures, successes := 0, 0
		for ordinal := int64(0); ordinal < 60; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("%s/%d", kind, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				text := strings.Repeat("1234567890", 100) + "/7"
				p := h.pair(text, "0")
				out := h.pair("0", "0")
				m := NewRationalStaticMatrix(h.m, 4, 4)
				v := NewIndexedRationalVector(h.m, 5)
				h.must(m.Set(1, 2, &p.X))
				h.must(m.Set(3, 1, &p.X))
				h.must(v.SetValue(&p.X, 4))
				var it RationalLinearIterator
				var e error
				switch kind {
				case "row":
					it, e = NewRationalRowIterator(m, 1)
				case "column":
					it, e = NewRationalColumnIterator(m, 1)
				case "indexed":
					it = NewRationalIndexedIterator(v)
				case "pivot":
					it = NewRationalPivotRowIterator(v, 0)
					index, ok, err := it.Next(&out.X)
					h.must(err)
					if !ok || index != 0 {
						t.Fatal("unit basis")
					}
				}
				h.must(e)
				t.Cleanup(func() { h.budget.FailAfter(-1); h.must(it.Close()); h.must(v.Close()); h.must(m.Close()) })
				h.budget.FailAfter(ordinal)
				index, ok, err := it.Next(&out.X)
				h.budget.FailAfter(-1)
				if err == nil {
					successes++
					done = true
					wantIndex := uint32(4)
					if kind == "row" {
						wantIndex = 2
					}
					if kind == "column" {
						wantIndex = 3
					}
					if !ok || index != wantIndex {
						t.Fatal("success", index, ok)
					}
					want, _ := new(big.Rat).SetString(text)
					lpSparseRationalCheck(t, h, &out.X, want)
				} else {
					if !lpExactPriorityPhysicalRefusal(err) || ok {
						t.Fatal(err, ok)
					}
					failures++
					// Row/indexed publish index before copy; column publishes it after.
					wantIndex := uint32(4)
					if kind == "row" {
						wantIndex = 2
					}
					if kind == "column" {
						wantIndex = 0
					}
					if index != wantIndex {
						t.Fatal("copy index publication", index, wantIndex)
					}
					if m.ready() != nil || v.calls.pending() {
						t.Fatal("source owner adopted iterator copy")
					}
					pending := false
					switch q := it.(type) {
					case *RationalRowIterator:
						pending = q.calls.pending()
						if q.offset != 1 {
							t.Fatal("row cursor")
						}
					case *RationalColumnIterator:
						pending = q.calls.pending()
						if q.offset != 0 {
							t.Fatal("column cursor")
						}
					case *RationalIndexedIterator:
						pending = q.calls.pending()
						if q.offset != 1 {
							t.Fatal("indexed cursor")
						}
					case *RationalPivotRowIterator:
						pending = q.inner.calls.pending()
						if q.inner.offset != 1 || q.calls.pending() {
							t.Fatal("pivot child ownership")
						}
					}
					if pending {
						if e := it.Reset(); e != ErrExactVectorPending {
							t.Fatal(e)
						}
						if _, _, e := it.Next(&out.X); e != ErrExactVectorPending {
							t.Fatal(e)
						}
						if _, _, e := it.NextIndex(); e != ErrExactVectorPending {
							t.Fatal(e)
						}
					}
				}
				h.must(it.Close())
				h.must(it.Close())
				h.must(v.Close())
				h.must(m.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
			if done {
				break
			}
		}
		if failures == 0 || successes != 1 {
			t.Fatal("incomplete copy campaign", kind, failures, successes)
		}
	}
}

func TestLPRationalIteratorsLiveBoundsAndLiteralIndexPaths(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("2", "3")
	out := h.pair("99", "100")
	m := NewRationalStaticMatrix(h.m, 1, 2)
	v := NewIndexedRationalVector(h.m, 1)
	h.must(m.Set(0, 0, &p.X))
	h.must(m.Set(0, 1, &p.Y))
	row, e := NewRationalRowIterator(m, 0)
	h.must(e)
	col, e := NewRationalColumnIterator(m, 0)
	h.must(e)
	ix := NewRationalIndexedIterator(v)
	pivot := NewRationalPivotRowIterator(v, 7)
	t.Cleanup(func() {
		h.must(row.Close())
		h.must(col.Close())
		h.must(ix.Close())
		h.must(pivot.Close())
		h.must(v.Close())
		h.must(m.Close())
	})
	if _, e := NewRationalRowIterator(m, 7); e != ErrBounds {
		t.Fatal(e)
	}
	if _, e := NewRationalColumnIterator(m, 7); e != ErrBounds {
		t.Fatal(e)
	}
	// Row equality-at-end differs from indexed >= exhaustion. Shrinking this
	// same borrowed row after advancing reaches the source row's bounds guard.
	_, _, e = row.NextIndex()
	h.must(e)
	_, _, e = row.NextIndex()
	h.must(e)
	h.must(m.RemoveElement(0, 1))
	if _, _, e = row.Next(&out.X); e != ErrBounds {
		t.Fatal(e)
	}
	if _, _, e = row.NextIndex(); e != ErrBounds {
		t.Fatal(e)
	}
	h.must(row.Reset())
	// The column index-only path does not dereference the row link.
	m.Columns[0][0].Offset = 7
	index, ok, e := col.NextIndex()
	h.must(e)
	if !ok || index != 0 {
		t.Fatal(index, ok)
	}
	h.must(col.Reset())
	if _, _, e = col.Next(&out.X); e != ErrBounds {
		t.Fatal(e)
	}
	v.Index = []uint32{7}
	index, ok, e = ix.Next(&out.X)
	if index != 7 || ok || e != ErrBounds {
		t.Fatal(index, ok, e)
	}
	h.must(ix.Reset())
	index, ok, e = ix.NextIndex()
	h.must(e)
	if !ok || index != 7 {
		t.Fatal(index, ok)
	}
	// Pivot's unit index is not bounded by the borrowed dense vector extent.
	index, ok, e = pivot.NextIndex()
	h.must(e)
	if !ok || index != 7 || pivot.Size() != 1 {
		t.Fatal(index, ok, pivot.Size())
	}
	h.must(pivot.Reset())
	index, ok, e = pivot.Next(&out.X)
	h.must(e)
	if !ok || index != 7 {
		t.Fatal(index, ok)
	}
	lpSparseRationalCheck(t, h, &out.X, big.NewRat(1, 1))
	// Signed column counter overflow is an explicitly guarded invalid domain.
	col.offset = 1<<31 - 1
	if _, _, e = col.Next(&out.X); e != ErrBounds {
		t.Fatal(e)
	}
	col.offset = 1<<31 - 1
	if _, _, e = col.NextIndex(); e != ErrBounds {
		t.Fatal(e)
	}
	for _, it := range []RationalLinearIterator{row, col, ix, pivot} {
		h.must(it.Close())
	}
	h.must(v.Close())
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPRationalIteratorsAndScopesLeaveOlderComparisonOwned(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
			z := h.pair("1", "2")
			out := h.pair("0", "0")
			m := NewRationalStaticMatrix(h.m, 2, 2)
			v := NewIndexedRationalVector(h.m, 1)
			h.must(m.Set(0, 0, &z.X))
			h.must(m.Set(1, 1, &z.Y))
			h.must(v.SetValue(&z.Y, 0))
			row, e := NewRationalRowIterator(m, 0)
			h.must(e)
			column, e := NewRationalColumnIterator(m, 0)
			h.must(e)
			iterators := []RationalLinearIterator{row, column, NewRationalIndexedIterator(v), NewRationalPivotRowIterator(v, 7)}
			t.Cleanup(func() {
				for _, it := range iterators {
					h.must(it.Close())
				}
				h.must(v.Close())
				h.must(m.Close())
			})
			var caller exactPriorityCalls
			b, e := caller.begin(h.m)
			h.must(e)
			h.budget.FailAfter(ordinal)
			_, e = h.m.Compare(&p.X, &p.Y)
			h.budget.FailAfter(-1)
			e = caller.after(h.m, b, e)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				if !lpExactPriorityPhysicalRefusal(e) {
					t.Fatal(e)
				}
				matched = true
				h.must(m.Push())
				h.must(m.TransposeRows(0, 1))
				h.must(m.Pop(0))
				h.must(m.Pop(1))
				for n, it := range iterators {
					_, ok, e := it.Next(&out.X)
					h.must(e)
					if !ok {
						t.Fatal("inline iterator")
					}
					want := int64(2)
					if n == 1 || n == 3 {
						want = 1
					}
					lpSparseRationalCheck(t, h, &out.X, big.NewRat(want, 1))
					h.must(it.Close())
				}
				h.must(m.Close())
				h.must(v.Close())
				if h.m.ComparisonInvocationState() != older {
					t.Fatal("consumed older comparison")
				}
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			for _, it := range iterators {
				h.must(it.Close())
			}
			h.must(m.Close())
			h.must(v.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no real stopped comparison")
	}
}

func TestLPRationalIteratorIndexOnlyNoNumeralReservations(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	h := newLPPairHarness(t)
	p := h.pair(strings.Repeat("9876543210", 100)+"/13", "0")
	m := NewRationalStaticMatrix(h.m, 1, 1)
	v := NewIndexedRationalVector(h.m, 1)
	h.must(m.Set(0, 0, &p.X))
	h.must(v.SetValue(&p.X, 0))
	row, e := NewRationalRowIterator(m, 0)
	h.must(e)
	column, e := NewRationalColumnIterator(m, 0)
	h.must(e)
	its := []RationalLinearIterator{row, column, NewRationalIndexedIterator(v), NewRationalPivotRowIterator(v, 0)}
	before := h.budget.Used()
	h.budget.FailAfter(0)
	for n, it := range its {
		h.must(it.Reset())
		count := 0
		for {
			_, ok, e := it.NextIndex()
			h.must(e)
			if !ok {
				break
			}
			count++
		}
		want := 1
		if n == 3 {
			want = 2
		}
		if count != want {
			t.Fatal(count, want)
		}
	}
	h.budget.FailAfter(-1)
	if h.budget.Used() != before {
		t.Fatal("index-only numeric allocation")
	}
	for _, it := range its {
		h.must(it.Close())
	}
	h.must(v.Close())
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPRationalIteratorConstructorsRejectStoppedMatrix(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("2", "0")
	m := NewRationalStaticMatrix(h.m, 1, 1)
	if e := m.Set(7, 0, &p.X); e == nil {
		t.Fatal("expected source bounds stop")
	}
	if _, e := NewRationalRowIterator(m, 0); e != ErrExactVectorPending {
		t.Fatal(e)
	}
	if _, e := NewRationalColumnIterator(m, 0); e != ErrExactVectorPending {
		t.Fatal(e)
	}
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}
