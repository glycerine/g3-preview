package lp

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPExactSparseBasisIndependentOracle(t *testing.T) {
	rng := rand.New(rand.NewPCG(187, 719))
	type entry struct {
		row   uint32
		value *big.Rat
	}
	for world := 0; world < 120; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair("0", "0")
			n := uint32(1 + rng.IntN(5))
			a := NewRationalStaticMatrix(h.m, n, n+2)
			t.Cleanup(func() { h.must(a.Close()) })
			input := make([][]entry, n+2)
			for k := 0; k < 20; k++ {
				i, j := rng.Uint32N(n), rng.Uint32N(n+2)
				x := new(big.Rat).SetFrac(new(big.Int).Lsh(big.NewInt(int64(rng.IntN(19)-9)), uint(rng.IntN(140))), big.NewInt(int64(1+rng.IntN(31))))
				h.must(h.m.SetString(&p.X, x.RatString()))
				h.must(a.AddNewElement(i, j, &p.X))
				input[j] = append(input[j], entry{i, x})
			}
			basis := make([]uint32, n)
			for j := range basis {
				basis[j] = rng.Uint32N(n + 2)
			}
			m, err := NewRationalSparseBasisMatrix(a, basis)
			if m != nil {
				t.Cleanup(func() { h.must(m.Close()) })
			}
			h.must(err)
			var count uint32
			rowLengths := make([]uint32, n)
			var owned []*numeral.Rational
			for j := int(n) - 1; j >= 0; j-- {
				if len(m.Columns[j].Values) != len(input[basis[j]]) || m.Columns[j].ShortenedMarkovitz != 0 {
					t.Fatal("column size")
				}
				for co, x := range input[basis[j]] {
					c := &m.Columns[j].Values[co]
					if c.Index != x.row || c.Other != rowLengths[x.row] {
						t.Fatal("column ordering")
					}
					r := &m.Rows[x.row][c.Other]
					if r.Index != uint32(j) || r.Other != uint32(co) {
						t.Fatal("row ordering/backlink")
					}
					lpSparseRationalCheck(t, h, &c.Value, x.value)
					lpSparseRationalCheck(t, h, &r.Value, x.value)
					owned = append(owned, &c.Value, &r.Value)
					rowLengths[x.row]++
					count++
				}
			}
			if m.Dimension() != n || m.ActiveElements() != count || m.pivotQueue.Size() != 0 {
				t.Fatal("counts")
			}
			for i := range m.Rows {
				if len(m.Rows[i]) != int(rowLengths[i]) || m.WorkPivot[i] != -1 || m.Processed[i] {
					t.Fatal("initial workspace")
				}
			}
			// Named-cell census proves independent ownership, including every
			// source coefficient, both sparse directions and lexical inputs.
			for i := range a.Rows {
				for j := range a.Rows[i] {
					owned = append(owned, &a.Rows[i][j].Value)
				}
			}
			lpExactVectorNamedCensus(t, h, owned)
			rows, columns := make([]uint32, n), make([]uint32, n)
			for i := range rows {
				rows[i] = uint32(i)
				columns[i] = uint32(i)
			}
			for step := 0; step < 12; step++ {
				if n > 1 {
					x := rng.Uint32N(n)
					y := (x + 1 + rng.Uint32N(n-1)) % n
					if step%2 == 0 {
						h.must(m.SwapRows(x, y))
						rows[x], rows[y] = rows[y], rows[x]
					} else {
						h.must(m.SwapColumns(x, y))
						columns[x], columns[y] = columns[y], columns[x]
					}
				}
				for i := uint32(0); i < n; i++ {
					physical, e := m.AdjustRow(i)
					h.must(e)
					if physical != rows[i] {
						t.Fatal("row mapping")
					}
					inverse, e := m.AdjustRowInverse(physical)
					h.must(e)
					if inverse != i {
						t.Fatal("inverse row")
					}
					physical, e = m.AdjustColumn(i)
					h.must(e)
					if physical != columns[i] {
						t.Fatal("column mapping")
					}
					inverse, e = m.AdjustColumnInverse(physical)
					h.must(e)
					if inverse != i {
						t.Fatal("inverse column")
					}
					var lowest uint32
					for logicalRow, sourceRow := range rows {
						for _, x := range input[basis[columns[i]]] {
							if x.row == sourceRow && uint32(logicalRow) > lowest {
								lowest = uint32(logicalRow)
							}
						}
					}
					got, e := m.LowestRowInColumn(i)
					h.must(e)
					if got != lowest {
						t.Fatal("lowest", got, lowest)
					}
					for j := uint32(0); j < n; j++ {
						want := new(big.Rat)
						for _, x := range input[basis[columns[j]]] {
							if x.row == rows[i] {
								want = x.value
								break
							}
						}
						value, e := m.Get(i, j)
						h.must(e)
						lpSparseRationalCheck(t, h, value, want)
					}
				}
			}
			// Destroying the source does not invalidate either sparse owner.
			h.must(a.Close())
			for j := range m.Columns {
				for k, x := range input[basis[j]] {
					lpSparseRationalCheck(t, h, &m.Columns[j].Values[k].Value, x.value)
				}
			}
			h.must(m.Close())
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactSparseBasisAllocationFailureCensus(t *testing.T) {
	for _, mode := range []string{"single", "duplicate"} {
		failures, successes, columnOnly, receipts := 0, 0, 0, 0
		for ordinal := int64(0); ordinal < 180; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				text := strings.Repeat("1234567890", 100) + "/" + strings.Repeat("9876543211", 70)
				p := h.pair(text, "0")
				a := NewRationalStaticMatrix(h.m, 2, 3)
				t.Cleanup(func() { h.must(a.Close()) })
				h.must(a.AddNewElement(0, 2, &p.X))
				basis := []uint32{0, 2}
				if mode == "duplicate" {
					basis[0] = 2
					h.must(a.AddNewElement(1, 2, &p.X))
				}
				h.budget.FailAfter(ordinal)
				m, err := NewRationalSparseBasisMatrix(a, basis)
				h.budget.FailAfter(-1)
				if m == nil {
					t.Fatal("missing partial owner", err)
				}
				t.Cleanup(func() { h.must(m.Close()) })
				if err != nil {
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					failures++
					if m.calls.pending() {
						receipts++
					}
					rowCount, colCount := 0, 0
					for _, r := range m.Rows {
						rowCount += len(r)
					}
					for _, c := range m.Columns {
						colCount += len(c.Values)
					}
					if rowCount != int(m.active) || colCount < rowCount || colCount > rowCount+1 {
						t.Fatal("publication prefix", rowCount, colCount, m.active)
					}
					if colCount > rowCount {
						columnOnly++
					}
					for _, f := range []func() error{
						func() error { _, e := m.Get(0, 0); return e }, func() error { _, e := m.GetNotAdjusted(0, 0); return e },
						func() error { _, e := m.LowestRowInColumn(0); return e }, func() error { return m.SwapRows(0, 1) }, func() error { return m.SwapColumns(0, 1) },
					} {
						if !errors.Is(f(), ErrExactVectorPending) {
							t.Fatal("stopped owner guard")
						}
					}
				} else {
					successes++
					done = true
					want, _ := new(big.Rat).SetString(text)
					for i := range m.Rows {
						for j := range m.Rows[i] {
							lpSparseRationalCheck(t, h, &m.Rows[i][j].Value, want)
						}
					}
					for i := range m.Columns {
						for j := range m.Columns[i].Values {
							lpSparseRationalCheck(t, h, &m.Columns[i].Values[j].Value, want)
						}
					}
					expected := uint32(1)
					if mode == "duplicate" {
						expected = 4
					}
					if m.ActiveElements() != expected {
						t.Fatal("successful count")
					}
				}
				h.must(m.Close())
				if a.ready() != nil {
					t.Fatal("borrowed source stopped")
				}
				want, _ := new(big.Rat).SetString(text)
				lpSparseRationalCheck(t, h, &a.Rows[0][0].Value, want)
				h.must(a.Close())
				lpExactVectorNamedCensus(t, h, nil)
				if h.m.RawInvocationState().Active != [3]bool{} || h.m.ComparisonInvocationState().Active || h.m.DivisionInvocationState().Active {
					t.Fatal("orphan invocation")
				}
			})
			if done {
				break
			}
		}
		if failures == 0 || successes != 1 || columnOnly == 0 || receipts == 0 {
			t.Fatal("incomplete campaign", mode, failures, successes, columnOnly, receipts)
		}
	}
}

