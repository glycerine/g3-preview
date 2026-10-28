package polynomial

// Source: src/math/polynomial/polynomial.cpp:6754-6819.
// Copied Go owner metadata is a guard domain, separate from source arithmetic.

import (
	"errors"
	"testing"
)

func TestPolynomialFactorCoreActualCanceledBodyCopiedAndStaleTuple(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	p := polynomialQueryTerms(t, full.core, []int32{1, 2, 1}, [][]MonomialPower{{}, {{0, 1}}, {{0, 2}}})
	out, e := NewPolynomialFactors(full)
	serviceMust(t, e)
	full.core.config.Checkpoint = func() bool { return false }
	returned := full.Factor(p, out, DefaultUPolynomialFactorParams())
	if !errors.Is(returned, ErrUPolynomialCanceled) || !full.matchesFactorCore(&full.factorRoot) {
		t.Fatal("actual body cancellation absent", returned)
	}
	old := full.factorRoot
	s := old.frame
	if !errors.Is(s.bodyCause, ErrUPolynomialCanceled) {
		t.Fatal("actual frame body history", s.bodyCause)
	}
	polynomialFactorsAudit(t, out, []Polynomial{p}, nil)
	copiedFrame := *s
	copied := polynomialFactorCoreInvocationReceipt{full, &copiedFrame, old.invocation}
	used, source, own := full.originalBudget.Used(), s.sourceRef, s.ownRef
	serviceMust(t, full.cleanupFactorCoreInvocation(&copied, true))
	if full.matchesFactorCore(&copied) || used != full.originalBudget.Used() || source != s.sourceRef || own != s.ownRef {
		t.Fatal("copied frame consumed original fields")
	}
	serviceMust(t, full.CleanupFactor())
	oldBody, oldSource, oldOwn := s.bodyCause, s.sourceCause, s.cleanupCause
	full.core.config.Checkpoint = func() bool { return true }
	serviceMust(t, out.Reset())
	serviceMust(t, full.Factor(p, out, DefaultUPolynomialFactorParams()))
	later := full.factorRoot
	before, used := rtStateForFactors(out), full.originalBudget.Used()
	serviceMust(t, full.cleanupFactorCoreInvocation(&old, true))
	if later != full.factorRoot || before != rtStateForFactors(out) || used != full.originalBudget.Used() || s.bodyCause != oldBody || s.sourceCause != oldSource || s.cleanupCause != oldOwn || !errors.Is(returned, ErrUPolynomialCanceled) {
		t.Fatal("stale acknowledged caller adopted later output/history")
	}
	polynomialFactorAssertReconstruction(t, full, p, out, 1)
	polynomialFactorsAudit(t, out, []Polynomial{p}, nil)
	serviceMust(t, out.CloseChecked())
}
