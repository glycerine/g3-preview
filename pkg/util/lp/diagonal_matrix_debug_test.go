//go:build g3debug

package lp

import "testing"

func TestLPDiagonalTailDebugDenseView(t *testing.T) {
	f := NewFloatDiagonalMatrix(2, 4)
	f.SetDimension(4)
	if f.RowCount() != 4 || f.ColumnCount() != 4 {
		t.Fatal("debug dimensions")
	}
	p, err := PermutationFromValues(4, []uint32{2, 3, 0, 1})
	lpStaticMust(t, err)
	lpStaticMust(t, f.Conjugate(p))
	for i := uint32(0); i < 4; i++ {
		for j := uint32(0); j < 4; j++ {
			want := float64(0)
			if i == j {
				want = 1
			}
			if i == 0 && j == 0 {
				want = .25
			}
			if f.Element(i, j) != want {
				t.Fatalf("float view %d,%d", i, j)
			}
		}
	}
}
