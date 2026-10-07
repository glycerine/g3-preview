// Source: src/math/polynomial/upolynomial.cpp:3042-3118.
// Explicit charged Go diagnostic adapter; no source NUM constructed by history.
package polynomial

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialFactorFrontendActualHistoryReservation(t *testing.T) {
	_, s := preparationPhysicalFixture(t)
	m := s.manager
	fs, e := m.NewFactors()
	serviceMust(t, e)
	defer func() { serviceMust(t, fs.Close()) }()
	var input upolyVectorState
	serviceMust(t, m.set(&input, []util.Int{util.NewInt(1), util.NewInt(1)}))
	defer func() { serviceMust(t, m.reset(&input)); m.freeBacking(&input) }()
	yes, e := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
	serviceMust(t, e)
	if !yes || s.frontend.history == nil || s.frontend.history.header == 0 {
		t.Fatal("actual completed frontend has no charged diagnostic history")
	}
	old := s.frontend
	h := old.history
	b := m.config.Integer.Budget()
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
	var large util.Int
	serviceMust(t, z.Integers().SetLimbs(&large, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
	var input upolyVectorState
	serviceMust(t, m.set(&input, []util.Int{large, large}))
	defer func() {
		z.Budget().FailAfter(-1)
		serviceMust(t, s.CleanupRetained())
		serviceMust(t, fs.Close())
		serviceMust(t, m.reset(&input))
		m.freeBacking(&input)
		serviceMust(t, z.Integers().Del(&large))
	}()
	z.Budget().FailAfter(1)
	_, first := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
	old := s.frontend.receipt
	if first == nil || old.history.gcdRaw.Invocation == 0 {
		t.Fatal("actual old retained GCD child not reached", first)
	}
	oldGCD := old.history.gcdRaw
	z.Budget().FailAfter(-1)
	serviceMust(t, s.cleanupFactorFrontendInvocation(&old, false))
	serviceMust(t, s.cleanupFactorFrontendInvocation(&old, true))
	if !old.history.complete || old.history.bodyCause != first {
		t.Fatal("completed diagnostic lost original returned cause")
	}
	z.Budget().FailAfter(1)
	_, later := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
	if later == nil || !z.Integers().MatchesRawInvocation(&s.frontend.gcdRaw) || s.frontend.gcdRaw.Invocation == oldGCD.Invocation {
		t.Fatal("later actual foreign GCD tuple not reached", later)
	}
	prior := s.frontend
	before := z.Budget().Used()
	serviceMust(t, s.cleanupFactorFrontendInvocation(&old, false))
	serviceMust(t, s.cleanupFactorFrontendInvocation(&old, true))
	if !reflect.DeepEqual(prior, s.frontend) || before != z.Budget().Used() || old.history.gcdRaw != oldGCD || old.history.bodyCause != first {
		t.Fatal("old ACK changed later exact roots or original captured provenance")
	}
}
