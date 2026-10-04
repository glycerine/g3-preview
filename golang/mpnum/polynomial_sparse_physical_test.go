// Source: src/math/polynomial/polynomial.cpp:3072-3182.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

// Independently named source owners. The full-manager children remain fresh
// in this sparse-only fixture; no allocator registry supplies roots.
func polynomialSparsePhysical(t *testing.T, s *PolynomialSparseInterpolator, out *PolynomialRef, polys []Polynomial, extraNum ...*Int) {
	t.Helper()
	full := s.owner
	m := full.core
	sk := s.skeleton
	raw := []memory.Block{m.queryPositions.raw, m.queryPositions.pending}
	public := []*Int{&s.interpolation.aux}
	public = append(public, extraNum...)
	extra := full.header + full.upm.header + full.upm.modular.header + full.upm.store.header + s.header
	if len(full.upm.pages) != 0 || len(full.upm.store.backings) != 0 {
		t.Fatal("unexpected UP child work")
	}
	raw = append(raw, full.degree2pos.raw, full.degree2pos.pending, full.varDegrees.raw, full.varDegrees.pending, full.lexPermutation.raw, full.lexPermutation.pending)
	for _, v := range []*polynomialVector{&s.inputs, &s.outputs, &s.interpolation.vectors[0], &s.interpolation.vectors[1], &s.interpolation.vectors[2]} {
		raw = append(raw, v.raw, v.pending, v.old)
		extra += uint64(cap(v.retired)) * uint64(reflect.TypeOf(Int{}).Size())
		for i := uint32(0); i < v.initialized; i++ {
			if v.raw == (memory.Block{}) {
				break
			}
			a, e := v.coefficient(m, i)
			serviceMust(t, e)
			public = append(public, &a)
		}
		if v.old != (memory.Block{}) {
			old := polynomialVector{raw: v.old, width: 12}
			for i := v.moved; i < v.oldInitialized; i++ {
				a, e := old.coefficient(m, i)
				serviceMust(t, e)
				public = append(public, &a)
			}
		}
		for i := uint32(0); i < v.retiredActive; i++ {
			public = append(public, &v.retired[i])
		}
	}
	raw = append(raw, s.interpolation.mons.raw, s.interpolation.mons.pending)
	solver := &s.interpolation.solver
	if s.interpolation.solverConstructed {
		extra += solver.header + uint64(cap(solver.rows))*uint64(reflect.TypeOf(linearNumeralVector{}).Size())
		vs := []*linearNumeralVector{&solver.A, &solver.b}
		for i := range solver.rows {
			vs = append(vs, &solver.rows[i])
		}
		for _, v := range vs {
			raw = append(raw, v.raw, v.pending, v.old)
			if v.width != 12 {
				continue
			}
			for i := uint32(0); i < v.initialized; i++ {
				a, e := v.intAt(m.memory, i)
				serviceMust(t, e)
				public = append(public, &a)
			}
		}
	}
	if sk != nil {
		extra += sk.header + uint64(cap(sk.retiredEntries)+cap(sk.retiredOriginals))*uint64(reflect.TypeOf(Monomial{}).Size())
		for _, v := range []*polynomialVector{&sk.entries, &sk.powers, &sk.originals} {
			raw = append(raw, v.raw, v.pending, v.old)
		}
		raw = append(raw, sk.ms.raw, sk.ms.pending)
	}
	handlers := []*PolynomialDeletionHandler{}
	for h := m.deletionHead; h != nil; h = h.next {
		handlers = append(handlers, h)
	}
	refs := []*PolynomialRef{}
	if out != nil {
		refs = append(refs, out)
		polys = append(polys, out.root.value, out.root.incoming)
	}
	polynomialNewtonPhysicalWithExtra(t, m, nil, refs, nil, nil, polys, handlers, raw, extra, public...)
	polynomialOperationMonomialAudit(t, m, polys, polynomialSkeletonRoots(t, sk))
}
