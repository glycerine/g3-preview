// Source: src/math/polynomial/polynomial.cpp:6237-6425; src/util/obj_ref.h.
// Licensed under the MIT license.
// Author: Leonardo de Moura.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func polynomialSubstitutionPhysical(t *testing.T, m *MultivariatePolynomialCore, refs []*PolynomialRef, polys []Polynomial, external []util.Block, values ...*util.Int) {
	polynomialSubstitutionPhysicalWithHandlers(t, m, refs, polys, external, nil, values...)
}
func polynomialSubstitutionPhysicalWithHandlers(t *testing.T, m *MultivariatePolynomialCore, refs []*PolynomialRef, polys []Polynomial, external []util.Block, handlers []*PolynomialDeletionHandler, values ...*util.Int) {
	t.Helper()
	d := &m.substitution
	s := &d.scope
	degrees := []*PolynomialVariableDegrees{}
	for _, b := range []util.Block{m.queryPositions.raw, m.queryPositions.pending} {
		if b != (util.Block{}) {
			external = append(external, b)
		}
	}
	values = append(values, &s.newA, &s.tmp)
	for _, r := range []*polynomialReference{&s.p1, &s.q1, &s.product} {
		polys = append(polys, r.value, r.incoming)
	}
	if s.returned != (Polynomial{}) {
		polys = append(polys, s.returned)
	}
	for _, r := range refs {
		polys = append(polys, r.root.value, r.root.incoming)
	}
	if m.powerLocal.active {
		polys = append(polys, m.powerLocal.result.value, m.powerLocal.result.incoming)
	}
	polynomialModDNewtonPhysicalWithRaw(t, m, nil, refs, nil, nil, polys, handlers, external, degrees, values...)
	monos := append([]Monomial{s.monomial.value, s.monomial.incoming}, d.orphans...)
	polynomialOperationMonomialAudit(t, m, polys, monos)
}

func polynomialSubstitutionNamedRaw(m *MultivariatePolynomialCore) []util.Block {
	d := &m.substitution
	raw := []util.Block{m.tmp1.raw, m.tmp1.pending, d.found.raw, d.found.pending}
	for _, v := range []*PolynomialVariableDegrees{&d.max.maxDegree, &d.max.variables, &d.positions.positions} {
		if !v.rawDestroyed {
			raw = append(raw, v.raw)
		}
		raw = append(raw, v.pending)
	}
	return raw
}
func polynomialSubstitutionNamedPolynomials(m *MultivariatePolynomialCore) []Polynomial {
	s := &m.substitution.scope
	return []Polynomial{s.p1.value, s.p1.incoming, s.q1.value, s.q1.incoming, s.product.value, s.product.incoming, s.returned}
}
