// Source: src/math/polynomial/polynomial.cpp:3384-3428; src/util/mpz.cpp array gcd.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialIntegerArrayActualObjectAndEarlyOne(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 7}, [][]MonomialPower{{{0, 1}}, nil})
	l, e := m.polynomialLocation(p)
	serviceMust(t, e)
	view := integerArrayView{allocator: m.monomials.allocator, object: l.raw, count: 64, offset: 12, stride: 12}
	g := NewInt(99)
	serviceMust(t, m.modular.GCDManyIndexed(view, &g))
	if g != NewInt(1) {
		t.Fatal("source early-one must leave the overlong tail unread", g)
	}
	// The deliberately overlong borrowed view is a bounded physical-guard
	// fixture; source early-one skipped it above. A read at the actual object boundary fails.
	view.count = 1
	view.offset = uint64(l.requested)
	e = m.modular.GCDManyIndexed(view, &g)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("reached object read guard", e)
	}
	polynomialPSCPhysical(t, m, nil, []Polynomial{p}, nil)
	serviceMust(t, m.modular.Del(&g))
}

func TestPolynomialIntegerArrayZeroLeavesForeignAllocatorUnread(t *testing.T) {
	for _, name := range []string{"foreign", "closed"} {
		t.Run(name, func(t *testing.T) {
			z, _ := integerManager(t)
			foreign, _ := integerManager(t)
			if name == "closed" {
				serviceMust(t, foreign.CloseChecked())
			}
			g := NewInt(19)
			view := integerArrayView{allocator: foreign.allocator, count: 0}
			serviceMust(t, z.GCDManyIndexed(view, &g))
			if g != NewInt(0) {
				t.Fatal("source count-zero RESET", name, g)
			}
			integerPhysicalAudit(t, z, &g)
		})
	}
}

func TestPolynomialIntegerArrayReachedRuntimeCohortAndUnreadZero(t *testing.T) {
	z, _ := integerManager(t)
	foreign, _ := integerManager(t)
	raw, e := foreign.memory.Allocate(12)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, foreign.memory.Deallocate(&raw)) })
	for offset, value := range []uint32{6, 0, 0} {
		serviceMust(t, foreign.memory.WriteUint32(raw, uint64(offset)*4, value))
	}
	view := integerArrayView{runtime: foreign.memory, raw: raw, stride: 12}
	g := NewInt(19)
	serviceMust(t, z.GCDManyIndexed(view, &g))
	if g != NewInt(0) {
		t.Fatal("empty view must not touch borrowed Runtime", g)
	}
	view.count = 1
	g = NewInt(19)
	e = z.GCDManyIndexed(view, &g)
	if !errors.Is(e, memory.ErrRuntimeOwner) || g != NewInt(19) {
		t.Fatal("reached foreign Runtime must reject before publication", e, g)
	}
	integerPhysicalAudit(t, z, &g)
	integerPhysicalAuditWithRaw(t, foreign, []memory.Block{raw})
	own, e := z.memory.Allocate(24)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, z.memory.Deallocate(&own)) })
	for offset, value := range []uint32{1, 0, 0, 7, 0, 0} {
		serviceMust(t, z.memory.WriteUint32(own, uint64(offset)*4, value))
	}
	serviceMust(t, z.GCDManyIndexed(integerArrayView{runtime: z.memory, raw: own, count: 64, stride: 12}, &g))
	if g != NewInt(1) {
		t.Fatal("same Runtime source early-one must skip physical tail", g)
	}
	integerPhysicalAuditWithRaw(t, z, []memory.Block{own}, &g)
}