func TestLPExactSparseBasisBoundsAndBorrowedOwner(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("7/13", "0")
	a := NewRationalStaticMatrix(h.m, 2, 3)
	t.Cleanup(func() { h.must(a.Close()) })
	h.must(a.Set(0, 2, &p.X))
	for _, basis := range [][]uint32{{9, 2}, {2}} {
		m, e := NewRationalSparseBasisMatrix(a, basis)
		if !errors.Is(e, ErrBounds) || m == nil {
			t.Fatal(e)
		}
		h.must(m.Close())
	}
	m, e := NewRationalSparseBasisMatrix(a, []uint32{0, 2, 99})
	h.must(e)
	t.Cleanup(func() { h.must(m.Close()) })
	for _, f := range []func() error{
		func() error { _, e := m.Get(2, 0); return e }, func() error { _, e := m.Get(0, 2); return e }, func() error { _, e := m.GetNotAdjusted(2, 0); return e },
		func() error { _, e := m.LowestRowInColumn(2); return e }, func() error { return m.SwapRows(0, 2) }, func() error { return m.SwapColumns(2, 0) },
	} {
		if !errors.Is(f(), ErrBounds) {
			t.Fatal("missing bounds")
		}
	}
	z, e := m.GetNotAdjusted(0, 99)
	h.must(e)
	lpSparseRationalCheck(t, h, z, new(big.Rat))
	if v, e := m.LowestRowInColumn(0); e != nil || v != 0 {
		t.Fatal("empty column", v, e)
	}
	// Destroy numeric owners before independently corrupting borrowed metadata.
	h.must(m.Close())
	if _, e = m.GetNotAdjusted(0, 0); !errors.Is(e, ErrBounds) {
		t.Fatal(e)
	}
	if _, e = m.LowestRowInColumn(0); !errors.Is(e, ErrBounds) {
		t.Fatal(e)
	}
	m.Columns = make([]RationalSparseColumn, 2)
	m.Columns[0].Values = []RationalIndexedValue{{Value: numeral.NewRational(), Index: 9}}
	if _, e = m.LowestRowInColumn(0); !errors.Is(e, ErrBounds) {
		t.Fatal(e)
	}
	h.must(m.Close())
	for _, bad := range []ColumnCell{{2, 0}, {0, 99}} {
		a.Columns[2] = append(a.Columns[2][:1], bad)
		x, e := NewRationalSparseBasisMatrix(a, []uint32{0, 2})
		if !errors.Is(e, ErrBounds) || x.ActiveElements() != 1 {
			t.Fatal("copy prefix", e)
		}
		h.must(x.Close())
	}
	a.Columns[2] = a.Columns[2][:1]
	empty := NewRationalStaticMatrix(h.m, 0, 0)
	x, e := NewRationalSparseBasisMatrix(empty, nil)
	h.must(e)
	if x.Dimension() != 0 {
		t.Fatal("empty dimension")
	}
	h.must(x.Close())
	h.must(empty.Close())
	// A real failed source Set retains lexical ownership even if it failed
	// before acquiring a numeric receipt. Sparse construction must reject it.
	large := h.pair(strings.Repeat("1234567890", 100)+"/7", "0")
	h.budget.FailAfter(0)
	e = a.Set(1, 0, &large.X)
	h.budget.FailAfter(-1)
	if !lpExactPriorityPhysicalRefusal(e) {
		t.Fatal("no actual source stop", e)
	}
	if x, e = NewRationalSparseBasisMatrix(a, []uint32{0, 2}); x != nil || e != ErrExactVectorPending {
		t.Fatal("borrowed source guard", x, e)
	}
	h.must(a.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactSparseBasisLeavesOlderComparisonOwned(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
			z := h.pair("3/7", "0")
			a := NewRationalStaticMatrix(h.m, 1, 2)
			h.must(a.Set(0, 0, &z.X))
			h.must(a.Set(0, 1, &p.X))
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
				m, e := NewRationalSparseBasisMatrix(a, []uint32{1})
				h.must(e)
				value, e := m.Get(0, 0)
				h.must(e)
				want, _ := new(big.Rat).SetString(strings.Repeat("1234567890", 100) + "/7")
				lpSparseRationalCheck(t, h, value, want)
				h.must(m.Close())
				h.must(a.Close())
				if h.m.ComparisonInvocationState() != older {
					t.Fatal("stole comparison ownership")
				}
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			h.must(a.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no physical stopped comparison")
	}
}
