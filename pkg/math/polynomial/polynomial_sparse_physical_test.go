// Source: src/math/polynomial/polynomial.cpp:3072-3182.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// Independently named source owners. The full-manager children remain fresh
// in this sparse-only fixture; no allocator registry supplies roots.
func polynomialSparsePhysical(t *testing.T, s *PolynomialSparseInterpolator, out *PolynomialRef, polys []Polynomial, extraNum ...*util.Int) {
	t.Helper()
	full := s.owner
	m := full.core
	sk := s.skeleton
	raw := []util.Block{}
	public := []*util.Int{&s.interpolation.aux}
	public = append(public, extraNum...)
	extra := full.header + full.upm.header + full.upm.modular.Header + full.upm.store.header + s.header
	if len(full.upm.pages) != 0 || len(full.upm.store.backings) != 0 {
		t.Fatal("unexpected UP child work")
	}
	for _, values := range [][]util.Int{s.inputs, s.outputs, s.interpolation.vectors[0], s.interpolation.vectors[1], s.interpolation.vectors[2]} {
		for i := range values {
			public = append(public, &values[i])
		}
	}
	solver := &s.interpolation.solver
	if s.interpolation.solverConstructed {
		extra += solver.header
		for i := range solver.b {
			public = append(public, &solver.b[i])
		}
		for i := range solver.A {
			for j := range solver.A[i] {
				public = append(public, &solver.A[i][j])
			}
		}
	}

	if sk != nil {
		extra += sk.header
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
