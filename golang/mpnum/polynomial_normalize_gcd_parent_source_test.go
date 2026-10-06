// Source: src/math/polynomial/polynomial.cpp:4235-4350,5904-5940.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "testing"

func TestPolynomialNormalizeGCDParentRetainsActualStoppedChild(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	var a, b Int
	aw, bw := make([]uint32, 80), make([]uint32, 77)
	aw[0], aw[79] = 6, 1<<29
	bw[0], bw[76] = 10, 1<<27
	serviceMust(t, c.integer.SetLimbs(&a, aw))
	serviceMust(t, c.integer.SetLimbs(&b, bw))
	t.Cleanup(func() { serviceMust(t, c.integer.Del(&a)); serviceMust(t, c.integer.Del(&b)) })
	x, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	p, e := c.MkTerms([]Int{a, b}, []Monomial{x, c.monomials.unit})
	serviceMust(t, e)
	p = polynomialCaller(t, c, p)
	s, e := full.beginGCDFunction()
	serviceMust(t, e)
	s.constructReference(c, 0)
	serviceMust(t, s.refs[0].root.assign(c, p))
	full.originalBudget.FailAfter(0)
	_, returned := full.normalizeWithReceipt(p, &s.normalizeInvocation)
	if !numericPhysicalReservationRefusal(returned) || !full.normalization.active {
		t.Fatal("actual reached Normalize refusal", returned)
	}
	child := s.normalizeInvocation
	s.bodyCause = returned
	serviceMust(t, full.finishGCDFunctionSource(s))
	if !s.active || s.normalizeInvocation != child {
		t.Error("source parent erased actual unfinished Normalize child")
	}
	full.gcdDepth--
	c.pscLeave()
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, []*Int{&a, &b}, 0)
	serviceMust(t, full.CleanupGCD())
	if full.normalization.active || s.normalizeInvocation != child || !full.hasNormalizeInvocationHistory(&s.normalizeInvocation) {
		t.Error("own parent omitted exact child or erased child history")
	}
	full.originalBudget.FailAfter(-1)
	// RED-only recovery keeps teardown diagnostics useful without changing the oracle.
	serviceMust(t, full.CleanupNormalization())
}
