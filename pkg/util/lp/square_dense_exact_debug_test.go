//go:build g3debug

package lp

import (
	"testing"
)

func TestLPExactDenseBlockDebugLChecks(t *testing.T) {
	for _, kind := range []string{"valid", "row-permutation", "prefix-column", "upper", "diagonal", "data", "column-map", "prefix-map"} {
		t.Run(kind, func(t *testing.T) {
			h := newLPPairHarness(t)
			parent := lpNewExactSparse(t, h, 3)
			m, err := NewRationalSquareDenseSubmatrix(parent, 1)
			h.must(err)
			for i, x := range []int64{2, 0, 3, 4} {
				h.must(h.m.SetInt64(&m.Data[i], x))
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
}
