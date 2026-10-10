// Source: src/math/polynomial/polynomial.cpp:4507-4528;
// src/util/mpz.cpp; src/util/mpzzp.h.
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

// Invalid Go invocation metadata is a refusal-domain guard proof. It cannot
// supply source acquisition credit; actual derivative prefixes test that.
func TestPolynomialDerivativeNumericContextInactiveReceiptUnchanged(t *testing.T) {
	m := polynomialPrimitiveFixture(t)
	c := m.core
	r := polynomialDerivativeInvocationReceipt{manager: m, frame: &m.derivative, invocation: 99}
	before := r
	var degree, result unumeral.Int
	var absent *unumeral.IntegerModularManager
	if e := numericIntegerModularManager_setUint32WithDerivativeReceipt(absent, &degree, 7, &r); e != unumeral.ErrModularOwner || r != before || degree != (unumeral.Int{}) {
		t.Fatal("nil receiver did not retain the original owner guard", e)
	}
	serviceMust(t, numericIntegerModularManager_setUint32WithDerivativeReceipt(c.modular, &degree, ^uint32(0), &r))
	three := unumeral.NewInt(3)
	serviceMust(t, numericIntegerModularManager_mulWithDerivativeReceipt(c.modular, &degree, &three, &result, &r))
	var expected unumeral.Int
	serviceMust(t, c.integer.SetInt64(&expected, 12884901885))
	equal, e := c.integer.Equal(&result, &expected)
	serviceMust(t, e)
	if !equal || r != before || (m.derivative.numeric).NumericAllocationRoots != (polynomialNumericRoots{}).NumericAllocationRoots || m.derivative.normalize != (unumeral.IntegerNormalizeInvocationReceipt{}) {
		t.Fatal("inactive receipt acquired direct numeric authority or changed math")
	}
	for _, v := range []*unumeral.Int{&degree, &result, &expected} {
		serviceMust(t, c.integer.Del(v))
	}
}

func TestPolynomialDerivativeNumericContextRejectsCopiedFrameAndOwner(t *testing.T) {
	m := polynomialPrimitiveFixture(t)
	c := m.core
	// These invalid metadata probes do not execute a source derivative body.
	// A receipt can authorize only the actual embedded frame and live owner.
	m.derivative.active, m.derivative.executing, m.derivative.invocation = true, true, 7
	t.Cleanup(func() {
		m.derivative.active, m.derivative.executing = false, false
	})
	live := polynomialDerivativeInvocationReceipt{manager: m, frame: &m.derivative, invocation: 7}
	if (numericIntegerManager_derivativeAllocationRoots(c.integer, &live)) != (&m.derivative.numeric) {
		t.Fatal("exact concrete frame refused in metadata guard probe")
	}
	copiedFrame := m.derivative
	copiedOwner := *m
	for _, r := range []polynomialDerivativeInvocationReceipt{
		{manager: m, frame: &copiedFrame, invocation: 7},
		{manager: &copiedOwner, frame: &copiedOwner.derivative, invocation: 7},
		{manager: m, frame: &m.derivative, invocation: 6},
	} {
		if (numericIntegerManager_derivativeAllocationRoots(c.integer, &r)) != nil {
			t.Fatal("foreign, copied or stale metadata acquired derivative roots")
		}
	}
	c.callbackDepth++
	if (numericIntegerManager_derivativeAllocationRoots(c.integer, &live)) != nil {
		t.Fatal("external callback acquired enclosing source roots")
	}
	c.callbackDepth--
}

func TestPolynomialDerivativeNumericContextRejectsForeignModularManager(t *testing.T) {
	m := polynomialPrimitiveFixture(t)
	c := m.core
	foreign, e := unumeral.NewIntegerModularManagerWithUint64(c.integer, 5, true)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, foreign.Close()) })
	// Same Integer owner does not make this a call on the parent's actual
	// modular manager. This probes invalid passed-owner metadata, not a source
	// derivative invocation or an allocation-prefix claim.
	m.derivative.active, m.derivative.executing, m.derivative.invocation = true, true, 9
	t.Cleanup(func() { m.derivative.active, m.derivative.executing = false, false })
	r := polynomialDerivativeInvocationReceipt{manager: m, frame: &m.derivative, invocation: 9}
	var result unumeral.Int
	serviceMust(t, numericIntegerModularManager_setUint32WithDerivativeReceipt(foreign, &result, 7, &r))
	if result != unumeral.NewInt(2) || m.derivative.normalize != (unumeral.IntegerNormalizeInvocationReceipt{}) || (m.derivative.numeric).NumericAllocationRoots != (polynomialNumericRoots{}).NumericAllocationRoots {
		t.Fatal("foreign modular manager adopted enclosing derivative authority")
	}
	serviceMust(t, c.integer.Del(&result))
}
