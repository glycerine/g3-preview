// Source: src/math/polynomial/polynomial.cpp:5588-5658; src/util/ref_vector.h:44-48.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "testing"

func TestPolynomialPSCChain2ResetCallbackNineReferencesAndOwnerRefusal(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	third, e := c.monomials.MkVariable(0, 3)
	serviceMust(t, e)
	old, e := c.MkTerms([]Int{NewInt(7), NewInt(3)}, []Monomial{third, c.monomials.unit})
	serviceMust(t, e)
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	serviceMust(t, out.Push(old))
	t.Cleanup(func() { serviceMust(t, m.CleanupPSCChain2()); serviceMust(t, out.Close()) })
	calls := 0
	h, e := NewPolynomialDeletionHandler(m.originalBudget, func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, token Polynomial) error {
		if token != old {
			return nil
		}
		calls++
		s := &m.pscChain2
		if !s.active || !s.executing || s.constructed != 9 || s.refs[0].root.value != p || s.refs[1].root.value != q {
			t.Fatal("literal nine refs and assignments before Reset", s.constructed)
		}
		inv, cause := s.invocation, s.cleanupCause
		if e := m.CleanupPSCChain2(); e != ErrPolynomialPending || s.cleanupCause != cause {
			t.Fatal("executing own cleanup mutated", e)
		}
		if e := m.PSCChain2(p, q, 0, out); e != ErrPolynomialPending || s.invocation != inv {
			t.Fatal("same frame reentry", e)
		}
		if e := m.CloseChecked(); e != ErrPolynomialPending || m.closing {
			t.Fatal("owner Close before scope guard", e)
		}
		if e := m.SourceFinalize(); e != ErrPolynomialPending || m.sourceStarted {
			t.Fatal("owner source finalization before scope guard", e)
		}
		if e := out.Close(); e != ErrPolynomialPending || out.closing {
			t.Fatal("caller vector reset reentry", e)
		}
		copied := *m
		if e := copied.finishPSCChain2Source(); e != ErrPolynomialOwner {
			t.Fatal("copied source owner", e)
		}
		if e := copied.CleanupPSCChain2(); e != ErrPolynomialOwner {
			t.Fatal("copied owning parent", e)
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	serviceMust(t, m.PSCChain2(p, q, 0, out))
	if calls != 1 {
		t.Fatal("actual reset deletion", calls)
	}
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
}
