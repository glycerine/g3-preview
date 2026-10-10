//go:build g3debug

package lp

import (
	"fmt"
	"math/big"
	"testing"
)

func TestLPExactDenseBlockDebugViewAndLChecks(t *testing.T) {
	for _, kind := range []string{"valid", "row-permutation", "prefix-column", "upper", "diagonal", "data", "column-map", "prefix-map"} {
		t.Run(kind, func(t *testing.T) {
			h := newLPPairHarness(t)
			parent := lpNewExactSparse(t, h, 3)
			m, err := NewRationalSquareDenseSubmatrix(parent, 1)
			h.must(err)
			for i, x := range []int64{2, 0, 3, 4} {
				h.must(h.m.SetInt64(&m.Data[i], x))
			}
			out := h.pair("0", "0")
			if kind == "valid" {
				if m.RowCount() != 3 || m.ColumnCount() != 3 {
					t.Fatal("dimensions")
				}
				want := [][]int64{{1, 0, 0}, {0, 2, 0}, {0, 3, 4}}
				for i := uint32(0); i < 3; i++ {
					for j := uint32(0); j < 3; j++ {
						h.must(m.Element(i, j, &out.X))
						lpSparseRationalCheck(t, h, &out.X, big.NewRat(want[i][j], 1))
					}
				}
			}
			want := ErrAssertion
			switch kind {
			case "row-permutation":
				h.must(m.RowPermutation.TransposeLeft(0, 1))
			case "prefix-column":
				h.must(m.ColumnPermutation.TransposeLeft(0, 1))
			case "upper":
				h.must(h.m.SetInt64(&m.Data[1], 1))
			case "diagonal":
				h.must(h.m.SetInt64(&m.Data[0], 0))
			case "data":
				m.Data = nil
				want = ErrBounds
			case "column-map":
				m.ColumnPermutation.values = m.ColumnPermutation.values[:1]
				want = ErrBounds
			case "prefix-map":
				m.ColumnPermutation.values = nil
				want = ErrBounds
			case "valid":
				want = nil
			}
			ok, err := m.IsLMatrix()
			if err != want || ok != (want == nil) {
				t.Fatal("L diagnostic", ok, err, want)
			}
			h.must(m.Close())
			h.must(parent.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	for _, kind := range []string{"row-map", "column-map", "data"} {
		t.Run(fmt.Sprint("view/", kind), func(t *testing.T) {
			h := newLPPairHarness(t)
			m := lpExactDenseFixture(t, h, 0)
			out := h.pair("0", "0")
			switch kind {
			case "row-map":
				m.RowPermutation.values = nil
			case "column-map":
				m.ColumnPermutation.reverse[0] = 4
			case "data":
				m.Data = nil
			}
			if err := m.Element(0, 0, &out.X); err != ErrBounds {
				t.Fatal(err)
			}
			h.must(m.Close())
			h.must(m.Parent.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactDenseBlockDebugViewAllocationFailure(t *testing.T) {
	failures, successes := 0, 0
	for ordinal := int64(0); ordinal < 100; ordinal++ {
		done := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := lpNewDirectSparseHarness(t)
			m := lpExactDenseFixture(t, h, 0)
			text := "123456789012345678901234567890123456789012345678901234567890/13"
			h.must(h.m.SetString(&m.Data[0], text))
			out := h.pair("0", "0")
			h.budget.FailAfter(ordinal)
			err := m.Element(0, 0, &out.X)
			h.budget.FailAfter(-1)
			if err != nil {
				failures++
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				if e := m.Element(0, 0, &out.X); e != ErrExactVectorPending {
					t.Fatal("stopped view", e)
				}
			} else {
				done = true
				successes++
				want, _ := new(big.Rat).SetString(text)
				lpSparseRationalCheck(t, h, &out.X, want)
			}
			h.must(m.Close())
			h.must(m.Parent.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
		if done {
			break
		}
	}
	if failures == 0 || successes != 1 {
		t.Fatal("incomplete physical view campaign", failures, successes)
	}
}
