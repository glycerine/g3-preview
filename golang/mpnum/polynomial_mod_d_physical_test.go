// Source: src/math/polynomial/polynomial.h:62-69; src/math/polynomial/polynomial.cpp:2117-2139,4905-5091.
// Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

// This independent named census additionally accounts the actual map cold owner
// and its published/pending Runtime vector. No frozen shared helper is edited.
func polynomialModDPhysical(t *testing.T, m *MultivariatePolynomialCore, q, r *PolynomialRef, d *PolynomialVariableDegrees, polys []Polynomial, public ...*Int) {
	t.Helper()
	s := &m.pseudoDivision
	for i := range s.refs {
		if s.refs[i].value != (Polynomial{}) {
			polys = append(polys, s.refs[i].value)
		}
		if s.refs[i].incoming != (Polynomial{}) {
			polys = append(polys, s.refs[i].incoming)
		}
	}
	if m.powerLocal.active {
		polys = append(polys, m.powerLocal.result.value, m.powerLocal.result.incoming)
	}
	refs := []*PolynomialRef{r}
	if q != nil {
		refs = append(refs, q)
	}
	raw := []memory.Block{}
	if m.queryPositions.raw != (memory.Block{}) {
		raw = append(raw, m.queryPositions.raw)
	}
	if m.queryPositions.pending != (memory.Block{}) {
		raw = append(raw, m.queryPositions.pending)
	}
	polynomialModDNewtonPhysicalWithRaw(t, m, nil, refs, nil, nil, polys, nil, raw, []*PolynomialVariableDegrees{d}, append(public, &s.numeral)...)
	monos := []Monomial{s.monomial.value, s.monomial.incoming}
	// Ref outputs are real named polynomial roots, not inferred directory entries.
	for _, ref := range refs {
		polys = append(polys, ref.root.value, ref.root.incoming)
	}
	polynomialOperationMonomialAudit(t, m, polys, monos)
}
func polynomialModDNewtonPhysicalWithRaw(t *testing.T, m *MultivariatePolynomialCore, owners []*PolynomialNewtonInterpolator, references []*PolynomialRef, refvectors []*PolynomialRefVector, lazy []*PolynomialNewtonVector, polys []Polynomial, handlers []*PolynomialDeletionHandler, external []memory.Block, degrees []*PolynomialVariableDegrees, public ...*Int) {
	t.Helper()
	var headers uint64
	for _, d := range degrees {
		headers += d.header
		if !d.rawDestroyed && d.raw != (memory.Block{}) {
			external = append(external, d.raw)
		}
		if d.pending != (memory.Block{}) {
			external = append(external, d.pending)
		}
	}
	polynomialNewtonPhysicalWithExtra(t, m, owners, references, refvectors, lazy, polys, handlers, external, headers, public...)
}
