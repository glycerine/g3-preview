// Source: src/util/mpzzp.h:58-68,154,230-233;
// src/math/polynomial/polynomial.cpp:1988-2039.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "testing"

// Invalid passed-owner metadata is a Go boundary guard fixture, not a source
// arithmetic invocation or a numerical allocation-prefix credit.
func TestPolynomialArithmeticReceiptRejectsForeignModularReceiver(t *testing.T) {
	full := polynomialArithmeticDirectNumericFixture(t)
	c := full.core
	foreign, e := NewIntegerModularManagerWithUint64(c.integer, 5, true)
	serviceMust(t, e)
	defer func() { serviceMust(t, foreign.Close()) }()
	var r polynomialArithmeticInvocationReceipt
	serviceMust(t, c.som.acquireArithmeticInvocation(&r))
	r.frame.executing = true
	var out Int
	serviceMust(t, foreign.setInt64WithArithmeticReceipt(&out, 7, &r))
	r.frame.executing = false
	if out != NewInt(2) || r.frame.normalize != (integerPNormalizeInvocationReceipt{}) || r.frame.numeric != (polynomialNumericRoots{}) {
		t.Fatal("foreign modular manager adopted passed arithmetic authority", out, r.frame.normalize, r.frame.numeric)
	}
	serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
	serviceMust(t, c.integer.Del(&out))
}

// The same concrete receipt address may be reused at a later actual buffer
// invocation. A previously passed allocation context must retain its old nonce.
func TestPolynomialArithmeticContextKeepsCapturedInvocationAcrossReceiptReuse(t *testing.T) {
	full := polynomialArithmeticDirectNumericFixture(t)
	c := full.core
	five := NewInt(5)
	serviceMust(t, c.modular.SetZP(&five))
	var r polynomialArithmeticInvocationReceipt
	serviceMust(t, c.som.acquireArithmeticInvocation(&r))
	r.frame.executing = true
	old := c.modular.arithmeticAllocationContext(&r)
	if old.roots(c.integer) != &r.frame.numeric {
		t.Fatal("actual current allocation context rejected")
	}
	r.frame.executing = false
	serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
	serviceMust(t, c.som.acquireArithmeticInvocation(&r))
	r.frame.executing = true
	if old.roots(c.integer) != nil {
		t.Fatal("old allocation context adopted later invocation through reused receipt pointer")
	}
	out := NewInt(7)
	serviceMust(t, c.modular.normalizeWithSourceContext(&out, old))
	if out != NewInt(2) || r.frame.normalize != (integerPNormalizeInvocationReceipt{}) {
		t.Fatal("stale context acquired younger PN authority", out, r.frame.normalize)
	}
	current := c.modular.arithmeticAllocationContext(&r)
	if current.roots(c.integer) != &r.frame.numeric {
		t.Fatal("new actual context rejected")
	}
	r.frame.executing = false
	serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
}
