//go:build g3debug

package lp

import (
	"testing"
)

func TestLPFloatDenseBlockDebugViewAndLChecks(t *testing.T) {
	m := lpFloatBlockFixture(t, 1)
	m.Data = []float64{2, 0, 3, 4}
	if m.RowCount() != 3 || m.ColumnCount() != 3 {
		t.Fatal("parent dimensions")
	}
	want := [][]float64{{1, 0, 0}, {0, 2, 0}, {0, 3, 4}}
	for i := uint32(0); i < 3; i++ {
		for j := uint32(0); j < 3; j++ {
			v, err := m.Element(i, j)
			lpStaticMust(t, err)
			if v != want[i][j] {
				t.Fatal("debug view", i, j, v)
			}
		}
	}
	for _, kind := range []string{"row-permutation", "prefix-column", "upper", "diagonal", "data", "column-map"} {
		t.Run(kind, func(t *testing.T) {
			x := lpFloatBlockFixture(t, 1)
			x.Data = []float64{2, 0, 3, 4}
			want := ErrAssertion
			switch kind {
			case "row-permutation":
				lpStaticMust(t, x.RowPermutation.TransposeLeft(0, 1))
			case "prefix-column":
				lpStaticMust(t, x.ColumnPermutation.TransposeLeft(0, 1))
			case "upper":
				x.Data[1] = 1
			case "diagonal":
				x.Data[0] = 0
			case "data":
				x.Data = nil
				want = ErrBounds
			case "column-map":
				x.ColumnPermutation.values = nil
				want = ErrBounds
			}
			if ok, err := x.IsLMatrix(); ok || err != want {
				t.Fatalf("L diagnostic %v %v", ok, err)
			}
		})
	}
	m.RowPermutation.values = nil
	if _, err := m.Element(0, 0); err != ErrBounds {
		t.Fatal("view row mapping guard", err)
	}
	m = lpFloatBlockFixture(t, 1)
	m.Data = nil
	if _, err := m.Element(1, 1); err != ErrBounds {
		t.Fatal("view coefficient guard", err)
	}
}
