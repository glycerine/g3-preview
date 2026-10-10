// Source: src/math/polynomial/polynomial.cpp:5574-5583;
// src/util/ref_vector.h:128-132.
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialPSCChain1AscendingSetAndLastUnitCallbackRefuseOwnerReentry(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	fourth, e := c.monomials.MkVariable(0, 4)
	serviceMust(t, e)
	linear, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	// Actual raw source inputs are borrowed. The source output SET acquires them;
	// later coefficient and final unit SETs are their literal last owning drops.
	p, e := c.MkTerms([]unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(1)}, []Monomial{fourth, c.monomials.unit})
	serviceMust(t, e)
	q, e := c.MkTerms([]unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(1)}, []Monomial{linear, c.monomials.unit})
	serviceMust(t, e)
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupPSCChain1()); serviceMust(t, out.Close()) })
	calls := 0
	h, e := NewPolynomialDeletionHandler(m.originalBudget, func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, token Polynomial) error {
		if token != p && token != q {
			return nil
		}
		calls++
		s := &m.pscChain1
		if !s.active || !s.executing || s.size != 5 || m.subresultant.active || m.subresultant.constructed != 0 {
			t.Fatal("source size captured after completed subchain", s.phase, s.size)
		}
		if token == q && (s.phase != 3 || s.index != 3) {
			t.Fatal("ascending source coefficient SET", s.phase, s.index)
		}
		if token == p && (s.phase != 4 || s.index != 4) {
			t.Fatal("last source unit SET", s.phase, s.index)
		}
		cause, inv := s.cleanupCause, s.invocation
		if x := m.CleanupPSCChain1(); x != ErrPolynomialPending || s.cleanupCause != cause {
			t.Fatal("executing caller cleanup changed history", x)
		}
		if x := m.PSCChain1(c.unit, c.unit, 0, out); x != ErrPolynomialPending || s.invocation != inv {
			t.Fatal("same caller reentry changed invocation", x)
		}
		if x := m.CloseChecked(); x != ErrPolynomialPending || m.closing {
			t.Fatal("parent owner close changed", x)
		}
		if x := m.SourceFinalize(); x != ErrPolynomialPending || m.sourceStarted {
			t.Fatal("parent source finalize changed", x)
		}
		if x := out.Close(); x != ErrPolynomialPending || out.closing {
			t.Fatal("executing actual SET vector closed", x)
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	serviceMust(t, m.PSCChain1(p, q, 0, out))
	if calls != 2 {
		t.Fatal("actual two source SET deletion callbacks", calls)
	}
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
}

// Actual factory authority transfers at the caller SET's completed incoming
// INC, before the old polynomial deletion handler starts a foreign Coeff.
func TestPolynomialPSCChain1FactoryTransferBeforeCallbackPreservesForeignCoefficient(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	other := polynomialQueryTerms(t, c, []int32{7, 11}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{1, 1}}})
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	var foreign polynomialReference
	t.Cleanup(func() {
		m.originalBudget.FailAfter(-1)
		serviceMust(t, m.CleanupPSCChain1())
		serviceMust(t, out.Close())
		serviceMust(t, foreign.closeValue(c, true))
		serviceMust(t, foreign.abort(c))
	})
	var foreignCoefficient polynomialCoefficientInvocationReceipt
	var foreignAppend polynomialCheapAppendInvocationReceipt
	var foreignFactory polynomialFactoryInvocationReceipt
	calls := 0
	h, e := NewPolynomialDeletionHandler(m.originalBudget, func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, token Polynomial) error {
		s := &m.pscChain1
		if s.phase != 3 || s.index != 0 || calls != 0 {
			return nil
		}
		old, x := out.Get(0)
		serviceMust(t, x)
		if token != old {
			return nil
		}
		calls++
		if out.state.incoming == (Polynomial{}) || s.factory.frame == nil || s.factory.frame.result != (Polynomial{}) || c.matchesFactoryInvocation(&s.factory) || c.cheap.appendScope.active {
			t.Fatal("factory not transferred at incoming INC before old DEC")
		}
		v, x := c.coeffWithReceipts(other, 1, 1, &foreignCoefficient, &foreignAppend, &foreignFactory)
		serviceMust(t, x)
		serviceMust(t, foreign.assignWithFactoryReceipt(c, v, &foreignFactory))
		m.originalBudget.FailAfter(0)
		x = m.originalBudget.Reserve(1)
		if x == nil {
			t.Fatal("actual returning callback reservation refusal absent")
		}
		return x
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	returned := m.PSCChain1(p, q, 1, out)
	if returned == nil || calls != 1 || foreign.value == (Polynomial{}) {
		t.Fatal("actual source SET callback refusal", returned, calls)
	}
	foreignToken := foreign.value
	// Deletion handler is an independent externally owned public object;
	// AddDeletionHandler borrows it and its exact header remains charged here.
	known := []Polynomial{p, q, other, foreign.value, foreign.incoming, out.state.incoming}
	now, x := out.Size()
	serviceMust(t, x)
	for i := out.state.drain; i < now; i++ {
		v, x := out.Get(i)
		serviceMust(t, x)
		known = append(known, v)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, known, nil, out.header+h.header, nil)
	serviceMust(t, c.RemoveDeletionHandler(h))
	m.originalBudget.FailAfter(-1)
	serviceMust(t, h.CloseChecked())
	serviceMust(t, m.CleanupPSCChain1())
	if foreign.value != foreignToken {
		t.Fatal("older caller ACK consumed foreign result")
	}
	n, e := out.Size()
	serviceMust(t, e)
	if n != 3 {
		t.Fatal("partial source output range", n)
	}
	got, e := out.Get(0)
	serviceMust(t, e)
	if got != (Polynomial{}) {
		t.Fatal("exact failed SET slot not retired", got)
	}
	got, e = out.Get(1)
	serviceMust(t, e)
	if got != q {
		t.Fatal("old caller ACK swept later source slot", got)
	}
	serviceMust(t, c.cleanupCoefficientInvocation(&foreignCoefficient, true))
	if foreign.value != foreignToken {
		t.Fatal("foreign lexical child ACK stole caller-owned result")
	}
	text := returned.Error()
	serviceMust(t, m.PSCChain1(p, q, 1, out))
	if returned.Error() != text {
		t.Fatal("old returned exception retargeted")
	}
	subresultantNamedAudit(t, m, out, []Polynomial{p, q, other, foreign.value}, nil)
}
