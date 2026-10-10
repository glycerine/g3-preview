//go:build g3debug

package lp

import (
	"math/big"
	"testing"
)

func TestLPDiagonalTailDebugDenseView(t *testing.T) {
	h := newLPPairHarness(t)
	d := h.pair("123456789123456789123456789/19", "0")
	m, err := NewRationalDiagonalMatrix(h.m, 2, &d.X)
	h.must(err)
	f := NewFloatDiagonalMatrix(2, 4)
	m.SetDimension(4)
	f.SetDimension(4)
	if m.RowCount() != 4 || m.ColumnCount() != 4 || f.RowCount() != 4 || f.ColumnCount() != 4 {
		t.Fatal("debug dimensions")
	}
	p, err := PermutationFromValues(4, []uint32{2, 3, 0, 1})
	h.must(err)
	h.must(m.Conjugate(p))
	lpStaticMust(t, f.Conjugate(p))
	want, _ := new(big.Rat).SetString("19/123456789123456789123456789")
	out := h.pair("0", "0")
	for i := uint32(0); i < 4; i++ {
		for j := uint32(0); j < 4; j++ {
			w := new(big.Rat)
			fw := float64(0)
			if i == j {
				w.SetInt64(1)
				fw = 1
			}
			if i == 0 && j == 0 {
				w.Set(want)
				fw = .25
			}
			h.must(m.Element(i, j, &out.X))
			lpSparseRationalCheck(t, h, &out.X, w)
			if f.Element(i, j) != fw {
				t.Fatalf("float view %d,%d", i, j)
			}
		}
	}
	if err := m.ApplyLeft(nil); err != ErrBounds {
		t.Fatalf("stop debug owner: %v", err)
	}
	if err := m.Element(0, 0, &out.X); err != ErrExactVectorPending {
		t.Fatalf("stopped debug view: %v", err)
	}
	h.must(m.Close())
}
