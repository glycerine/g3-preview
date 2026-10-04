// Source: src/math/polynomial/polynomial.cpp:1885-2214; src/util/vector.h. Licensed under the MIT license.
// Source author: Leonardo de Moura (leonardo), 2011-11-15.
package mpnum

import (
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func polynomialSOMPhysicalAudit(t *testing.T, m *MultivariatePolynomialCore, v *PolynomialSOMVector, callerMonos []Monomial, public ...*Int) {
	t.Helper()
	polynomialSOMPhysicalWithExternal(t, m, v, callerMonos, nil, public...)
}
func polynomialSOMPhysicalWithExternal(t *testing.T, m *MultivariatePolynomialCore, v *PolynomialSOMVector, callerMonos []Monomial, external []memory.Block, public ...*Int) {
	t.Helper()
	raw := append([]memory.Block{}, external...)
	var mono []Monomial
	var values []Int
	extra := uint64(0)
	var buffers []*PolynomialSOMBuffer
	addRaw := func(a memory.Block) {
		if a != (memory.Block{}) {
			raw = append(raw, a)
		}
	}
	if v != nil {
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
	for _, b := range buffers {
		a := &b.buffer
		extra += b.header + uint64(cap(b.retiredNumerals))*uint64(reflect.TypeOf(Int{}).Size()) + uint64(cap(b.retiredMonomials))*uint64(reflect.TypeOf(somRetainedMonomial{}).Size())
		extra += a.positions.header + uint64(cap(a.as.retired))*uint64(reflect.TypeOf(Int{}).Size()) + uint64(cap(b.sortNumerals.retired))*uint64(reflect.TypeOf(Int{}).Size())
		for _, s := range []*polynomialVector{&a.as, &a.ms, &b.sortNumerals} {
			addRaw(s.raw)
			addRaw(s.pending)
			addRaw(s.old)
		}
		addRaw(a.positions.raw)
		addRaw(a.positions.pending)
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
		if sv.old != (memory.Block{}) {
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
		if a.as.old != (memory.Block{}) {
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
	roots := append([]*Int{}, public...)
	for i := range values {
		roots = append(roots, &values[i])
	}
	polynomialNewtonPhysicalWithExtra(t, m, nil, nil, nil, nil, nil, nil, raw, extra, roots...)
	mono = append(mono, callerMonos...)
	polynomialOperationMonomialAudit(t, m, nil, mono)
}
