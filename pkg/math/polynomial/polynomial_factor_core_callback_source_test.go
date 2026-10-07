// Source: src/math/polynomial/polynomial.cpp:6754-6819.
package polynomial

import (
	"errors"
	"testing"
)

func TestPolynomialFactorCoreCheckpointPreservesActualCallerAndOutput(t *testing.T) {
	for _, operation := range []string{"source", "close", "cleanup", "outputSource", "outputClose", "outputCleanup", "typedSource", "typedOwn", "typedBody", "factor"} {
		t.Run(operation, func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			p := polynomialQueryTerms(t, full.core, []int32{1, 2, 1}, [][]MonomialPower{{}, {{0, 1}}, {{0, 2}}})
			out, e := NewPolynomialFactors(full)
			serviceMust(t, e)
			seen := false
			full.core.config.Checkpoint = func() bool {
				if !seen && full.matchesFactorCore(&full.factorRoot) && full.factorRoot.frame.executing {
					seen = true
					s := full.factorRoot.frame
					before, used := rtStateForFactors(out), full.originalBudget.Used()
					refs, source, own := s.references, s.sourceRef, s.ownRef
					closing, stage := full.closing, full.cleanupStage
					var cause error
					switch operation {
					case "source":
						cause = full.SourceFinalize()
					case "close":
						cause = full.CloseChecked()
					case "cleanup":
						cause = full.CleanupFactor()
					case "outputSource":
						cause = out.SourceFinalize()
					case "outputClose":
						cause = out.CloseChecked()
					case "outputCleanup":
						cause = out.Cleanup()
					case "typedSource":
						cause = full.finishFactorCoreSource(&full.factorRoot)
					case "typedOwn":
						cause = full.cleanupFactorCoreInvocation(&full.factorRoot, true)
					case "typedBody":
						var foreign polynomialFactorCoreInvocationReceipt
						cause = full.factorCoreWithReceipt(p, out, DefaultUPolynomialFactorParams(), &foreign)
					case "factor":
						cause = full.Factor(p, out, DefaultUPolynomialFactorParams())
					}
					if !errors.Is(cause, ErrPolynomialPending) || before != rtStateForFactors(out) || used != full.originalBudget.Used() || refs != s.references || source != s.sourceRef || own != s.ownRef || closing != full.closing || stage != full.cleanupStage {
						t.Errorf("reentered executing Factor mutated caller/output: %v", cause)
					}
				}
				return true
			}
			serviceMust(t, full.Factor(p, out, DefaultUPolynomialFactorParams()))
			if !seen {
				t.Fatal("actual Factor checkpoint absent")
			}
			polynomialFactorAssertReconstruction(t, full, p, out, 1)
			polynomialFactorsAudit(t, out, []Polynomial{p}, nil)
			serviceMust(t, out.CloseChecked())
		})
	}
}
