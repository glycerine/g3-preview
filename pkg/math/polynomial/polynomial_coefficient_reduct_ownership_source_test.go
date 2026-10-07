// Source: src/math/polynomial/polynomial.cpp:3297-3312,4905-4959.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// The literal coeff(B,x,degB,r_B) call has two real cheap buffers. The
// caller's reduct reference is older than any unjoined child row/factory.
func TestPolynomialCoefficientReductActualTwoBufferOwnershipPrefixes(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 128 && !success; prefix++ {
		full := polynomialPSCFullFixture(t)
		c := full.core
		for i := 0; i < 2; i++ {
			_, e := c.monomials.MkVar()
			serviceMust(t, e)
		}
		term, e := c.monomials.MkMonomial([]MonomialPower{{0, 1}, {1, 1}})
		serviceMust(t, e)
		unit, e := c.monomials.Unit()
		serviceMust(t, e)
		p, e := c.MkTerms([]util.Int{util.NewInt(2), util.NewInt(1)}, []Monomial{term, unit})
		serviceMust(t, e)
		p = polynomialCaller(t, c, p)
		var reduct polynomialReference
		var coefficient polynomialCoefficientInvocationReceipt
		var appendReceipt polynomialCheapAppendInvocationReceipt
		var factory polynomialFactoryInvocationReceipt
		c.integer.Budget().FailAfter(prefix)
		result, body := c.coeffCoreWithReceipts(p, 0, 1, &reduct, &coefficient, &appendReceipt, &factory)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result, reduct.value, reduct.incoming}, nil, 0)
		if body != nil && !numericPhysicalReservationRefusal(body) {
			t.Fatal("reached literal coeff body", prefix, body)
		}
		c.integer.Budget().FailAfter(-1)
		if body == nil {
			success = true
			serviceMust(t, c.IncRef(result))
			serviceMust(t, c.transferFactoryResult(&factory, result))
		}
		serviceMust(t, reduct.closeValue(c, true))
		serviceMust(t, reduct.abort(c))
		serviceMust(t, c.cleanupCoefficientInvocation(&coefficient, true))
		serviceMust(t, c.cleanupFactoryInvocation(&factory, true))
		serviceMust(t, c.cheap.cleanupAppendInvocation(&appendReceipt, true))
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result}, nil, 0)
		for _, b := range []*polynomialSOM{&c.cheap, &c.cheap2} {
			as, e := b.as.size(c)
			serviceMust(t, e)
			ms, e := b.ms.size(c)
			serviceMust(t, e)
			if as != 0 || ms != 0 || b.hasPendingReference {
				t.Fatalf("reached two-buffer child absent from exact owning continuation prefix=%d body=%v buffer=%v as=%d ms=%d", prefix, body, b == &c.cheap2, as, ms)
			}
		}
		if body == nil {
			serviceMust(t, c.DecRef(result))
			t.Log("actual first success", prefix)
		}
	}
	if !success {
		t.Fatal("actual two-buffer first success not reached")
	}
}
