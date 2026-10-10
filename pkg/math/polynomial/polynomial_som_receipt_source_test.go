// Source: src/math/polynomial/polynomial.cpp:2020-2080,2159-2214.
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialSOMStoppedCalleeReplacementIdentity(t *testing.T) {
	m := polynomialCellDirectObjectFixture(t)
	v, e := NewPolynomialSOMVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	b, e := v.At(0)
	serviceMust(t, e)
	x, e := m.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	p, e := m.MkTerms([]unumeral.Int{unumeral.NewInt(3)}, []Monomial{x})
	serviceMust(t, e)
	polynomialCaller(t, m, p)
	a := unumeral.NewInt(2)
	m.integer.Budget().FailAfter(0)
	e = b.AddMul(&a, x, p)
	if !numericPhysicalReservationRefusal(e) || !b.calleeCall {
		t.Fatal("actual failed product factory", e, b.calleeCall)
	}
	polynomialSOMPhysicalAudit(t, m, v, []Monomial{x}, &a)
	m.integer.Budget().FailAfter(-1)
	serviceMust(t, m.monomials.CleanupCreation())
	m.integer.Budget().FailAfter(0)
	_, e = m.monomials.MkMonomial([]MonomialPower{{0, 31}, {1, 37}, {2, 41}})
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal("replacement real factory", e)
	}
	before := m.monomials.creation
	polynomialSOMPhysicalAudit(t, m, v, []Monomial{x}, &a)
	m.integer.Budget().FailAfter(-1)
	serviceMust(t, v.CloseChecked())
	if m.monomials.creation != before {
		t.Fatal("stale buffer receipt swept foreign replacement", before, m.monomials.creation)
	}
	serviceMust(t, m.monomials.CleanupCreation())
}
