package polynomial

// Source: src/math/polynomial/polynomial.cpp:4916-4929,6898-6922.

import (
	"testing"
)

func TestPolynomialPseudoConstantDivisorActualModDChild(t *testing.T) {
	polynomialPseudoConstantModDChild(t, 1)
}
func TestPolynomialPseudoConditionalPowerBorrowedPositionContinuation(t *testing.T) {
	polynomialPseudoConstantModDChild(t, 2)
}
func polynomialPseudoConstantModDChild(t *testing.T, degree uint32) {
	success := false
	for prefix := int64(0); prefix < 192 && !success; prefix++ {
		full := polynomialPSCFullFixture(t)
		c := full.core
		p := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, degree}}, nil})
		q := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{1, 1}}, nil})
		bounds := polynomialDegreeMap(t, c)
		serviceMust(t, bounds.SetDegree(1, degree))
		Q, R := polynomialPseudoOutputs(t, c)
		audit := func() {
			polynomialImpPhysicalAuditNamedRaw(t, full, []Polynomial{p, q, Q.root.value, Q.root.incoming, R.root.value, R.root.incoming}, nil, Q.header+R.header+bounds.header, nil)
		}
		c.integer.Budget().FailAfter(prefix)
		body := c.ExactPseudoDivideModD(p, q, 0, bounds, Q, R)
		audit()
		if c.pseudoDivision.constructed&(uint16(1)<<pseudoNumeral) != 0 {
			t.Fatal("constant divisor constructed erased minus_a NUM")
		}
		c.integer.Budget().FailAfter(-1)
		t.Log("actual prefix", prefix, "body", body, "depth", c.pseudoDivision.depth, "power", c.pseudoDivision.power.invocation, "op", c.pseudoDivision.operation.invocation, "creation", c.pending, "factory", c.factoryInvocation.active)
		if prefix == 22 {
			t.Log("actual op", c.pseudoDivision.operation, "AR", c.som.arithmeticScope, "append", c.som.appendScope, "pendingMM", c.som.pendingMonomial, "MMcreation", c.monomials.creation)
		}
		serviceMust(t, c.CleanupPseudoDivision())
		if body != nil {
			as, e := c.cheap.as.size(c)
			serviceMust(t, e)
			ms, e := c.cheap.ms.size(c)
			serviceMust(t, e)
			if as != 0 || ms != 0 || c.cheap.hasPendingReference {
				t.Fatal("actual post-Mul ModD child omitted", prefix, body, as, ms)
			}
		} else {
			success = true
			if degree == 1 {
				polynomialQueryExpect(t, c, Q.root.value, map[[3]uint32]int64{{1, 0, 0}: 2, {}: 1})
			} else {
				polynomialQueryExpect(t, c, Q.root.value, map[[3]uint32]int64{{2, 1, 0}: 8, {2, 0, 0}: 2, {0, 1, 0}: 4, {}: 1})
			}
			zero, e := c.IsZero(R.root.value)
			serviceMust(t, e)
			if !zero {
				t.Fatal("constant divisor remainder")
			}
			t.Log("actual ModD first success", prefix)
		}
		audit()
	}
	if !success {
		t.Fatal("ModD first success not reached")
	}
}
