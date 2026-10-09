// Source: src/math/polynomial/polynomial.cpp:5940-5951,2217-2255,2481-2495.
package polynomial

import (
	"fmt"
	"testing"
)

// A warm scalar Neg may now succeed at ordinal zero: its locator directory
// uses native append, and it needs no new numeric or pooled-object allocation.
func TestPolynomialNegDirectActualAppendFactoryEveryPrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 192 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			p := polynomialICCPUnownedTerms(t, c, []int32{7, 11}, [][]MonomialPower{{{0, 2}}, {{0, 1}}})
			p = polynomialCaller(t, c, p)
			named := []Polynomial{p}
			var receipt polynomialNegInvocationReceipt
			c.integer.Budget().FailAfter(ordinal)
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
			c.integer.Budget().FailAfter(-1)
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
	if !success {
		t.Fatal("actual source first success not reached")
	}
}
