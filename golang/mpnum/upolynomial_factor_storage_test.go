// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestUPolynomialBerlekampPivotSourceResizeConstructor(t *testing.T) {
	for _, tc := range []struct{ n, capacity uint32 }{{0, 0}, {1, 2}, {2, 2}, {3, 3}, {4, 5}, {6, 8}, {70, 93}} {
		b := arena.NewBudget(1024)
		var values []int32
		if e := upolyFactorInt32Resize(&values, tc.n, -1, b); e != nil {
			t.Fatal(tc, e)
		}
		if uint32(cap(values)) != tc.capacity || b.Used() != uint64(tc.capacity)*4 {
			t.Fatal("source vector(SZ,elem) resize growth", tc, cap(values), b.Used())
		}
		for i := uint32(0); i < tc.n; i++ {
			if values[i] != -1 {
				t.Fatal(i, values[i])
			}
		}
		b.Release(uint64(cap(values)) * 4)
	}
	b := arena.NewBudget(1024)
	b.FailAfter(2)
	var values []int32
	if e := upolyFactorInt32Resize(&values, 4, -1, b); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if len(values) != 0 || cap(values) != 3 || b.Used() != 12 {
		t.Fatal("third growth fails before source logical/default publication", len(values), cap(values), b.Used())
	}
	for _, v := range values {
		if v != 0 {
			t.Fatal("fill must wait until all capacity expansions", v)
		}
	}
	b.FailAfter(-1)
	b.Release(12)
	// resize(0) performs no reserve, including a zero-byte reserve callback.
	b.FailAfter(0)
	values = nil
	if e := upolyFactorInt32Resize(&values, 0, -1, b); e != nil {
		t.Fatal(e)
	}
}
