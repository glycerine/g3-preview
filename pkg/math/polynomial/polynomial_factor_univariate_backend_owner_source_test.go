// Source: src/math/polynomial/polynomial.cpp:6694-6724;
// src/math/polynomial/upolynomial.h:401-423.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialFactorUnivariateActualBackendOutputCloseRefusal(t *testing.T) {
	full := polynomialPSCFullFixtureWithLimit(t, 2<<20)
	primes, e := util.NewPrimeGenerator(full.originalBudget)
	serviceMust(t, e)
	full.factorConfig = UPolynomialFactorizationConfig{Primes: primes}
	p := polynomialQueryTerms(t, full.core, []int32{-1, 1}, [][]MonomialPower{{}, {{0, 2}}})
	out, e := NewPolynomialFactors(full)
	serviceMust(t, e)
	unrelated, e := full.upm.NewFactors()
	serviceMust(t, e)
	seen := false
	full.upm.config.Checkpoint = func() bool {
		s := &full.factorUnivariate
		if !seen && s.executing && s.fs != nil {
			seen = true
			fs := s.fs
			used, count, constant, live := full.originalBudget.Used(), fs.count, fs.constant, fs.manager.liveFactors
			closing, cleared := fs.closing, fs.closeCleared
			cause := fs.Close()
			if cause != ErrPolynomialPending || fs.closed || fs.closing != closing || fs.closeCleared != cleared || used != full.originalBudget.Used() || count != fs.count || constant != fs.constant || live != fs.manager.liveFactors {
				t.Errorf("actual passed backend output retired: %v", cause)
			}
			serviceMust(t, unrelated.Close())
		}
		return true
	}
	serviceMust(t, full.Factor(p, out, DefaultUPolynomialFactorParams()))
	if !seen || !unrelated.closed {
		t.Fatal("actual backend output checkpoint or unrelated-owner close absent")
	}
	polynomialFactorAssertReconstruction(t, full, p, out, 2)
	polynomialFactorsAuditExtra(t, out, []Polynomial{p}, nil, primes.Stats().TotalBytes)
	serviceMust(t, out.CloseChecked())
	t.Cleanup(func() { serviceMust(t, primes.Close()) })
}
