// Source: src/math/polynomial/polynomial.cpp:1613-1683;
// src/util/vector.h; src/util/mpz.cpp.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func polynomialFactorsAudit(t *testing.T, f *PolynomialFactors, caller []Polynomial, input []*Int, refs ...*PolynomialRef) {
	polynomialFactorsAuditExtra(t, f, caller, input, 0, refs...)
}
func polynomialFactorsAuditExtra(t *testing.T, f *PolynomialFactors, caller []Polynomial, input []*Int, extraHeader uint64, refs ...*PolynomialRef) {
	polynomialFactorsAuditOwnersExtra(t, f, caller, input, extraHeader, nil, nil, refs...)
}
func polynomialFactorsAuditOwnersExtra(t *testing.T, f *PolynomialFactors, caller []Polynomial, input []*Int, extraHeader uint64, external []memory.Block, other []*PolynomialFactors, refs ...*PolynomialRef) {
	t.Helper()
	m := f.manager.core
	polys := append([]Polynomial(nil), caller...)
	add := func(p Polynomial) {
		if p == (Polynomial{}) {
			return
		}
		l, e := m.polynomialLocation(p)
		serviceMust(t, e)
		if !l.released {
			polys = append(polys, p)
		}
	}
	h := extraHeader
	raw := append([]memory.Block(nil), external...)
	owners := append([]*PolynomialFactors{f}, other...)
	for _, f := range owners {
		if f == nil || f.closed {
			continue
		}
		for i, phase := range f.phases {
			// Published-before-INC pointers are borrowed, including the source-defined
			// degree-push failure followed by caller lexical destruction. Their actual
			// owning caller/frame roots are enumerated independently; the stale pointer
			// itself grants neither polynomial nor physical-block ownership.
			if phase == 0 || phase == factorPublished || phase == factorDrained || phase == factorDropConsumed {
				continue
			}
			p, e := f.factor(uint32(i))
			serviceMust(t, e)
			add(p)
		}
		add(f.current.value)
		add(f.current.incoming)
		add(f.freshResult)
		h += f.header + uint64(cap(f.phases)) + uint64(cap(f.memberSetHistory))*uint64(reflect.TypeOf(polynomialFactorMemberSetHistory{}).Size()) + uint64(cap(f.constantHistory))*uint64(reflect.TypeOf(polynomialFactorConstantHistory{}).Size())
		input = append(input, &f.constant, &f.local)
		raw = append(raw, f.factors.raw, f.factors.pending, f.factors.old, f.degrees.raw, f.degrees.pending, f.degrees.old)
	}
	for _, r := range refs {
		if r != nil && !r.closed {
			h += r.header
			add(r.root.value)
			add(r.root.incoming)
		}
	}
	polynomialImpPhysicalAuditNamedExtra(t, f.manager, polys, input, h, raw)
}
