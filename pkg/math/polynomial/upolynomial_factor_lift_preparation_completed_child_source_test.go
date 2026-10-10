// Source: src/math/polynomial/upolynomial_factorization.cpp:1153-1175.
package polynomial

import (
	"fmt"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialFactorLiftPreparationCompletedChildPrecedesOlderSource(t *testing.T) {
	complete := false
	for ordinal := int64(0); ordinal < 512; ordinal++ {
		done := false
		okay := t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			z, s := preparationPhysicalFixture(t)
			m := s.manager
			out, e := m.NewFactors()
			serviceMust(t, e)
			defer func() { z.Budget().FailAfter(-1); serviceMust(t, s.CleanupRetained()); serviceMust(t, out.Close()) }()
			c, e := s.newSquareFreeScope()
			serviceMust(t, e)
			c.constructed = 1
			serviceMust(t, m.set(&c.fpp, []unumeral.Int{unumeral.NewInt(-6), unumeral.NewInt(1), unumeral.NewInt(1)}))
			serviceMust(t, s.preparePrimeSearch(out))
			// Compiled eligible post-loop prime profile, using the real trial engine.
			serviceMust(t, m.modular.SetInt32(&c.p, 7))
			serviceMust(t, c.zp.SetZP(&c.p))
			trials := uint32(0)
			route, e := s.factorPrimeTrial(c.zp, &c.fpp, out, c.best, 1, &c.p, &c.bestP, &c.ds, &trials)
			serviceMust(t, e)
			if route != 1 || c.best.count != 2 || c.bestP != unumeral.NewInt(7) {
				t.Fatal("actual saved-factor setup", route, c.best.count, c.bestP)
			}
			z.Budget().FailAfter(ordinal)
			body := s.prepareFactorLift(out)
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{out})
			receipt := c.primePreparation.receipt

			if body != nil && c.hensel.active && c.hensel.sourceCause == nil && c.primePreparation.sourceCause == nil {
				serviceMust(t, c.hensel.finish())
				oldNonce, oldBody := c.hensel.invocation, c.hensel.bodyCause
				if c.hensel.outerSource != 0 || c.hensel.quadraticSource != 0 || c.hensel.iterationSource != 0 {
					t.Fatal("child source not complete")
				}
				source := c.cleanup()
				if c.constructed == 9 && c.primePreparation.factorSource[1] == (upolyPreparedFactorCursor{}) {
					t.Fatal("completed child blocked older source class before owning abort", ordinal, source)
				}
				if c.hensel.invocation != oldNonce || c.hensel.bodyCause != oldBody {
					t.Fatal("older source changed child tuple")
				}
				primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{out})
				done = true
			}
			serviceMust(t, c.primePreparation.own(&receipt, false))
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{out})
			z.Budget().FailAfter(-1)
			serviceMust(t, s.CleanupRetained())
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{out})
		})
		if !okay {
			return
		}
		if done {
			t.Log("actual completed-child/older-source refusal prefix", ordinal)
			complete = true
			break
		}
	}
	if !complete {
		t.Fatal("no actual completed-child owning retention")
	}
}
