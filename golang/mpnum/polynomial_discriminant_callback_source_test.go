// Source: src/math/polynomial/polynomial.cpp:5473-5500; src/util/obj_ref.h.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"testing"
)

func TestPolynomialDiscriminantDegreeZeroConstructsRefBeforeCallbackAndRefusesOwnerClose(t *testing.T) {
	m := polynomialDiscriminantFixture(t)
	c := m.core
	five, seven := NewInt(5), NewInt(7)
	p, e := c.MkConstantTakeCopy(&five)
	serviceMust(t, e)
	p = polynomialCaller(t, c, p)
	old, e := c.MkConstantTakeCopy(&seven)
	serviceMust(t, e)
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, r.Assign(old))
	calls := 0
	var closeError error
	h, e := NewPolynomialDeletionHandler(c.integer.budget, func(_ *PolynomialDeletionHandler, core *MultivariatePolynomialCore, value Polynomial) error {
		if value != old {
			return nil
		}
		calls++
		s := &m.discriminant
		if !s.active || !s.executing || s.depth != 1 || s.stack[0] != 0 || s.refs[0].value != (Polynomial{}) || s.derivative.invocation != 0 || s.constant.invocation != 0 || !r.root.assigning || r.root.incoming != core.zero {
			t.Fatal("literal zero-degree p_prime/assignment prefix")
		}
		closeError = m.CloseChecked()
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	serviceMust(t, m.Discriminant(p, 0, r))
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
	serviceMust(t, r.Close())
	if calls != 1 || !errors.Is(closeError, ErrPolynomialPending) || m.discriminant.active {
		t.Fatal("active actual parent close guard", calls, closeError, m.discriminant.active)
	}
}

func TestPolynomialDiscriminantDegreeZeroCallbackRefusesSourceFinalizeBeforeMutation(t *testing.T) {
	m := polynomialDiscriminantFixture(t)
	c := m.core
	five, seven := NewInt(5), NewInt(7)
	p, e := c.MkConstantTakeCopy(&five)
	serviceMust(t, e)
	p = polynomialCaller(t, c, p)
	old, e := c.MkConstantTakeCopy(&seven)
	serviceMust(t, e)
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, r.Assign(old))
	calls := 0
	var closeError error
	h, e := NewPolynomialDeletionHandler(c.integer.budget, func(_ *PolynomialDeletionHandler, core *MultivariatePolynomialCore, value Polynomial) error {
		if value != old {
			return nil
		}
		calls++
		s := &m.discriminant
		if !s.active || !s.executing || s.depth != 1 || s.stack[0] != 0 || s.refs[0].value != (Polynomial{}) || s.derivative.invocation != 0 || s.constant.invocation != 0 || !r.root.assigning || r.root.incoming != core.zero {
			t.Fatal("literal zero-degree p_prime/assignment prefix")
		}
		started, stage, cause := m.sourceStarted, m.bodyStage, m.sourceCause
		closeError = m.SourceFinalize()
		if started != m.sourceStarted || stage != m.bodyStage || cause != m.sourceCause {
			t.Fatal("executing Discriminant SourceFinalize mutated source owner prefix", closeError)
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	serviceMust(t, m.Discriminant(p, 0, r))
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
	serviceMust(t, r.Close())
	if calls != 1 || !errors.Is(closeError, ErrPolynomialPending) || m.discriminant.active {
		t.Fatal("active actual parent close guard", calls, closeError, m.discriminant.active)
	}
}
