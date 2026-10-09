// Source: src/util/ref_buffer.h; src/math/polynomial/polynomial.cpp:5654-5810.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// Ownership follows actual completed local fields and native reference slices.
func polynomialPSCPhysical(t *testing.T, m *MultivariatePolynomialCore, refs []*PolynomialRef, polys []Polynomial, external []util.Block) {
	polynomialPSCPhysicalVectors(t, m, refs, nil, polys, external)
}
func polynomialPSCPhysicalVectors(t *testing.T, m *MultivariatePolynomialCore, refs []*PolynomialRef, vectors []*PolynomialRefVector, polys []Polynomial, external []util.Block) {
	t.Helper()
	b := &m.pscBuffer
	if s := &m.pscLocal; s.active {
		for i := uint8(0); i < s.constructed; i++ {
			polys = append(polys, s.refs[i].root.value, s.refs[i].root.incoming)
		}
	}
	if s := &m.pscNext; s.active {
		for i := uint8(0); i < s.constructed; i++ {
			polys = append(polys, s.refs[i].value, s.refs[i].incoming)
		}
	}
	if b.constructed {
		if !b.finalized {
			for i := b.cursor; uint64(i) < uint64(len(b.values)); i++ {
				p, e := b.physical(m, i)
				serviceMust(t, e)
				polys = append(polys, p)
			}
		}
	}
	if s := &m.lazardLocal; s.active {
		if s.constructed >= 1 {
			polys = append(polys, s.x.value, s.x.incoming)
		}
		if s.constructed >= 2 {
			polys = append(polys, s.c.value, s.c.incoming)
		}
	}
	var public []*util.Int
	polynomialNewtonPhysicalWithRaw(t, m, nil, refs, vectors, nil, polys, nil, external, public...)
}
