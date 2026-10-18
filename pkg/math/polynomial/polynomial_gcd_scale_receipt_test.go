// Source: polynomial.cpp, gcd_content and modular/recursive scalar products.
package polynomial

import "testing"

func TestPolynomialGCDContentScaleReceiptReservationPrefixes(t *testing.T) {
	success, captured := false, false
	for prefix := int64(0); prefix < 256 && !success; prefix++ {
		full := polynomialSparseDirectFactoryFixture(t)
		c := full.core
		p := polynomialQueryTerms(t, c, []int32{6, 9}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{1, 1}}})
		q := polynomialQueryTerms(t, c, []int32{3}, [][]MonomialPower{{{1, 1}}})
		out, e := NewPolynomialRef(c)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, full.CleanupGCD()); serviceMust(t, out.Close()) })
		full.originalBudget.FailAfter(prefix)
		body := full.gcdContent(p, 0, q, &out.root)
		full.originalBudget.FailAfter(-1)
		if body != nil && !numericPhysicalReservationRefusal(body) {
			t.Fatal("unexpected source refusal", prefix, body)
		}
		if len(full.gcdFrames) > 0 {
			r := full.gcdFrames[0].directArithmetic
			if body != nil && c.matchesArithmeticInvocation(&r) {
				captured = true
				if r.frame.buffer != &c.cheap {
					t.Fatal("scale receipt lost actual buffer identity")
				}
				serviceMust(t, c.cleanupArithmeticInvocation(&r, false))
				serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
				n, e := c.cheap.ms.size(c)
				serviceMust(t, e)
				if n != 0 || c.cheap.hasPendingReference {
					t.Fatal("exact scale cleanup retained source rows", prefix, n)
				}
			}
		}
		serviceMust(t, full.CleanupGCD())
		if body == nil {
			success = true
			polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{0, 1, 0}: 3})
			t.Log("actual first success", prefix)
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, out.root.value}, nil, out.header)
	}
	if !success || !captured {
		t.Fatal("missed completed source or retained scale receipt", success, captured)
	}
}
