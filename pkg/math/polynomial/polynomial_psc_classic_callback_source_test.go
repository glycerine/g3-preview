package polynomial

// Source: src/math/polynomial/polynomial.cpp:5817-5895; src/util/obj_ref.h:31-112.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestPolynomialPSCClassicResetBeforeConstructorsAndOwnerReentry(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	old := polynomialICCPUnownedTerms(t, c, []int32{23, 11}, [][]MonomialPower{{{0, 3}}, nil})
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	serviceMust(t, out.Push(old))
	t.Cleanup(func() { serviceMust(t, c.CleanupPSCClassic()); serviceMust(t, out.Close()) })
	calls := 0
	h, e := NewPolynomialDeletionHandler(full.originalBudget, func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, v Polynomial) error {
		if v != old {
			return nil
		}
		calls++
		s := &c.pscClassic
		if s.constructed != 0 || s.cursor != 0 || !s.active || !s.executing {
			t.Fatal("source Reset precedes all thirteen constructors")
		}
		inv, cause := s.invocation, s.cleanupCause
		if c.CleanupPSCClassic() != ErrPolynomialPending || s.cleanupCause != cause {
			t.Fatal("executing own mutated")
		}
		if full.PSCChainClassic(p, q, 0, out) != ErrPolynomialPending || s.invocation != inv {
			t.Fatal("same frame reentry")
		}
		if full.CloseChecked() != ErrPolynomialPending || full.closing {
			t.Fatal("owner Close before source guard")
		}
		if full.SourceFinalize() != ErrPolynomialPending || full.sourceStarted {
			t.Fatal("source owner before guard")
		}
		if out.Close() != ErrPolynomialPending || out.closing {
			t.Fatal("reset caller recursion")
		}
		copied := *c
		if copied.CleanupPSCClassic() != ErrPolynomialOwner {
			t.Fatal("copied owner")
		}
		return arena.ErrBounds
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	returned := full.PSCChainClassic(p, q, 0, out)
	s := &c.pscClassic
	if !errors.Is(returned, arena.ErrBounds) || calls != 1 || s.constructed != 0 || s.cursor != 0 {
		t.Fatal("real Reset stop constructor boundary", returned, calls, s.constructed)
	}
	// Exact stop graph includes the real callback owner header and vector storage.
	known := []Polynomial{p, q, out.state.incoming}
	for i := uint32(0); uint64(i) < uint64(len(out.state.values)); i++ {
		v, e := out.state.get(c, i)
		serviceMust(t, e)
		known = append(known, v)
	}
	polynomialImpPhysicalAuditNamedRaw(t, full, known, nil, out.header+h.header, nil)
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
	serviceMust(t, c.CleanupPSCClassic())
	// The supplied vector owns its stopped Reset; the callee has no row-sweep authority.
	serviceMust(t, out.state.resetMode(c, true))
	serviceMust(t, full.PSCChainClassic(p, q, 0, out))
}

func TestPolynomialPSCClassicActualSourceAndOwnFinalizerReentry(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{1, 2, 1}, [][]MonomialPower{{{0, 4}}, {{0, 1}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.CleanupPSCClassic()); serviceMust(t, out.Close()) })
	sourceCalls, ownCalls := 0, 0
	owning := false
	h, e := NewPolynomialDeletionHandler(full.originalBudget, func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, v Polynomial) error {
		s := &c.pscClassic
		if s.executing || !s.finishing {
			return nil
		}
		if owning {
			ownCalls++
		} else {
			sourceCalls++
		}
		cursor, state, cause := s.cursor, c.memory.State(), s.cleanupCause
		if c.finishPSCClassicPhase(false, false) != ErrPolynomialPending || c.CleanupPSCClassic() != ErrPolynomialPending || s.cursor != cursor || c.memory.State() != state || s.cleanupCause != cause {
			t.Fatal("recursive destructor consumed current prefix")
		}
		return arena.ErrBounds
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	returned := full.PSCChainClassic(p, q, 0, out)
	s := &c.pscClassic
	if !errors.Is(returned, arena.ErrBounds) || sourceCalls != 1 || s.sourceCause == nil || s.cursor >= s.constructed {
		t.Fatal("actual source reference DT stop", returned, sourceCalls, s.cursor, s.constructed)
	}
	source, body := s.sourceCause, s.bodyCause
	owning = true
	own := c.CleanupPSCClassic()
	if !errors.Is(own, arena.ErrBounds) || ownCalls != 1 || s.cleanupCause != own {
		t.Fatal("actual own reference DT stop", own, ownCalls)
	}
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
	pscClassicNamedAudit(t, full, out, p, q)
	serviceMust(t, c.CleanupPSCClassic())
	if s.sourceCause != source || s.bodyCause != body || s.cleanupCause != own || !errors.Is(returned, source) {
		t.Fatal("first body/source/own history changed")
	}
}

