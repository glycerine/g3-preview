// Source: src/util/ref_buffer.h; src/math/polynomial/polynomial.cpp:5654-5810.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

// Ownership follows actual completed local fields and the unjoined INC.
// sourceRaw is a nonowning stale-source token after a consumed old free.
func polynomialPSCPhysical(t *testing.T, m *MultivariatePolynomialCore, refs []*PolynomialRef, polys []Polynomial, external []memory.Block) {
	polynomialPSCPhysicalVectors(t, m, refs, nil, polys, external)
}
func polynomialPSCPhysicalVectors(t *testing.T, m *MultivariatePolynomialCore, refs []*PolynomialRef, vectors []*PolynomialRefVector, polys []Polynomial, external []memory.Block) {
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
			for i := b.cursor; i < b.count; i++ {
				var p Polynomial
				var e error
				if b.pendingOwns {
					p, e = pscReadHandle(m, b.pending, i)
				} else {
					p, e = b.physical(m, i)
				}
				serviceMust(t, e)
				polys = append(polys, p)
			}
		}
		polys = append(polys, b.incoming)
		external = append(external, b.raw, b.pending)
	}
	if s := &m.lazardLocal; s.active {
		if s.constructed >= 1 {
			polys = append(polys, s.x.value, s.x.incoming)
		}
		if s.constructed >= 2 {
			polys = append(polys, s.c.value, s.c.incoming)
		}
	}
	var public []*Int
	polynomialNewtonPhysicalWithRaw(t, m, nil, refs, vectors, nil, polys, nil, external, public...)
}
