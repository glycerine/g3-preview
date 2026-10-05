// Source: src/math/polynomial/polynomial.cpp:5940-5951,2217-2255,2481-2495.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"fmt"
	"testing"
)

func TestPolynomialNegDirectActualAppendFactoryEveryPrefix(t *testing.T) {
	success, pendingFactory := false, false
	for ordinal := int64(0); ordinal < 192 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			p := polynomialICCPUnownedTerms(t, c, []int32{7, 11}, [][]MonomialPower{{{0, 2}}, {{0, 1}}})
			p = polynomialCaller(t, c, p)
			named := []Polynomial{p}
			// Fill real cold location capacity with retained caller polynomials.
			// The next actual factory object acquisition precedes this growth.
			for len(c.locations) < cap(c.locations) {
				filler := polynomialICCPUnownedTerms(t, c, []int32{13}, [][]MonomialPower{{{0, 1}}})
				filler = polynomialCaller(t, c, filler)
				named = append(named, filler)
			}
			var receipt polynomialNegInvocationReceipt
			c.integer.budget.FailAfter(ordinal)
			result, cause := c.negWithReceipt(p, &receipt)
			audit := func() { polynomialImpPhysicalAuditNamed(t, full, append(named, result), nil, 0) }
			audit()
			if cause != nil && !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual Neg body/source suffix", cause)
			}
			as, e := c.cheap.as.size(c)
			serviceMust(t, e)
			ms, e := c.cheap.ms.size(c)
			serviceMust(t, e)
			factory := c.factoryInvocation.result
			if c.matchesFactoryInvocation(&c.negFactory) && (c.pending.raw.OwnerID() != 0 || factory != (Polynomial{})) {
				pendingFactory = true
			}
			firstCause := c.negInvocationSourceCause(&receipt)
			serviceMust(t, c.cleanupNegInvocation(&receipt, false))
			audit()
			nowAs, e := c.cheap.as.size(c)
			serviceMust(t, e)
			nowMs, e := c.cheap.ms.size(c)
			serviceMust(t, e)
			if as != nowAs || ms != nowMs || c.factoryInvocation.result != factory || c.negInvocationSourceCause(&receipt) != firstCause {
				t.Fatal("lexical owning phase consumed unjoined source row/factory or erased first cause")
			}
			own := c.cleanupNegInvocation(&receipt, true)
			if own != nil && !numericPhysicalReservationRefusal(own) {
				t.Fatal("actual matched owning suffix", own)
			}
			audit()
			c.integer.budget.FailAfter(-1)
			serviceMust(t, c.cleanupNegInvocation(&receipt, true))
			audit()
			as, e = c.cheap.as.size(c)
			serviceMust(t, e)
			ms, e = c.cheap.ms.size(c)
			serviceMust(t, e)
			if as != 0 || ms != 0 || c.cheap.hasPendingReference || c.factoryInvocation.active || c.cheap.appendScope.active || c.negativeActive {
				t.Fatal("matched Neg owning suffix retained actual rows/factory", as, ms)
			}
			if cause == nil {
				polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{2, 0, 0}: -7, {1, 0, 0}: -11})
				result = polynomialCaller(t, c, result)
				success = true
				t.Log("actual Neg first success", ordinal)
			} else if result != (Polynomial{}) {
				t.Fatal("failed source exposed raw result")
			}
		})
	}
	if !success || !pendingFactory {
		t.Fatal("actual source factory acquisition/first success not reached", success, pendingFactory)
	}
}
