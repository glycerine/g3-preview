package polynomial

// Source: src/math/polynomial/polynomial.cpp:3297-3312,4949.

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialCoefficientReductActualTwoBIGBufferPrefixes(t *testing.T) {
	success, sawLeading, sawReduct := false, false, false
	for prefix := int64(0); prefix < 180 && !success; prefix++ {
		full := polynomialExactDivDirectPageOneFixture(t)
		c := full.core
		for i := 0; i < 2; i++ {
			_, e := c.monomials.MkVar()
			serviceMust(t, e)
		}
		var leading, remaining unumeral.Int
		words := make([]uint32, 80)
		words[0] = 7
		words[79] = 1
		serviceMust(t, c.integer.SetLimbs(&leading, words))
		words = make([]uint32, 129)
		words[0] = 3
		words[128] = 1
		serviceMust(t, c.integer.SetLimbs(&remaining, words))
		term, e := c.monomials.MkMonomial([]MonomialPower{{0, 1}, {1, 1}})
		serviceMust(t, e)
		unit, e := c.monomials.Unit()
		serviceMust(t, e)
		p, e := c.MkTerms([]unumeral.Int{leading, remaining}, []Monomial{term, unit})
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		var reduct polynomialReference
		var receipt polynomialCoefficientInvocationReceipt
		var appendReceipt polynomialCheapAppendInvocationReceipt
		var factory polynomialFactoryInvocationReceipt
		c.integer.Budget().FailAfter(prefix)
		result, body := c.coeffCoreWithReceipts(p, 0, 1, &reduct, &receipt, &appendReceipt, &factory)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result, reduct.value, reduct.incoming}, []*unumeral.Int{&leading, &remaining}, 0)
		if body != nil && !numericPhysicalReservationRefusal(body) {
			t.Fatal("actual two BIG Copy/factory prefix", prefix, body)
		}
		if receipt.invocation != 0 {
			for i, ar := range receipt.frame.arithmetic {
				if c.matchesArithmeticInvocation(&ar) && polynomialNegNumericPending(&ar.frame.numeric) {
					if i == 0 {
						sawLeading = true
					} else {
						sawReduct = true
					}
				}
			}
		}
		serviceMust(t, c.cleanupCoefficientInvocation(&receipt, false))
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result, reduct.value, reduct.incoming}, []*unumeral.Int{&leading, &remaining}, 0)
		c.integer.Budget().FailAfter(-1)
		if body == nil {
			a, e := c.coefficientCore(result, 0)
			serviceMust(t, e)
			cmp, e := c.integer.Compare(&a, &leading)
			serviceMust(t, e)
			if cmp != 0 {
				t.Fatal("leading mathematical value", prefix)
			}
			a, e = c.coefficientCore(reduct.value, 0)
			serviceMust(t, e)
			cmp, e = c.integer.Compare(&a, &remaining)
			serviceMust(t, e)
			if cmp != 0 {
				t.Fatal("reduct mathematical value", prefix)
			}
			serviceMust(t, c.IncRef(result))
			serviceMust(t, c.transferFactoryResult(&factory, result))
			success = true
			t.Log("two BIG actual first success", prefix)
		}
		serviceMust(t, reduct.closeValue(c, true))
		serviceMust(t, reduct.abort(c))
		serviceMust(t, c.cleanupCoefficientInvocation(&receipt, true))
		serviceMust(t, c.cleanupFactoryInvocation(&factory, true))
		serviceMust(t, c.cheap.cleanupAppendInvocation(&appendReceipt, true))
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result}, []*unumeral.Int{&leading, &remaining}, 0)
		if body == nil {
			serviceMust(t, c.DecRef(result))
		}
		serviceMust(t, c.DecRef(p))
		serviceMust(t, c.integer.Del(&leading))
		serviceMust(t, c.integer.Del(&remaining))
	}
	if !success || !sawLeading || !sawReduct {
		t.Fatal("actual both buffer numeric acquisitions/horizon", success, sawLeading, sawReduct)
	}
}
