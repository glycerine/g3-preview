// Source: src/math/polynomial/polynomial.cpp:5940-5951;
// src/util/mpzzp.h:131; src/util/mpz.h:124-149.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "testing"

func TestPolynomialNegDirectBIGCopyActualAllocationPrefixes(t *testing.T) {
	success, pending := false, false
	for prefix := int64(0); prefix < 192 && !success; prefix++ {
		full := polynomialPSCFullFixture(t)
		c := full.core
		a := NewInt(1)
		serviceMust(t, c.integer.Mul2kInPlace(&a, 15000))
		term, e := c.monomials.MkMonomial([]MonomialPower{{0, 1}})
		serviceMust(t, e)
		p, e := c.MkTerms([]Int{a}, []Monomial{term})
		serviceMust(t, e)
		p = polynomialCaller(t, c, p)
		serviceMust(t, c.integer.Neg(&a))
		var receipt polynomialNegInvocationReceipt
		c.integer.budget.FailAfter(prefix)
		result, body := c.negWithReceipt(p, &receipt)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result}, []*Int{&a}, 0)
		if body != nil && polynomialNegNumericPending(&c.numeric) {
			pending = true
			if receipt.manager != c || receipt.frame != &c.polynomialNegScope || receipt.invocation != c.invocation || !c.negativeActive {
				t.Fatal("actual pending Copy lost concrete Neg frame", prefix, body)
			}
		}
		c.integer.budget.FailAfter(-1)
		serviceMust(t, c.cleanupNegInvocation(&receipt, false))
		serviceMust(t, c.cleanupNegInvocation(&receipt, true))
		if body == nil {
			success = true
			t.Log("actual BIG Copy first success", prefix)
			result = polynomialCaller(t, c, result)
			value, e := c.Coefficient(result, 0)
			serviceMust(t, e)
			negative, e := c.integer.IsNeg(&value)
			serviceMust(t, e)
			if !negative {
				t.Fatal("literal BIG minus_a result lost sign", prefix)
			}
			equal, e := c.integer.Equal(&value, &a)
			serviceMust(t, e)
			if !equal {
				t.Fatal("BIG minus_a lost exact magnitude", prefix)
			}
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result}, []*Int{&a}, 0)
		serviceMust(t, c.integer.Del(&a))
	}
	if !success || !pending {
		t.Fatal("actual BIG Copy pending and first success not reached", success, pending)
	}
}
