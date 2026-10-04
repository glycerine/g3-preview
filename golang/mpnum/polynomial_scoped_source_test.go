// Source: src/math/polynomial/polynomial.cpp:1863-1879,5940-5951. Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

// The endangered third physical read is reached after the real big GCD was
// published. The completed default scoped numeral must unwind on that guard;
// this is not a failed numeral constructor or a fabricated free refusal.
func TestPolynomialNormalizeCompletedLocalGuardUnwind(t *testing.T) {
	m := polynomialCellFixture(t)
	b := &m.som
	for _, v := range []int64{1 << 40, 3 << 40} {
		serviceMust(t, b.as.appendCoefficient(m, Int{}))
		i, e := b.as.size(m)
		serviceMust(t, e)
		a, e := b.as.coefficient(m, i-1)
		serviceMust(t, e)
		serviceMust(t, m.integer.SetInt64(&a, v))
		serviceMust(t, b.as.putCoefficient(m, i-1, a))
	}
	serviceMust(t, b.as.publishSize(m, 3))
	e := b.normalize()
	if e != arena.ErrBounds {
		t.Fatal("actual third field guard", e)
	}
	if b.gcd.cell != 0 {
		t.Error("completed source scoped GCD survived body guard", b.gcd)
	}
	// No ownership repair: restore only the deliberately endangered SIZE after
	// checking the scoped destructor result, then independently census roots.
	serviceMust(t, b.as.publishSize(m, 2))
	polynomialPhysicalAudit(t, m)
}

func TestPolynomialNegCompletedLocalGuardUnwind(t *testing.T) {
	m := polynomialCellFixture(t)
	terms := [2]Monomial{}
	for i := range terms {
		var e error
		terms[i], e = m.monomials.MkVariable(uint32(i), 1)
		serviceMust(t, e)
		serviceMust(t, m.monomials.IncRef(terms[i]))
	}
	coefficients := [2]Int{}
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
	if m.negativeCoefficient.cell != 0 {
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
