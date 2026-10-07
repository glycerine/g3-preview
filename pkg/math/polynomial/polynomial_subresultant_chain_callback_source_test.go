// Source: src/math/polynomial/polynomial.cpp:5509-5520;
// src/util/ref_vector.h:57-60.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialSubresultantChainResetCallbackBeforeLocalConstructorsRefusesOwnerClose(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 4}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupSubresultantChain()); serviceMust(t, out.Close()) })
	term, e := c.monomials.MkVariable(0, 3)
	serviceMust(t, e)
	old, e := c.MkTerms([]util.Int{util.NewInt(5), util.NewInt(1)}, []Monomial{term, c.monomials.unit})
	serviceMust(t, e)
	serviceMust(t, out.Push(old))
	calls := 0
	h, e := NewPolynomialDeletionHandler(m.originalBudget, func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, token Polynomial) error {
		if token != old || calls != 0 {
			return nil
		}
		calls++
		if !m.subresultant.active || !m.subresultant.executing || m.subresultant.constructed != 0 {
			t.Fatal("source reset precedes all three local reference constructors")
		}
		closing, stage, started := m.closing, m.cleanupStage, m.sourceStarted
		cleanup := m.subresultant.cleanupCause
		if e := m.CleanupSubresultantChain(); e != ErrPolynomialPending || m.subresultant.cleanupCause != cleanup {
			t.Fatal("executing parent cleanup mutated cause", e, m.subresultant.cleanupCause)
		}
		if e := m.CloseChecked(); e != ErrPolynomialPending || m.closing != closing || m.cleanupStage != stage {
			t.Fatal("executing full owner close mutated", e)
		}
		if e := m.SourceFinalize(); e != ErrPolynomialPending || m.sourceStarted != started {
			t.Fatal("executing source owner finalize mutated", e)
		}
		if e := out.Close(); e != ErrPolynomialPending || out.closing || out.closed {
			t.Fatal("executing source vector close mutated", e, out.closing, out.closed)
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	serviceMust(t, m.SubresultantChain(p, q, 0, out))
	if calls != 1 {
		t.Fatal("actual source reset callback", calls)
	}
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
}

// A returning deletion handler is a supported owner callback boundary. The
// source reset failed before R/prem/newS constructors; caller output still
// owns its reset continuation and is not swept by parent acknowledgment.
func TestPolynomialSubresultantChainResetFailureHasNoLocalDestruction(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 4}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupSubresultantChain()); serviceMust(t, out.Close()) })
	term, e := c.monomials.MkVariable(0, 3)
	serviceMust(t, e)
	old, e := c.MkTerms([]util.Int{util.NewInt(5), util.NewInt(1)}, []Monomial{term, c.monomials.unit})
	serviceMust(t, e)
	serviceMust(t, out.Push(old))
	cause := errors.New("actual subresultant source reset callback")
	h, e := NewPolynomialDeletionHandler(m.originalBudget, func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, token Polynomial) error {
		if token == old {
			return cause
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	returned := m.SubresultantChain(p, q, 0, out)
	if !errors.Is(returned, cause) || m.subresultant.constructed != 0 || m.subresultant.bodyCause != cause || m.subresultant.sourceCause != nil || !out.state.draining {
		t.Fatal("reset failure prefix", returned, m.subresultant.constructed, m.subresultant.bodyCause, m.subresultant.sourceCause, out.state.draining)
	}
	serviceMust(t, m.CleanupSubresultantChain())
	if !out.state.draining || m.subresultant.bodyCause != cause {
		t.Fatal("parent acknowledged caller reset or erased body history")
	}
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
	serviceMust(t, out.Close())
	fresh, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fresh.Close()) })
	serviceMust(t, m.SubresultantChain(p, q, 0, fresh))
	if !errors.Is(returned, cause) {
		t.Fatal("old returned error retargeted to new invocation", returned)
	}
}

// ref_vector.h:128-132 increments the new token before the old DEC callback.
// A stopped SET keeps exactly that incoming INC; its receipt owns no other slot.
func TestPolynomialSubresultantVectorSetIncomingBeforeOldDropAndExactReceipt(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	incoming := polynomialQueryTerms(t, c, []int32{7, 1}, [][]MonomialPower{{{0, 1}}, nil})
	foreign := polynomialQueryTerms(t, c, []int32{9, 1}, [][]MonomialPower{{{0, 2}}, nil})
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	term, e := c.monomials.MkVariable(0, 3)
	serviceMust(t, e)
	old, e := c.MkTerms([]util.Int{util.NewInt(5), util.NewInt(1)}, []Monomial{term, c.monomials.unit})
	serviceMust(t, e)
	serviceMust(t, out.Push(old))
	serviceMust(t, out.Push(foreign))
	cause := errors.New("actual source SET older DEC callback")
	h, e := NewPolynomialDeletionHandler(m.originalBudget, func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, token Polynomial) error {
		if token != old {
			return nil
		}
		if out.state.incoming != incoming || !out.state.setExecuting {
			t.Fatal("incoming INC not retained before old DEC")
		}
		if e := out.Close(); e != ErrPolynomialPending || out.closing {
			t.Fatal("SET callback close mutated", e)
		}
		return cause
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	var receipt polynomialReferenceVectorSetReceipt
	returned := out.setWithReceipt(0, incoming, &receipt)
	if !errors.Is(returned, cause) || !receipt.matches() || out.state.incoming != incoming || !out.state.setActive {
		t.Fatal("actual SET stopped prefix", returned, out.state)
	}
	copied := out.state
	forged := receipt
	forged.frame = &copied
	serviceMust(t, forged.cleanup())
	if out.state.incoming != incoming || !out.state.setActive {
		t.Fatal("copied frame consumed actual incoming")
	}
	serviceMust(t, receipt.cleanup())
	if receipt.bodyCause() != cause || out.state.incoming != (Polynomial{}) || out.state.setActive {
		t.Fatal("exact ACK/history", receipt.bodyCause(), out.state)
	}
	got, e := out.Get(1)
	serviceMust(t, e)
	if got != foreign {
		t.Fatal("ACK swept foreign slot", got)
	}
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
	var later polynomialReferenceVectorSetReceipt
	serviceMust(t, out.setWithReceipt(0, incoming, &later))
	serviceMust(t, receipt.cleanup())
	got, e = out.Get(0)
	serviceMust(t, e)
	if got != incoming || later.invocation == receipt.invocation || receipt.matches() {
		t.Fatal("stale SET receipt touched later slot")
	}
	if !errors.Is(returned, cause) {
		t.Fatal("returned cause changed after later invocation")
	}
}
