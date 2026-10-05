// Source: src/math/polynomial/polynomial.cpp:1988-2039.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "testing"

func TestPolynomialRegularActualINCBindsRowBeforeCoefficientConstruction(t *testing.T) {
	for prefix := int64(0); prefix < 32; prefix++ {
		full := polynomialArithmeticDirectNumericFixture(t)
		c := full.core
		seven := NewInt(7)
		p, e := c.MkConstantTakeCopy(&seven)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		var r polynomialArithmeticInvocationReceipt
		serviceMust(t, c.som.acquireArithmeticInvocation(&r))
		before, e := c.monomialRefCount(c.monomials.unit)
		serviceMust(t, e)
		one := NewInt(1)
		c.integer.budget.FailAfter(prefix)
		e = c.som.addMulWithArithmeticReceipt(&one, c.monomials.unit, p, &r)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
		ms, x := c.som.ms.size(c)
		serviceMust(t, x)
		as, x := c.som.as.size(c)
		serviceMust(t, x)
		after, x := c.monomialRefCount(c.monomials.unit)
		serviceMust(t, x)
		t.Log("regular actual prefix", prefix, e, "columns", ms, as, "unitRC", before, after)
		if e != nil && ms == 1 && as == 0 && after == before+1 {
			if !c.som.matchesAppendInvocation(&r.frame.append) || len(c.som.monomialOwners) == 0 || c.som.monomialOwners[0] != r.frame.append.invocation || r.frame.append.frame.monomialClaims != 1 {
				t.Fatal("reached source INC lacks exact per-slot authority", prefix, e)
			}
			c.integer.budget.FailAfter(-1)
			serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
			afterOwn, x := c.monomialRefCount(c.monomials.unit)
			serviceMust(t, x)
			position, x := c.som.positions.Get(c.monomials.unit)
			serviceMust(t, x)
			if afterOwn != before || position != ^uint32(0) {
				t.Fatal("matched regular row abort retained source position/INC", afterOwn, position)
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
			serviceMust(t, c.som.addMul(&one, c.monomials.unit, p))
			serviceMust(t, c.som.close())
			serviceMust(t, c.DecRef(p))
			return
		}
		c.integer.budget.FailAfter(-1)
		serviceMust(t, c.som.close())
		serviceMust(t, c.DecRef(p))
	}
	t.Fatal("did not reach actual regular ms-push/INC before coefficient ctor")
}
