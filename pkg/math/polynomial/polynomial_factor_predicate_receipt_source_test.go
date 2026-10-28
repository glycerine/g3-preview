package polynomial

// Source: src/math/polynomial/polynomial.cpp:6737-6752,3536-3549,4593-4598.

import (
	"testing"
)

func TestPolynomialFactorCompiledPredicateActualChildReceipts(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	p := polynomialQueryTerms(t, full.core, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, {}})
	var primitive, square polynomialGCDInvocationReceipt
	yes, e := full.isPrimitiveWithReceipt(p, 0, &primitive)
	serviceMust(t, e)
	if !yes || primitive.invocation == 0 || !full.hasGCDInvocationHistory(&primitive) {
		t.Fatal("actual primitive source child")
	}
	serviceMust(t, full.cleanupGCDInvocation(&primitive, false))
	serviceMust(t, full.cleanupGCDInvocation(&primitive, true))
	yes, e = full.isSquareFreeWithReceipt(p, 0, true, &square)
	serviceMust(t, e)
	if !yes || square.invocation == 0 || !full.hasGCDInvocationHistory(&square) {
		t.Fatal("actual square-free source child")
	}
	serviceMust(t, full.cleanupGCDInvocation(&square, false))
	serviceMust(t, full.cleanupGCDInvocation(&square, true))
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
}
