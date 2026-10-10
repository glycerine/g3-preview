package lp

import (
	"fmt"
	"math/big"
	"testing"
)

func lpExactDenseFixture(t *testing.T, h *lpPairHarness, start uint32) *RationalSquareDenseSubmatrix {
	t.Helper()
	parent := lpNewExactSparse(t, h, 2)
	p := h.pair("2", "0")
	for i := start; i < 2; i++ {
		for j := start; j < 2; j++ {
			h.must(h.m.SetInt64(&p.X, int64(2+i*4+j*2)))
			h.must(parent.AddNewElement(i, j, &p.X))
		}
	}
	m, err := NewRationalSquareDenseSubmatrix(parent, start)
	h.must(err)
	return m
}

func TestLPExactDenseBlockBoundsAndPublication(t *testing.T) {
	for _, kind := range []string{"start", "row-map", "rows", "column-map", "earlier-column"} {
		t.Run(kind, func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair("7", "0")
			parent := lpNewExactSparse(t, h, 2)
			h.must(parent.AddNewElement(1, 1, &p.X))
			start := uint32(1)
			switch kind {
			case "start":
				start = 3
			case "row-map":
				parent.RowPermutation.values = nil
			case "rows":
				parent.Rows = parent.Rows[:1] // discarded coefficient is inline
			case "column-map":
				parent.ColumnPermutation.values = nil
			case "earlier-column":
				h.must(parent.AddNewElement(1, 0, &p.X))
			}
			m, err := NewRationalSquareDenseSubmatrix(parent, start)
			want := ErrBounds
			if kind == "earlier-column" && sourceDebugTests {
				want = ErrAssertion
			}
			if err != want {
				t.Fatalf("constructor %v want %v", err, want)
			}
			if kind == "earlier-column" {
				lpSparseRationalCheck(t, h, &m.Data[0], big.NewRat(7, 1))
			}
			h.must(m.Close())
			h.must(parent.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	for _, kind := range []string{"find", "divide", "row", "pivot", "self", "zero-divide", "zero-row", "cell"} {
		t.Run(kind, func(t *testing.T) {
			h := newLPPairHarness(t)
			m := lpExactDenseFixture(t, h, 0)
			var err error
			switch kind {
			case "find":
				m.Data = m.Data[:0]
				_, err = m.FindPivotColumn(0)
			case "divide":
				m.Data = m.Data[:0]
				err = m.DivideRowByPivot(0)
			case "row":
				m.Data = m.Data[:2]
				err = m.PivotRowToRow(0, 1, nil)
			case "pivot":
				m.Data = m.Data[:3]
				err = m.Pivot(0, nil)
			case "self":
				err = m.PivotRowToRow(0, 0, nil)
			case "zero-divide":
				h.must(h.m.SetInt64(&m.Data[0], 0))
				err = m.DivideRowByPivot(0)
			case "zero-row":
				h.must(h.m.SetInt64(&m.Data[0], 0))
				err = m.PivotRowToRow(0, 1, nil)
			case "cell":
				_, err = m.cell(2, 0)
			}
			want := ErrBounds
			if kind == "self" || kind == "zero-row" || kind == "zero-divide" {
				want = nil
				if sourceDebugTests {
					want = ErrAssertion
				}
			}
			if kind == "cell" && sourceDebugTests {
				want = ErrAssertion
			}
			if err != want {
				t.Fatalf("operation %v want %v", err, want)
			}
			// Source release omits the nonzero assertion. mpq normalization
			// leaves denominator zero; cleanup must still destroy that storage.
			if kind == "zero-divide" && !sourceDebugTests {
				for i := 0; i < 2; i++ {
					if m.Data[i].Num.Val != 1 || m.Data[i].Den.Val != 0 {
						t.Fatal("release zero division representation", i, m.Data[i])
					}
				}
			}
			if kind == "pivot" {
				for i, want := range []*big.Rat{big.NewRat(1, 2), big.NewRat(2, 1), big.NewRat(-3, 1)} {
					lpSparseRationalCheck(t, h, &m.Data[i], want)
				}
			}
			if kind == "self" && !sourceDebugTests {
				lpSparseRationalCheck(t, h, &m.Data[0], big.NewRat(-4, 1))
				lpSparseRationalCheck(t, h, &m.Data[1], big.NewRat(-4, 1))
			}
			h.must(m.Close())
			h.must(m.Parent.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	for _, kind := range []string{"find", "cell"} {
		t.Run("below-start/"+kind, func(t *testing.T) {
			h := newLPPairHarness(t)
			m := lpExactDenseFixture(t, h, 1)
			var err error
			if kind == "find" {
				_, err = m.FindPivotColumn(0)
			} else {
				_, err = m.cell(0, 1)
			}
			want := ErrBounds
			if sourceDebugTests {
				want = ErrAssertion
			}
			if err != want {
				t.Fatalf("below start %v want %v", err, want)
			}
			h.must(m.Close())
			h.must(m.Parent.Close())
		})
	}
	// A stopped parent refuses construction before acquiring a child owner.
	h := newLPPairHarness(t)
	parent := lpNewExactSparse(t, h, 1)
	p := h.pair("1", "0")
	if err := parent.AddNewElement(2, 0, &p.X); err != ErrBounds {
		t.Fatal(err)
	}
	if m, err := NewRationalSquareDenseSubmatrix(parent, 0); m != nil || err != ErrExactVectorPending {
		t.Fatal("stopped parent", m, err)
	}
	h.must(parent.Close())
	// Permutation errors leave numerical storage and ownership unchanged.
	for _, kind := range []string{"swap", "row", "column", "cell"} {
		t.Run(fmt.Sprint("map/", kind), func(t *testing.T) {
			h := newLPPairHarness(t)
			m := lpExactDenseFixture(t, h, 0)
			var err error
			switch kind {
			case "swap":
				err = m.SwapColumns(0, 2)
			case "row":
				m.RowPermutation.values = nil
				_, err = m.AdjustRow(0)
			case "column":
				m.ColumnPermutation.values = nil
				_, err = m.AdjustColumnInverse(0)
			case "cell":
				m.ColumnPermutation.reverse[0] = 4
				_, err = m.cell(0, 0)
			}
			if err != ErrBounds {
				t.Fatal("permutation guard", err)
			}
			h.must(m.Close())
			h.must(m.Parent.Close())
		})
	}
}
