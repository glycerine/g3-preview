// Source: src/math/polynomial/polynomial.cpp:3264-3277,1392-1423.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"testing"
)

// Invalid copied Go metadata must not acknowledge a real child handoff. The
// actual incoming INC/frontier is tested separately from this identity guard.
func TestPolynomialCoefficientFactoryACKRequiresActualSourceFrame(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	p = polynomialCaller(t, c, p)
	var coefficient polynomialCoefficientInvocationReceipt
	var appendReceipt polynomialCheapAppendInvocationReceipt
	var factory polynomialFactoryInvocationReceipt
	result, e := c.coeffCoreWithReceipts(p, 0, 1, nil, &coefficient, &appendReceipt, &factory)
	serviceMust(t, e)
	ar := &c.cheap.arithmeticScope
	if !ar.active || !ar.coefficientReady || !coefficient.frame.bufferActive {
		t.Fatal("actual completed but untransferred child", ar)
	}
	copied := *factory.frame
	c.acknowledgeCoefficientArithmeticFactory(&copied, &factory)
	if !ar.active || !coefficient.frame.bufferActive {
		t.Fatal("copied factory frame acknowledged real child")
	}
	serviceMust(t, c.IncRef(result))
	serviceMust(t, c.transferFactoryResult(&factory, result))
	if ar.active || coefficient.frame.bufferActive {
		t.Fatal("actual incoming INC handoff did not acknowledge matched child")
	}
	serviceMust(t, c.cleanupCoefficientInvocation(&coefficient, true))
	serviceMust(t, c.DecRef(result))
}

func TestPolynomialCoefficientTwoBufferStaleChildrenPreserveReplacement(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	p = polynomialCaller(t, c, p)
	var reduct polynomialReference
	var old, current polynomialCoefficientInvocationReceipt
	var oldAppend, appendReceipt polynomialCheapAppendInvocationReceipt
	var oldFactory, factory polynomialFactoryInvocationReceipt
	first, e := c.coeffCoreWithReceipts(p, 0, 1, &reduct, &old, &oldAppend, &oldFactory)
	serviceMust(t, e)
	oldChildren := old.frame.arithmetic
	serviceMust(t, c.IncRef(first))
	serviceMust(t, c.transferFactoryResult(&oldFactory, first))
	second, e := c.coeffCoreWithReceipts(p, 0, 1, &reduct, &current, &appendReceipt, &factory)
	serviceMust(t, e)
	if old.frame != current.frame || old.invocation == current.invocation || oldChildren[0].frame != current.frame.arithmetic[0].frame || oldChildren[0].invocation == current.frame.arithmetic[0].invocation {
		t.Fatal("actual reused child/frame lacks captured generation")
	}
	before := c.coefficientInvocation
	factoryBefore := c.factoryInvocation
	serviceMust(t, c.cleanupCoefficientInvocation(&old, false))
	serviceMust(t, c.cleanupCoefficientInvocation(&old, true))
	for i := range oldChildren {
		serviceMust(t, c.cleanupArithmeticInvocation(&oldChildren[i], false))
		serviceMust(t, c.cleanupArithmeticInvocation(&oldChildren[i], true))
	}
	serviceMust(t, c.cleanupFactoryInvocation(&oldFactory, true))
	serviceMust(t, c.cheap.cleanupAppendInvocation(&oldAppend, true))
	if c.coefficientInvocation != before || c.factoryInvocation != factoryBefore {
		t.Fatal("stale child consumed replacement metadata")
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, first, second, reduct.value, reduct.incoming}, nil, 0)
	serviceMust(t, c.IncRef(second))
	serviceMust(t, c.transferFactoryResult(&factory, second))
	serviceMust(t, reduct.closeValue(c, true))
	serviceMust(t, reduct.abort(c))
	serviceMust(t, c.cleanupCoefficientInvocation(&current, true))
	serviceMust(t, c.DecRef(first))
	serviceMust(t, c.DecRef(second))
}
