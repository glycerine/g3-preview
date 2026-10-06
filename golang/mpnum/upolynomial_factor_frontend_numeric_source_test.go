// Source: src/math/polynomial/upolynomial.cpp:3042-3057,213-232;
// src/util/mpz.cpp:560-843.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"fmt"
	"testing"
)

func TestUPolynomialFactorFrontendActualPrimitiveGCDCapture(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 24 && !reached; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			z, s := preparationPhysicalFixture(t)
			m := s.manager
			var large Int
			serviceMust(t, z.z.SetLimbs(&large, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
			fs, e := m.NewFactors()
			serviceMust(t, e)
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
			z.budget.FailAfter(ordinal)
			_, body := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
			caller := []*Int{&large}
			for i := range m.values(&input) {
				caller = append(caller, &m.values(&input)[i])
			}
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, caller...)
			if body != nil && z.z.gcdRaw.active {
				reached = true
				if !s.frontend.active {
					t.Fatal("frontend retired while actual captured RAW body remains unjoined")
				}
				r := s.frontend.gcdRaw
				if r.manager != z.z || r.frame != &z.z.gcdRaw || r.invocation != z.z.gcdRaw.invocation {
					t.Fatalf("actual primitive GCD RAW child not captured: ordinal=%d body=%v actual=%d captured=%d", ordinal, body, z.z.gcdRaw.invocation, r.invocation)
				}
				receipt := s.frontend.receipt
				firstBody, firstSource := s.frontend.bodyCause, s.frontend.sourceCause
				stop := s.cleanupFactorFrontendInvocation(&receipt, false)
				if stop != nil && !numericPhysicalReservationRefusal(stop) {
					t.Fatal("actual false child phase", stop)
				}
				primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, caller...)
				if stop == nil {
					stop = s.cleanupFactorFrontendInvocation(&receipt, true)
				}
				if stop != nil && !numericPhysicalReservationRefusal(stop) {
					t.Fatal("actual true child phase", stop)
				}
				primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, caller...)
				if s.frontend.bodyCause != firstBody || s.frontend.sourceCause != firstSource || s.frontend.gcdRaw != r {
					t.Fatal("before-lift owning phases changed original cause/tuple")
				}

			}
		})
	}
	if !reached {
		t.Fatal("actual primitive GCD retained source prefix not reached")
	}
}
