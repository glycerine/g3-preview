package spacer

// Source: src/muz/spacer/spacer_matrix.cpp normalization and reached Gaussian
// elimination assignment/subtraction order.

import (
	"math/big"
	"testing"
)

func TestSpacerMatrixNormalizeAndSourceEliminationOrder(t *testing.T) {
	normalized := newSpacerMatrix(1, 2)
	if err := normalized.set(0, 0, big.NewRat(1, 2)); err != nil {
		t.Fatal(err)
	}
	if err := normalized.set(0, 1, big.NewRat(1, 3)); err != nil {
		t.Fatal(err)
	}
	if err := normalized.normalize(); err != nil {
		t.Fatal(err)
	}
	first, _ := normalized.get(0, 0)
	second, _ := normalized.get(0, 1)
	if first.Cmp(big.NewRat(3, 1)) != 0 || second.Cmp(big.NewRat(2, 1)) != 0 {
		t.Fatal("normalized row", first, second)
	}

	ordered := newSpacerMatrix(2, 2)
	_ = ordered.set(0, 0, big.NewRat(1, 1))
	_ = ordered.set(0, 1, big.NewRat(2, 1))
	_ = ordered.set(1, 1, big.NewRat(1, 1))
	rank, err := ordered.gaussianElimination()
	if err != nil || rank != 2 {
		t.Fatal("ordered rank", rank, err)
	}
	upper, _ := ordered.get(0, 1)
	if upper.Cmp(big.NewRat(2, 1)) != 0 {
		t.Fatal("source subtraction unexpectedly revisited row zero", upper)
	}

	swapped := newSpacerMatrix(2, 2)
	_ = swapped.set(0, 1, big.NewRat(1, 1))
	_ = swapped.set(1, 0, big.NewRat(2, 1))
	_ = swapped.set(1, 1, big.NewRat(3, 1))
	rank, err = swapped.gaussianElimination()
	if err != nil || rank != 1 {
		t.Fatal("source row-reorder assignment parity", rank, err)
	}
}
