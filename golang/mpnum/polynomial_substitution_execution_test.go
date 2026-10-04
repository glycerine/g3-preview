// Source: src/math/polynomial/polynomial.cpp:6401-6425; src/util/obj_ref.h.
// Licensed under the MIT license. Author: Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialSubstitutionStoppedCleanupDoesNotOwnLaterPower(t *testing.T) {
	m := polynomialCellFixture(t)
	r := polynomialQueryTerms(t, m, []int32{3}, [][]MonomialPower{{{0, 2}, {1, 1}}})
	p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{1, 1}}, nil})
	q := polynomialQueryTerms(t, m, []int32{2}, [][]MonomialPower{nil})
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	foreignOut, e := NewPolynomialRef(m)
	serviceMust(t, e)
	var big Int
	limbs := make([]uint32, 33)
	limbs[0] = 17
	limbs[32] = 1
	serviceMust(t, m.integer.SetLimbs(&big, limbs))
	term, e := m.monomials.MkVariable(2, 17)
	serviceMust(t, e)
	foreign, e := m.MkTerms([]Int{big, NewInt(1)}, []Monomial{term, m.monomials.unit})
	serviceMust(t, e)
	polynomialCaller(t, m, foreign)

	h, e := NewPolynomialDeletionHandler(m.integer.budget, func(h *PolynomialDeletionHandler, m *MultivariatePolynomialCore, p Polynomial) error {
		s := &m.substitution.scope
		if !s.productConstructed || p != s.product.value {
			return nil
		}
		h.State++
		m.integer.budget.FailAfter(0)
		b, e := m.memory.Allocate(1)
		if b != (memory.Block{}) {
			t.Fatal("unexpected raw publication", b)
		}
		return e
	})
	serviceMust(t, e)
	serviceMust(t, m.AddDeletionHandler(h))
	e = m.SubstitutePolynomialFraction(r, 0, p, q, out)
	if !numericPhysicalReservationRefusal(e) || !m.substitution.scope.active || h.State != 1 {
		t.Fatal("actual source local stop", e, h.State)
	}
	polynomialSubstitutionPhysicalWithHandlers(t, m, []*PolynomialRef{out, foreignOut}, []Polynomial{r, p, q, foreign}, nil, []*PolynomialDeletionHandler{h}, &big)
	m.integer.budget.FailAfter(-1)
	m.integer.budget.FailAfter(0)
	e = m.Power(foreign, 2, foreignOut)
	if !numericPhysicalReservationRefusal(e) || (m.monomials.creation.stage == 0 && m.pending.raw == (memory.ObjectRef{}) && m.pending.value == (Polynomial{})) {
		t.Fatal("actual later caller construction", e, m.monomials.creation, m.pending)
	}
	before := m.monomials.creation
	beforePending := m.pending
	polynomialSubstitutionPhysicalWithHandlers(t, m, []*PolynomialRef{out, foreignOut}, []Polynomial{r, p, q, foreign}, nil, []*PolynomialDeletionHandler{h}, &big)
	m.integer.budget.FailAfter(-1)
	serviceMust(t, m.CleanupSubstitution())
	if m.monomials.creation != before || m.pending != beforePending {
		t.Fatal("retained Sub scope acquired later Power construction", before, m.monomials.creation, beforePending, m.pending)
	}
	if h.State != 1 || m.substitution.scope.active {
		t.Fatal("completed original unwind/actual callback count", h.State, m.substitution.scope.active)
	}
	polynomialSubstitutionPhysicalWithHandlers(t, m, []*PolynomialRef{out, foreignOut}, []Polynomial{r, p, q, foreign}, nil, []*PolynomialDeletionHandler{h}, &big)
	serviceMust(t, m.CleanupPolynomialPower())
	serviceMust(t, m.CleanupConstruction())
	serviceMust(t, m.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
	serviceMust(t, out.Close())
	serviceMust(t, foreignOut.Close())
	serviceMust(t, m.integer.Del(&big))
}
