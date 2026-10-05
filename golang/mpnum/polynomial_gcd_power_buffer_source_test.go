// Source: src/math/polynomial/polynomial.cpp:3207-3256,4391-4392;
// src/util/buffer.h. Licensed under the MIT license.
// Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"fmt"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialGCDPowerSortIndependentUnsignedPairs(t *testing.T) {
	for _, n := range []uint32{0, 1, 2, 3, 31, 32, 33, 127, 129} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			c := polynomialCellFixture(t)
			var b polynomialGCDPowerBuffer
			b.construct(c.memory)
			t.Cleanup(func() { serviceMust(t, b.close(true)) })
			for i := n; i > 0; i-- {
				x := i - 1
				serviceMust(t, b.push(x, x*3+7))
			}
			serviceMust(t, b.sortVariables())
			for i := uint32(0); i < n; i++ {
				x, e := b.word(i, 0)
				serviceMust(t, e)
				k, e := b.word(i, 1)
				serviceMust(t, e)
				if x != i || k != i*3+7 {
					t.Fatal("independent sorted pair", n, i, x, k)
				}
			}
			polynomialNewtonPhysicalWithRaw(t, c, nil, nil, nil, nil, nil, nil, []memory.Block{b.raw, b.pending})
			serviceMust(t, b.close(false))
			polynomialPhysicalAudit(t, c)
		})
	}
}
func TestPolynomialGCDPowerSortUnsignedBoundary(t *testing.T) {
	c := polynomialCellFixture(t)
	var b polynomialGCDPowerBuffer
	b.construct(c.memory)
	for _, x := range []uint32{^uint32(0), 0, 1 << 31, 7, 1 << 31, 0} {
		serviceMust(t, b.push(x, x^19))
	}
	serviceMust(t, b.sortVariables())
	for i, x := range []uint32{0, 0, 7, 1 << 31, 1 << 31, ^uint32(0)} {
		got, e := b.word(uint32(i), 0)
		serviceMust(t, e)
		degree, e := b.word(uint32(i), 1)
		serviceMust(t, e)
		if got != x || degree != x^19 {
			t.Fatal("unsigned lt_var and paired payload", i, got, degree)
		}
	}
	serviceMust(t, b.close(false))
	polynomialPhysicalAudit(t, c)
}

func TestPolynomialGCDPowerDegreeSortSourceAdapter(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	var b polynomialGCDPowerBuffer
	b.construct(full.core.memory)
	for i := uint32(0); i < 41; i++ {
		serviceMust(t, b.push(i+100, 40-i))
	}
	serviceMust(t, b.sortDegrees())
	for i := uint32(0); i < 41; i++ {
		degree, e := b.word(i, 1)
		serviceMust(t, e)
		variable, e := b.word(i, 0)
		serviceMust(t, e)
		if degree != i || variable != 140-i {
			t.Fatal("source degree-only order preserves pair", i, variable, degree)
		}
	}
	serviceMust(t, b.close(false))
	serviceMust(t, b.close(true))
	polynomialImpPhysicalAudit(t, full)
}
