// Source: src/math/polynomial/polynomial.cpp:1885-2156,2665-2744.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "testing"

func TestPolynomialArithmeticGuardedEmptySourceBufferAcquisition(t *testing.T) {
	c := polynomialCellFixture(t)
	var r polynomialArithmeticInvocationReceipt
	before := c.integer.budget.Used()
	c.integer.budget.FailAfter(0)
	e := c.som.acquireArithmeticInvocation(&r)
	if e != nil || !c.matchesArithmeticInvocation(&r) || r.frame != &c.som.arithmeticScope || c.integer.budget.Used() != before {
		t.Fatal("actual embedded acquisition", e)
	}
	saved := *r.frame
	var rejected polynomialArithmeticInvocationReceipt
	if e = c.som.acquireArithmeticInvocation(&rejected); e != ErrPolynomialPending || rejected.manager != nil || *r.frame != saved {
		t.Fatal("busy guarded entry changed retained identity", e)
	}
	copied := c.som
	if e = copied.acquireArithmeticInvocation(&rejected); e != ErrPolynomialOwner || rejected.manager != nil || *r.frame != saved {
		t.Fatal("copied buffer acquired authority", e)
	}
	c.integer.budget.FailAfter(-1)
	// This actual empty acquisition has no NUM, monomial INC, or allocation roots.
	polynomialPhysicalAudit(t, c, nil, nil, nil)
}

func TestPolynomialRegularZeroAddMulPreservesUnreadInvocation(t *testing.T) {
	c := polynomialCellFixture(t)
	var older polynomialArithmeticInvocationReceipt
	serviceMust(t, c.som.acquireArithmeticInvocation(&older))
	saved := *older.frame
	var erased polynomialArithmeticInvocationReceipt
	zero := NewInt(0)
	before := c.integer.budget.Used()
	c.integer.budget.FailAfter(0)
	e := c.som.addMulWithArithmeticReceipt(&zero, Monomial{}, Polynomial{}, &erased)
	if e != nil || erased.manager != nil || *older.frame != saved || c.integer.budget.Used() != before {
		t.Fatal("source zero addmul inspected/acquired erased dependencies", e)
	}
	c.integer.budget.FailAfter(-1)
	polynomialPhysicalAudit(t, c)
}
