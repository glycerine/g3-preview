//go:build g3debug

package lp

import "testing"

func TestLPDriverPermutationDebugActualProductChecks(t *testing.T) {
	original := []uint32{4, 2, 0, 3, 1}
	for _, values := range lpPermutations(5) {
		l, err := FloatPermutationFromValues(5, values)
		if err != nil {
			t.Fatal(err)
		}
		// Construct the expected composition using an independent matrix oracle.
		product := lpMatrixProduct(lpPermutationMatrix(original), lpMatrixTranspose(lpPermutationMatrix(values)))
		mapping := make([]uint32, 5)
		for i, row := range product {
			for j, value := range row {
				if value == 1 {
					mapping[i] = uint32(j)
				}
			}
		}
		p, err := FloatPermutationFromValues(5, mapping)
		if err != nil {
			t.Fatal(err)
		}
		if err = driverCheckPermutationProduct(p, l, original); err != nil {
			t.Fatal(values, err)
		}
		if err = p.TransposeRight(0, 1); err != nil {
			t.Fatal(err)
		}
		if err = driverCheckPermutationProduct(p, l, original); err != ErrAssertion {
			t.Fatal(values, err)
		}
	}
	l := NewFloatPermutation(5)
	if err := driverCheckPermutationProduct(l, l, nil); err != ErrBounds {
		t.Fatal(err)
	}
	if err := driverCheckPermutationProduct(l, NewFloatPermutation(4), original); err != ErrAssertion {
		t.Fatal(err)
	}
	if err := driverCheckPermutationProduct(NewFloatPermutation(4), l, original); err != ErrAssertion {
		t.Fatal(err)
	}
}
