// Source: src/math/polynomial/polynomial.cpp:1885-2214; src/util/vector.h.
package polynomial

import (
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func polynomialSOMPhysicalAudit(t *testing.T, m *MultivariatePolynomialCore, v *PolynomialSOMVector, callerMonos []Monomial, public ...*unumeral.Int) {
	t.Helper()
	polynomialSOMPhysicalWithExternal(t, m, v, callerMonos, nil, public...)
}
func polynomialSOMPhysicalWithExternal(t *testing.T, m *MultivariatePolynomialCore, v *PolynomialSOMVector, callerMonos []Monomial, external []umemory.Block, public ...*unumeral.Int) {
	t.Helper()
	polynomialSOMPhysicalWithVectors(t, m, []*PolynomialSOMVector{v}, callerMonos, external, public...)
}
func polynomialSOMPhysicalWithVectors(t *testing.T, m *MultivariatePolynomialCore, vectors []*PolynomialSOMVector, callerMonos []Monomial, external []umemory.Block, public ...*unumeral.Int) {
	polynomialSOMPhysicalWithPolynomials(t, m, vectors, callerMonos, external, nil, public...)
}
func polynomialSOMPhysicalWithPolynomials(t *testing.T, m *MultivariatePolynomialCore, vectors []*PolynomialSOMVector, callerMonos []Monomial, external []umemory.Block, polys []Polynomial, public ...*unumeral.Int) {
	polynomialSOMPhysicalWithOwners(t, m, vectors, callerMonos, external, polys, nil, public...)
}
func polynomialSOMPhysicalWithOwners(t *testing.T, m *MultivariatePolynomialCore, vectors []*PolynomialSOMVector, callerMonos []Monomial, external []umemory.Block, polys []Polynomial, references []*PolynomialRef, public ...*unumeral.Int) {
	t.Helper()
	raw, mono, values, extra := polynomialSOMNamedRoots(t, m, vectors)
	raw = append(raw, external...)
	roots := append([]*unumeral.Int{}, public...)
	for i := range values {
		roots = append(roots, &values[i])
	}
	polynomialNewtonPhysicalWithExtra(t, m, nil, references, nil, nil, polys, nil, raw, extra, roots...)
	if m.translation.constructed {
		mono = append(mono, m.translation.xk.value, m.translation.xk.incoming)
	}
	mono = append(mono, callerMonos...)
	for _, r := range references {
		if r != nil {
			polys = append(polys, r.root.value, r.root.incoming)
		}
	}
	polynomialOperationMonomialAudit(t, m, polys, mono)
}

// Collect only the actual named vector owners and their retained children.
func polynomialSOMNamedRoots(t *testing.T, m *MultivariatePolynomialCore, vectors []*PolynomialSOMVector) (raw []umemory.Block, mono []Monomial, values []unumeral.Int, extra uint64) {
	t.Helper()
	vectors = append(vectors, &m.translateBuffers)
	seenVectors := map[*PolynomialSOMVector]bool{}
	var buffers []*PolynomialSOMBuffer
	for _, v := range vectors {
		if v != nil && !seenVectors[v] {
			seenVectors[v] = true
			extra += v.header
			for _, b := range v.children {
				if b != nil {
					buffers = append(buffers, b)
				}
			}
			if v.pending != nil {
				found := false
				for _, b := range buffers {
					if b == v.pending {
						found = true
					}
				}
				if !found {
					buffers = append(buffers, v.pending)
				}
			}
		}
	}
	for _, b := range buffers {
		a := &b.buffer
		extra += b.header + a.positions.header
		values = append(values, b.sortNumerals.values...)
		values = append(values, b.sortNumerals.retired...)
		values = append(values, a.gcd)
		values = append(values, a.as.values...)
		values = append(values, a.as.retired...)
		values = append(values, b.retiredNumerals[b.numeralCursor:b.retiredNumeralCount]...)
		n, e := a.ms.size(m)
		serviceMust(t, e)
		for i := b.monomialTransferred; i < n; i++ {
			if b.ownsMonomial(i) {
				v, e := a.ms.monomial(m, i)
				serviceMust(t, e)
				mono = append(mono, v)
			}
		}
		if a.hasPendingReference {
			mono = append(mono, a.pendingMonomial)
		}
		for i := b.monomialCursor; i < b.retiredMonomialCount; i++ {
			mono = append(mono, b.retiredMonomials[i].value)
		}
	}
	return
}
