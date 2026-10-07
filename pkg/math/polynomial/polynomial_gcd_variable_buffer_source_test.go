// Source: src/math/polynomial/polynomial.cpp:3209,4243-4264; src/util/buffer.h.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package polynomial

import (
	"testing"
)

func TestPolynomialGCDVariableBufferActualInlineAndOverflow(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	var b polynomialGCDVariableBuffer
	b.construct(c.memory)
	for i := uint32(0); i < 33; i++ {
		serviceMust(t, b.push(i+100))
	}
	if b.capacity != 64 || b.count != 33 {
		t.Fatal("actual source32 expansion", b.capacity, b.count)
	}
	for i := uint32(0); i < 33; i++ {
		v, e := b.get(i)
		serviceMust(t, e)
		if v != i+100 {
			t.Fatal("actual variable field", i, v)
		}
	}
	serviceMust(t, b.close(false))
	serviceMust(t, b.close(true))
	polynomialImpPhysicalAudit(t, full)
}
