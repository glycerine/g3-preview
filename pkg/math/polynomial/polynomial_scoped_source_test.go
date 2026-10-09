// Source: src/math/polynomial/polynomial.cpp:1863-1879,5940-5951.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialNegCompletedLocalGuardUnwind(t *testing.T) {
	m := polynomialCellFixture(t)
	terms := [2]Monomial{}
	for i := range terms {
		var e error
		terms[i], e = m.monomials.MkVariable(uint32(i), 1)
		serviceMust(t, e)
		serviceMust(t, m.monomials.IncRef(terms[i]))
	}
	coefficients := [2]util.Int{}
	serviceMust(t, m.integer.SetInt64(&coefficients[0], 1<<40))
	serviceMust(t, m.integer.SetInt64(&coefficients[1], 3<<40))
	p, e := m.MkCoreTake(coefficients[:], terms[:])
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	serviceMust(t, m.writeWord(p, 8, 3))
	result, e := m.Neg(p)
	if e != ErrMonomialOwner || result != (Polynomial{}) {
		t.Fatal("actual source-field guard not reached", result, e)
	}
	if (m.negativeCoefficient).Cell != 0 {
		t.Error("completed minus_a local survived guarded exit", m.negativeCoefficient)
	}
	serviceMust(t, m.writeWord(p, 8, 2))
	a, e := m.Coefficient(p, 0)
	serviceMust(t, e)
	b, e := m.Coefficient(p, 1)
	serviceMust(t, e)
	polynomialPhysicalAudit(t, m, &a, &b)
	serviceMust(t, m.DecRef(p))
}

func TestPolynomialCachedCallerReferencesBlockOwningClose(t *testing.T) {
	m := polynomialCellFixture(t)
	for _, p := range []Polynomial{m.zero, m.unit} {
		serviceMust(t, m.IncRef(p))
		e := m.CloseChecked()
		if e != ErrPolynomialLive {
			t.Fatalf("actual cached caller root must block owning teardown: %v", e)
		}
		if m.closing || m.closed {
			t.Fatal("rejected owning Close consumed source fields")
		}
		serviceMust(t, m.DecRef(p))
	}
	polynomialPhysicalAudit(t, m)
}
