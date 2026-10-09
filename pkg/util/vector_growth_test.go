// Source: src/util/vector.h:467-487.
package util

import (
	"errors"
	"testing"
)

func TestSourceVectorGrowthWidths(t *testing.T) {
	if n, b, e := VectorGrowth(0x40000000); e != nil || n != 0x60000000 || b != 0x80000008 {
		t.Fatal(n, b, e)
	}
	for _, p := range [][2]uint32{{0, 2}, {2, 3}, {3, 5}, {5, 8}, {8, 12}, {12, 18}, {18, 27}} {
		n, _, e := VectorGrowth(p[0])
		if e != nil || n != p[1] {
			t.Fatal(p, n, e)
		}
	}
	for _, old := range []uint32{0x55555555, 0xaaaaaaaa, ^uint32(0)} {
		_, _, e := VectorGrowth(old)
		if !errors.Is(e, Failure{Kind: FailureVectorOverflow}) {
			t.Fatal(old, e)
		}
	}
}
