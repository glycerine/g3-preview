// Source: src/math/polynomial/polynomial.cpp:4960-5029.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "testing"

func TestPolynomialPseudoActualRAndQBufferChildren(t *testing.T) {
	polynomialPseudoBufferChildPrefixes(t, 1, false)
}
func TestPolynomialPseudoActualGeneratedMonomialChild(t *testing.T) {
	polynomialPseudoBufferChildPrefixes(t, 3, false)
}
func TestPolynomialPseudoActualVerificationChildren(t *testing.T) {
	polynomialPseudoBufferChildPrefixes(t, 3, true)
}
func polynomialPseudoBufferChildPrefixes(t *testing.T, degree uint32, verification bool) {
	complete := false
	for prefix := int64(0); prefix < 192 && !complete; prefix++ {
		full := polynomialPSCFullFixture(t)
		c := full.core
		p := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, degree}}, nil})
		q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
		Q, R := polynomialPseudoOutputs(t, c)
		var d uint32
		c.integer.budget.FailAfter(prefix)
		var body error
		if verification {
			body = c.PseudoRemainder(p, q, 0, &d, R)
		} else {
			body = c.PseudoDivide(p, q, 0, &d, Q, R)
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, Q.root.value, Q.root.incoming, R.root.value, R.root.incoming}, nil, Q.header+R.header)
		c.integer.budget.FailAfter(-1)
		serviceMust(t, c.CleanupPseudoDivision())
		if body != nil {
			if degree == 3 && (c.generatedActive || c.generatedCalleePending) {
				t.Fatal("actual generated factory child omitted", prefix, body, c.generatedActive, c.generatedCalleePending)
			}
			for _, buf := range []*polynomialSOM{&c.som, &c.som2} {
				as, e := buf.as.size(c)
				serviceMust(t, e)
				ms, e := buf.ms.size(c)
				serviceMust(t, e)
				if as != 0 || ms != 0 || buf.hasPendingReference {
					t.Fatal("actual parent omits R/Q source child", prefix, body, as, ms, buf == &c.som2)
				}
			}
		} else {
			complete = true
			expectedQ := polynomialQueryTerms(t, c, []int32{2}, [][]MonomialPower{nil})
			if degree == 3 {
				expectedQ = polynomialQueryTerms(t, c, []int32{2, -2, 2}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
			}
			expectedR := polynomialQueryTerms(t, c, []int32{-1}, [][]MonomialPower{nil})
			if !verification {
				eq, e := c.Equal(Q.root.value, expectedQ)
				serviceMust(t, e)
				if !eq {
					t.Fatal("literal quotient")
				}
			}
			eq, e := c.Equal(R.root.value, expectedR)
			serviceMust(t, e)
			if !eq {
				t.Fatal("literal remainder")
			}
			if d != degree {
				t.Fatal("literal iteration count", d)
			}
			t.Log("actual first complete R/Q", prefix)
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, Q.root.value, Q.root.incoming, R.root.value, R.root.incoming}, nil, Q.header+R.header)
	}
	if !complete {
		t.Fatal("source buffer first success not reached")
	}
}

func TestPolynomialPseudoMatchedAbortPreservesForeignQuotientRow(t *testing.T) {
	proof := false
	for prefix := int64(0); prefix < 160 && !proof; prefix++ {
		full := polynomialPSCFullFixture(t)
		c := full.core
		p := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}}, nil})
		q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
		foreign := polynomialQueryTerms(t, c, []int32{9}, [][]MonomialPower{{{2, 1}}})
		term, e := c.monomialCore(foreign, 0)
		serviceMust(t, e)
		Q, R := polynomialPseudoOutputs(t, c)
		var d uint32
		c.integer.budget.FailAfter(prefix)
		body := c.PseudoDivide(p, q, 0, &d, Q, R)
		audit := func() {
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, foreign, Q.root.value, Q.root.incoming, R.root.value, R.root.incoming}, nil, Q.header+R.header)
		}
		audit()
		as, e := c.som2.as.size(c)
		serviceMust(t, e)
		ms, e := c.som2.ms.size(c)
		serviceMust(t, e)
		child := c.pseudoDivision.quotientBuffer
		c.integer.budget.FailAfter(-1)
		var count uint32
		if body != nil && as > 0 && as == ms && !c.som2.hasPendingReference && c.matchesArithmeticInvocation(&child) {
			seven := NewInt(7)
			serviceMust(t, c.som2.add(&seven, term, false))
			location, e := c.monomials.location(term)
			serviceMust(t, e)
			count, e = c.monomials.allocator.ReadUint32(location.raw, 0)
			serviceMust(t, e)
			audit()
			proof = true
		}
		serviceMust(t, c.pseudoCleanupPhase(true, false))
		serviceMust(t, c.pseudoCleanupPhase(true, true))
		if proof {
			as, e = c.som2.as.size(c)
			serviceMust(t, e)
			ms, e = c.som2.ms.size(c)
			serviceMust(t, e)
			value, e := c.som2.as.coefficient(c, 0)
			serviceMust(t, e)
			actual, e := c.som2.ms.monomial(c, 0)
			serviceMust(t, e)
			location, e := c.monomials.location(term)
			serviceMust(t, e)
			after, e := c.monomials.allocator.ReadUint32(location.raw, 0)
			serviceMust(t, e)
			if as != 1 || ms != 1 || actual != term || value != NewInt(7) || after != count {
				t.Fatal("matched Pseudo cleanup changed actual foreign row", prefix, as, ms, value, actual, after, count)
			}
			audit()
			serviceMust(t, c.som2.reset())
			t.Log("actual foreign boundary", prefix)
		}
		audit()
	}
	if !proof {
		t.Fatal("actual published quotient row failure not reached")
	}
}
