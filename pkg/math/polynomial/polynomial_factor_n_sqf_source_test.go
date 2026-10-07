// Source: src/math/polynomial/polynomial.cpp:6727-6735.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"fmt"
	"testing"
)

func TestPolynomialHigherFactorSourceTODOKeepsOriginalFactor(t *testing.T) {
	for k := uint32(1); k <= 4; k++ {
		m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{3, 0, 0}}, {1, [3]uint32{0, 1, 0}}})
		f, e := NewPolynomialFactors(m)
		serviceMust(t, e)
		before, e := m.core.readWord(p, 0)
		serviceMust(t, e)
		serviceMust(t, m.factorHigherSquareFreePrimitive(p, f, 0, k))
		n, e := f.Distinct()
		serviceMust(t, e)
		v, e := f.factor(0)
		serviceMust(t, e)
		degree, e := f.Degree(0)
		serviceMust(t, e)
		after, e := m.core.readWord(p, 0)
		serviceMust(t, e)
		if n != 1 || v != p || degree != k || after != before+1 {
			t.Fatal("literal source TODO pushes original rather than substitute factorization", n, v, p, degree, k, after, before)
		}
		polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
		serviceMust(t, f.CloseChecked())
		after, e = m.core.readWord(p, 0)
		serviceMust(t, e)
		if after != before {
			t.Fatal("actual factor INC not consumed", after, before)
		}
	}
}

func TestPolynomialHigherFactorDegreeAssertionPrecedesPublication(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {1, [3]uint32{}}})
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, f.CloseChecked()) }()
	budget, raw := m.core.integer.Budget().Used(), m.core.memory.State()
	e = m.factorHigherSquareFreePrimitive(p, f, 0, 2)
	if polynomialSourceDebug {
		if e != ErrPolynomialAssertion || m.core.integer.Budget().Used() != budget || m.core.memory.State() != raw {
			t.Fatal("literal degree>2 assertion frontier", e)
		}
		n, e := f.Distinct()
		serviceMust(t, e)
		if n != 0 {
			t.Fatal("failed assertion published factor", n)
		}
	} else {
		serviceMust(t, e)
		v, e := f.factor(0)
		serviceMust(t, e)
		if v != p {
			t.Fatal("compiled-erased assertion source push", v, p)
		}
	}
	polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
}

func TestPolynomialHigherFactorEveryReachedAssertionAndPushPrefix(t *testing.T) {
	for ordinal := int64(0); ordinal < 1024; ordinal++ {
		completed := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{3, 0, 0}}, {1, [3]uint32{0, 1, 0}}})
			f, e := NewPolynomialFactors(m)
			serviceMust(t, e)
			before, e := m.core.readWord(p, 0)
			serviceMust(t, e)
			m.core.integer.Budget().FailAfter(ordinal)
			e = m.factorHigherSquareFreePrimitive(p, f, 0, 3)
			polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual source assertion/Push prefix", ordinal, e)
			}
			// These source assertions call the actual public predicate engines;
			// their lexical frames remain owned by the full manager on refusal.
			// This helper constructs no additional polynomial or numeral local.
			m.core.integer.Budget().FailAfter(-1)
			serviceMust(t, m.CleanupGCD())
			if e == nil {
				v, x := f.factor(0)
				serviceMust(t, x)
				degree, x := f.Degree(0)
				serviceMust(t, x)
				if v != p || degree != 3 {
					t.Fatal("literal source original-factor handoff", v, p, degree)
				}
				completed = true
			}
			serviceMust(t, f.Cleanup())
			polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
			serviceMust(t, f.CloseChecked())
			after, x := m.core.readWord(p, 0)
			serviceMust(t, x)
			if after != before {
				t.Fatal("actual factor acquisition/owning abort changed borrowed caller", ordinal, after, before)
			}
		})
		if completed {
			t.Log("first complete actual source prefix", ordinal)
			return
		}
	}
	t.Fatal("actual source assertion/Push campaign never completed")
}
