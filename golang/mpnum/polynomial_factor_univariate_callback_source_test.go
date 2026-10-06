// Source: src/math/polynomial/polynomial.cpp:6683-6725.
package mpnum

import "testing"

func TestPolynomialFactorUnivariateCheckpointProtectsOutputOwner(t *testing.T) {
	for _, operation := range []string{"source", "close", "cleanup", "fullSource", "fullClose", "typedSource", "typedOwn"} {
		t.Run(operation, func(t *testing.T) {
			full := polynomialPSCFullFixtureWithLimit(t, 2<<20)
			primes, e := NewPrimeGenerator(full.core.integer.budget)
			serviceMust(t, e)
			full.factorConfig = UPolynomialFactorizationConfig{Primes: primes}
			p := polynomialQueryTerms(t, full.core, []int32{-1, 1}, [][]MonomialPower{{}, {{0, 2}}})
			out, e := NewPolynomialFactors(full)
			serviceMust(t, e)
			seen := false
			var receipt polynomialFactorUnivariateInvocationReceipt
			full.upm.config.Checkpoint = func() bool {
				if full.factorUnivariate.executing && full.factorUnivariate.fs != nil && !seen {
					seen = true
					before := rtStateForFactors(out)
					used := full.originalBudget.Used()
					cursor := full.factorUnivariate.sourceCursor
					var cause error
					switch operation {
					case "source":
						cause = out.SourceFinalize()
					case "close":
						cause = out.CloseChecked()
					case "cleanup":
						cause = out.Cleanup()
					case "fullSource":
						cause = full.SourceFinalize()
					case "fullClose":
						cause = full.CloseChecked()
					case "typedSource":
						cause = full.finishFactorUnivariateSource(&receipt)
					case "typedOwn":
						cause = full.cleanupFactorUnivariateInvocation(&receipt, true)
					}
					if cause != ErrPolynomialPending || rtStateForFactors(out) != before || used != full.originalBudget.Used() || full.factorUnivariate.sourceCursor != cursor {
						t.Errorf("actual checkpoint retired output owner: %v", cause)
					}
				}
				return true
			}
			serviceMust(t, full.factorSquareFreePrimitiveUnivariateWithReceipt(p, out, 1, DefaultUPolynomialFactorParams(), &receipt))
			if !seen {
				t.Fatal("actual backend checkpoint not reached")
			}
			serviceMust(t, full.cleanupFactorUnivariateInvocation(&receipt, false))
			serviceMust(t, full.cleanupFactorUnivariateInvocation(&receipt, true))
			serviceMust(t, out.CloseChecked())
			t.Cleanup(func() { serviceMust(t, primes.Close()) })
		})
	}
}
