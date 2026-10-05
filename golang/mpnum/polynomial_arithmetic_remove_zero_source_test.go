// Source: src/math/polynomial/polynomial.cpp:2067-2092,2094-2135.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "testing"

func TestPolynomialRegularSourceZeroRemovalTransfersActualTags(t *testing.T) {
	full := polynomialArithmeticDirectNumericFixture(t)
	c := full.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	x, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	y, e := c.monomials.MkVariable(1, 1)
	serviceMust(t, e)
	p, e := c.MkTerms([]Int{NewInt(2), NewInt(1)}, []Monomial{x, y})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	two := NewInt(2)
	serviceMust(t, c.modular.SetZP(&two))
	var r polynomialArithmeticInvocationReceipt
	serviceMust(t, c.som.acquireArithmeticInvocation(&r))
	serviceMust(t, c.som.addPolynomialWithArithmeticReceipt(p, &r))
	serviceMust(t, c.som.removeZeros(false))
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
	n, e := c.som.ms.size(c)
	serviceMust(t, e)
	term, e := c.som.ms.monomial(c, 0)
	serviceMust(t, e)
	a, e := c.som.as.coefficient(c, 0)
	serviceMust(t, e)
	if n != 1 || term != y || a != NewInt(1) {
		t.Fatal("literal source zero-filter result", n, term, a)
	}
	serviceMust(t, c.cleanupArithmeticInvocation(&r, false))
	serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
	serviceMust(t, c.DecRef(p))
}
