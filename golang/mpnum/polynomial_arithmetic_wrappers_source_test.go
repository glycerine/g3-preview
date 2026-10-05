// Source: src/math/polynomial/polynomial.cpp:2665-2744.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "testing"

func TestPolynomialArithmeticWrappersUseActualReturnedReceipt(t *testing.T) {
	full := polynomialArithmeticDirectNumericFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{3}, [][]MonomialPower{nil})
	q := polynomialQueryTerms(t, c, []int32{2}, [][]MonomialPower{nil})
	for op := 0; op < 3; op++ {
		var r polynomialArithmeticInvocationReceipt
		var result Polynomial
		var e error
		switch op {
		case 0:
			result, e = c.addWithReceipt(p, q, &r)
		case 1:
			result, e = c.subWithReceipt(p, q, &r)
		case 2:
			result, e = c.mulWithReceipt(p, q, &r)
		}
		serviceMust(t, e)
		polynomialCaller(t, c, result)
		want := []int64{5, 1, 6}[op]
		polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{}: want})
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, result}, nil, 0)
		if r.invocation == 0 || c.matchesArithmeticInvocation(&r) {
			t.Fatal("actual returned buffer scope not acknowledged", op)
		}
		before := c.integer.budget.Used()
		serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
		if c.integer.budget.Used() != before {
			t.Fatal("returned token remained owned by invocation", op)
		}
	}
}
