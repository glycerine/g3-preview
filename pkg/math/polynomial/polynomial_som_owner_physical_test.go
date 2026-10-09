// Source: src/math/polynomial/polynomial.cpp:1885-2214; src/util/vector.h.
package polynomial

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func polynomialSOMPhysicalAudit(t *testing.T, m *MultivariatePolynomialCore, v *PolynomialSOMVector, callerMonos []Monomial, public ...*util.Int) {
	t.Helper()
	polynomialSOMPhysicalWithExternal(t, m, v, callerMonos, nil, public...)
}
func polynomialSOMPhysicalWithExternal(t *testing.T, m *MultivariatePolynomialCore, v *PolynomialSOMVector, callerMonos []Monomial, external []util.Block, public ...*util.Int) {
	t.Helper()
	polynomialSOMPhysicalWithVectors(t, m, []*PolynomialSOMVector{v}, callerMonos, external, public...)
}
func polynomialSOMPhysicalWithVectors(t *testing.T, m *MultivariatePolynomialCore, vectors []*PolynomialSOMVector, callerMonos []Monomial, external []util.Block, public ...*util.Int) {
	polynomialSOMPhysicalWithPolynomials(t, m, vectors, callerMonos, external, nil, public...)
}
func polynomialSOMPhysicalWithPolynomials(t *testing.T, m *MultivariatePolynomialCore, vectors []*PolynomialSOMVector, callerMonos []Monomial, external []util.Block, polys []Polynomial, public ...*util.Int) {
	polynomialSOMPhysicalWithOwners(t, m, vectors, callerMonos, external, polys, nil, public...)
}
func polynomialSOMPhysicalWithOwners(t *testing.T, m *MultivariatePolynomialCore, vectors []*PolynomialSOMVector, callerMonos []Monomial, external []util.Block, polys []Polynomial, references []*PolynomialRef, public ...*util.Int) {
	t.Helper()
	raw, mono, values, extra := polynomialSOMNamedRoots(t, m, vectors)
	raw = append(raw, external...)
	roots := append([]*util.Int{}, public...)
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
func polynomialSOMNamedRoots(t *testing.T, m *MultivariatePolynomialCore, vectors []*PolynomialSOMVector) (raw []util.Block, mono []Monomial, values []util.Int, extra uint64) {
	t.Helper()
	vectors = append(vectors, &m.translateBuffers)
	seenVectors := map[*PolynomialSOMVector]bool{}
	var buffers []*PolynomialSOMBuffer
	addRaw := func(a util.Block) {
		if a != (util.Block{}) {
			raw = append(raw, a)
		}
	}
	for _, v := range vectors {
		if v != nil && !seenVectors[v] {
			seenVectors[v] = true
			extra += v.header + uint64(cap(v.children))*uint64(reflect.TypeOf((*PolynomialSOMBuffer)(nil)).Size())
			addRaw(v.data.raw)
			addRaw(v.data.pending)
			addRaw(v.data.old)
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
		extra += b.header + uint64(cap(b.retiredNumerals))*uint64(reflect.TypeOf(util.Int{}).Size()) + uint64(cap(b.retiredMonomials))*uint64(reflect.TypeOf(somRetainedMonomial{}).Size())
		extra += a.positions.header + uint64(cap(a.as.retired))*uint64(reflect.TypeOf(util.Int{}).Size()) + uint64(cap(b.sortNumerals.retired))*uint64(reflect.TypeOf(util.Int{}).Size())
		for _, s := range []*polynomialVector{&a.as, &a.ms, &b.sortNumerals} {
			addRaw(s.raw)
			addRaw(s.pending)
			addRaw(s.old)
		}
		sv := &b.sortNumerals
		start := b.sortTransferred
		if start < b.sortCleanupIndex {
			start = b.sortCleanupIndex
		}
		for i := start; i < sv.initialized; i++ {
			value, e := sv.coefficient(m, i)
			serviceMust(t, e)
			values = append(values, value)
		}
		if sv.old != (util.Block{}) {
			old := polynomialVector{raw: sv.old, width: 12}
			start := sv.oldCleanupIndex
			if start < sv.moved {
				start = sv.moved
			}
			for i := start; i < sv.oldInitialized; i++ {
				value, e := old.coefficient(m, i)
				serviceMust(t, e)
				values = append(values, value)
			}
		}
		values = append(values, sv.retired[:sv.retiredActive]...)
		values = append(values, a.gcd)
		for i := b.numeralTransferred; i < a.as.initialized; i++ {
			v, e := a.as.coefficient(m, i)
			serviceMust(t, e)
			values = append(values, v)
		}
		if a.as.old != (util.Block{}) {
			old := polynomialVector{raw: a.as.old, width: 12}
			start := a.as.oldCleanupIndex
			if start < a.as.moved {
				start = a.as.moved
			}
			for i := start; i < a.as.oldInitialized; i++ {
				v, e := old.coefficient(m, i)
				serviceMust(t, e)
				values = append(values, v)
			}
		}
		values = append(values, a.as.retired[:a.as.retiredActive]...)
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