func TestPolynomialPSCClassicCompiledConstantGuards(t *testing.T) {
	if !polynomialSourceDebug {
		return
	}
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	serviceMust(t, out.Push(c.unit))
	t.Cleanup(func() { serviceMust(t, c.CleanupPSCClassic()); serviceMust(t, out.Close()) })
	for _, pair := range [][2]Polynomial{{c.unit, p}, {p, c.unit}} {
		returned := full.PSCChainClassic(pair[0], pair[1], 0, out)
		if !errors.Is(returned, ErrPolynomialAssertion) || c.pscClassic.constructed != 0 {
			t.Fatal("assert before Reset/reference constructors", returned)
		}
		n, e := out.Size()
		serviceMust(t, e)
		v, e := out.Get(0)
		serviceMust(t, e)
		if n != 1 || v != c.unit {
			t.Fatal("guard changed caller")
		}
		serviceMust(t, c.CleanupPSCClassic())
	}
}

func TestPolynomialPSCClassicOldChildTuplePreservesLaterForeignCoefficient(t *testing.T) {
	full := polynomialArithmeticDirectObjectFixture(t)
	_, e := full.core.monomials.MkVar()
	serviceMust(t, e)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{1, 2, 1}, [][]MonomialPower{{{0, 4}}, {{0, 1}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	var old polynomialPSCClassicInvocationReceipt
	full.originalBudget.FailAfter(0)
	returned := c.pscChainClassicWithReceipt(p, q, 0, out, &old)
	s := &c.pscClassic
	if returned == nil || s.constructed != 12 || s.cursor != 12 || !s.active || !c.matchesCoefficientInvocation(&s.coefficient) {
		t.Fatal("actual stopped first lc_Q child and older source suffix", returned, s.constructed, s.cursor, s.active)
	}
	full.originalBudget.FailAfter(-1)
	serviceMust(t, c.cleanupCoefficientInvocation(&s.coefficient, true))
	serviceMust(t, c.cleanupFactoryInvocation(&s.factory, true))
	serviceMust(t, c.cheap.cleanupAppendInvocation(&s.append, true))
	var later polynomialCoefficientInvocationReceipt
	var append polynomialCheapAppendInvocationReceipt
	var factory polynomialFactoryInvocationReceipt
	full.originalBudget.FailAfter(0)
	_, foreignError := c.coeffWithReceipts(q, 0, 0, &later, &append, &factory)
	if foreignError == nil || !c.matchesCoefficientInvocation(&later) || !later.frame.active {
		t.Fatal("actual later coefficient failure absent", foreignError)
	}
	full.originalBudget.FailAfter(-1)
	before, used, body, source := *later.frame, full.originalBudget.Used(), s.bodyCause, s.sourceCause
	serviceMust(t, c.cleanupPSCClassicInvocation(&old, false))
	serviceMust(t, c.cleanupPSCClassicInvocation(&old, true))
	if *later.frame != before || full.originalBudget.Used() != used || s.active || s.bodyCause != body || s.sourceCause != source {
		t.Fatal("expired coefficient tuple consumed foreign frame")
	}
	pscClassicNamedAudit(t, full, out, p, q)
	serviceMust(t, c.cleanupCoefficientInvocation(&later, true))
	serviceMust(t, c.cleanupFactoryInvocation(&factory, true))
	serviceMust(t, c.cheap.cleanupAppendInvocation(&append, true))
	serviceMust(t, out.Close())
}
