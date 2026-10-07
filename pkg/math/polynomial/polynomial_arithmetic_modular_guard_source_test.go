// Source: src/util/mpzzp.h:58-68,154,230-233;
// src/math/polynomial/polynomial.cpp:1988-2039.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// Invalid passed-owner metadata is a Go boundary guard fixture, not a source
// arithmetic invocation or a numerical allocation-prefix credit.
func TestPolynomialArithmeticReceiptRejectsForeignModularReceiver(t *testing.T) {
	full := polynomialArithmeticDirectNumericFixture(t)
	c := full.core
	foreign, e := util.NewIntegerModularManagerWithUint64(c.integer, 5, true)
	serviceMust(t, e)
	defer func() { serviceMust(t, foreign.Close()) }()
	var r polynomialArithmeticInvocationReceipt
	serviceMust(t, c.som.acquireArithmeticInvocation(&r))
	r.frame.executing = true
	var out util.Int
	serviceMust(t, numericIntegerModularManager_setInt64WithArithmeticReceipt(foreign, &out, 7, &r))
	r.frame.executing = false
	if out != util.NewInt(2) || r.frame.normalize != (util.IntegerNormalizeInvocationReceipt{}) || (r.frame.numeric).NumericAllocationRoots != (polynomialNumericRoots{}).NumericAllocationRoots {
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
	five := util.NewInt(5)
	serviceMust(t, c.modular.SetZP(&five))
	var r polynomialArithmeticInvocationReceipt
	serviceMust(t, c.som.acquireArithmeticInvocation(&r))
	r.frame.executing = true
	old := numericIntegerModularManager_arithmeticAllocationContext(c.modular, &r)
	if (old.roots(c.integer)) != (&r.frame.numeric) {
		t.Fatal("actual current allocation context rejected")
	}
	r.frame.executing = false
	serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
	serviceMust(t, c.som.acquireArithmeticInvocation(&r))
	r.frame.executing = true
	if (old.roots(c.integer)) != nil {
		t.Fatal("old allocation context adopted later invocation through reused receipt pointer")
	}
	out := util.NewInt(7)
	serviceMust(t, numericIntegerModularManager_normalizeWithSourceContext(c.modular, &out, old))
	if out != util.NewInt(2) || r.frame.normalize != (util.IntegerNormalizeInvocationReceipt{}) {
		t.Fatal("stale context acquired younger PN authority", out, r.frame.normalize)
	}
	current := numericIntegerModularManager_arithmeticAllocationContext(c.modular, &r)
	if (current.roots(c.integer)) != (&r.frame.numeric) {
		t.Fatal("new actual context rejected")
	}
	r.frame.executing = false
	serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
}
