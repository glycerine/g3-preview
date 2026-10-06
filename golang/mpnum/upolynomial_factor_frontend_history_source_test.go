// Source: src/math/polynomial/upolynomial.cpp:3042-3118.
// Explicit charged Go diagnostic adapter; no source NUM constructed by history.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"reflect"
	"testing"
)

func TestUPolynomialFactorFrontendActualHistoryReservation(t *testing.T) {
	_, s := preparationPhysicalFixture(t)
	m := s.manager
	fs, e := m.NewFactors()
	serviceMust(t, e)
	defer func() { serviceMust(t, fs.Close()) }()
	var input upolyVectorState
	serviceMust(t, m.set(&input, []Int{NewInt(1), NewInt(1)}))
	defer func() { serviceMust(t, m.reset(&input)); m.freeBacking(&input) }()
	yes, e := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
	serviceMust(t, e)
	if !yes || s.frontend.history == nil || s.frontend.history.header == 0 {
		t.Fatal("actual completed frontend has no charged diagnostic history")
	}
	old := s.frontend
	h := old.history
	b := m.config.Integer.budget
	b.FailAfter(0)
	_, e = s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
	if e == nil || !reflect.DeepEqual(s.frontend, old) {
		t.Fatal("actual new history reservation did not preserve prior tuple", e)
	}
	b.FailAfter(-1)
	_, e = s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
	serviceMust(t, e)
	if s.frontend.history == h || h.invocation != old.invocation || h.bodyCause != old.bodyCause || h.sourceCause != old.sourceCause || h.ownCause != old.ownCause {
		t.Fatal("later actual frontend changed prior diagnostic history")
	}
}

func TestUPolynomialFactorFrontendPriorCapturedChildACKPreservesLater(t *testing.T) {
	z, s := preparationPhysicalFixture(t)
	m := s.manager
	fs, e := m.NewFactors()
	serviceMust(t, e)
	var large Int
	serviceMust(t, z.z.SetLimbs(&large, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
	var input upolyVectorState
	serviceMust(t, m.set(&input, []Int{large, large}))
	defer func() {
		z.budget.FailAfter(-1)
		serviceMust(t, s.CleanupRetained())
		serviceMust(t, fs.Close())
		serviceMust(t, m.reset(&input))
		m.freeBacking(&input)
		serviceMust(t, z.z.Del(&large))
	}()
	z.budget.FailAfter(1)
	_, first := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
	old := s.frontend.receipt
	if first == nil || old.history.gcdRaw.invocation == 0 {
		t.Fatal("actual old retained GCD child not reached", first)
	}
	oldGCD := old.history.gcdRaw
	z.budget.FailAfter(-1)
	serviceMust(t, s.cleanupFactorFrontendInvocation(&old, false))
	serviceMust(t, s.cleanupFactorFrontendInvocation(&old, true))
	if !old.history.complete || old.history.bodyCause != first {
		t.Fatal("completed diagnostic lost original returned cause")
	}
	z.budget.FailAfter(1)
	_, later := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
	if later == nil || !z.z.matchesRawInvocation(&s.frontend.gcdRaw) || s.frontend.gcdRaw.invocation == oldGCD.invocation {
		t.Fatal("later actual foreign GCD tuple not reached", later)
	}
	prior := s.frontend
	before := z.budget.Used()
	serviceMust(t, s.cleanupFactorFrontendInvocation(&old, false))
	serviceMust(t, s.cleanupFactorFrontendInvocation(&old, true))
	if !reflect.DeepEqual(prior, s.frontend) || before != z.budget.Used() || old.history.gcdRaw != oldGCD || old.history.bodyCause != first {
		t.Fatal("old ACK changed later exact roots or original captured provenance")
	}
}
