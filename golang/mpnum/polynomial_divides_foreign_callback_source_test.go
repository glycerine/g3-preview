// Source: src/math/polynomial/polynomial.cpp:5154-5194.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"testing"
)

func TestPolynomialDividesCheckpointForeignRootsSurviveMatchedAbort(t *testing.T) {
	full := polynomialArithmeticDirectNumericFixture(t)
	c := full.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	var foreign Int
	var foreignPolynomial Polynomial
	calls := 0
	c.config.Checkpoint = func() bool {
		calls++
		if calls != 1 {
			t.Fatal("source returned after cancellation")
		}
		if c.callbackDepth == 0 || !c.dividesLocal.executing {
			t.Fatal("actual source callback frontier absent")
		}
		serviceMust(t, c.integer.Copy(&foreign, &c.integer.two64))
		foreignPolynomial, e = c.MkVariable(0, 1)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(foreignPolynomial))
		return false
	}
	var receipt polynomialDividesInvocationReceipt
	_, returned := c.dividesWithReceipt(c.unit, c.unit, &receipt)
	if calls != 1 || !errors.Is(returned, ErrUPolynomialCanceled) {
		t.Fatal("actual source checkpoint cancellation", calls, returned)
	}
	cell := foreign.cell
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{foreignPolynomial}, []*Int{&foreign}, 0)
	serviceMust(t, c.cleanupDividesInvocation(&receipt, false))
	serviceMust(t, c.cleanupDividesInvocation(&receipt, true))
	if foreign.cell != cell {
		t.Fatal("matched parent abort consumed actual foreign numeral")
	}
	equal, e := c.integer.Equal(&foreign, &c.integer.two64)
	serviceMust(t, e)
	info, e := c.Info(foreignPolynomial)
	serviceMust(t, e)
	if !equal || info.RefCount != 1 {
		t.Fatal("foreign callback value/RC changed", equal, info)
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{foreignPolynomial}, []*Int{&foreign}, 0)
	c.config.Checkpoint = func() bool { return true }
	answer, e := c.Divides(c.unit, c.unit)
	serviceMust(t, e)
	if !answer || !errors.Is(returned, ErrUPolynomialCanceled) {
		t.Fatal("new invocation changed returned first cause")
	}
	used := c.integer.budget.Used()
	serviceMust(t, c.cleanupDividesInvocation(&receipt, true))
	if c.integer.budget.Used() != used || foreign.cell != cell {
		t.Fatal("stale parent acquired later or foreign roots")
	}
	serviceMust(t, c.DecRef(foreignPolynomial))
	serviceMust(t, c.integer.Del(&foreign))
}
