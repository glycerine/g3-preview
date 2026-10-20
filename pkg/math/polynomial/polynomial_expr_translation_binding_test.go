package polynomial

import "testing"

func TestPolynomialExprTranslationBorrowedBindings(t *testing.T) {
	m := polynomialCellFixture(t)
	integers, err := m.IntegerManager()
	serviceMust(t, err)
	monomials, err := m.MonomialManager()
	serviceMust(t, err)
	if integers != m.integer || monomials != m.monomials {
		t.Fatal("translation bindings created shadow owners")
	}
	before, err := m.NumVars()
	serviceMust(t, err)
	variable, err := m.MkVar()
	serviceMust(t, err)
	after, err := m.NumVars()
	serviceMust(t, err)
	if variable != before || after != before+1 {
		t.Fatal("source variable forwarding", before, variable, after)
	}
	zero, err := m.Zero()
	serviceMust(t, err)
	yes, err := m.IsZero(zero)
	serviceMust(t, err)
	if !yes {
		t.Fatal("canonical zero binding")
	}
}
