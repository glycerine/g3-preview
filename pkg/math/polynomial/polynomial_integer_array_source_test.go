// Source: src/math/polynomial/polynomial.cpp:3384-3428; src/util/mpz.cpp array gcd.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestPolynomialIntegerArrayActualObjectAndEarlyOne(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 7}, [][]MonomialPower{{{0, 1}}, nil})
	l, e := m.polynomialLocation(p)
	serviceMust(t, e)
	view := util.IntegerArrayView{Allocator: m.monomials.allocator, Object: l.raw, Count: 64, Offset: 12, Stride: 12}
	g := util.NewInt(99)
	serviceMust(t, m.modular.GCDManyIndexed(view, &g))
	if g != util.NewInt(1) {
		t.Fatal("source early-one must leave the overlong tail unread", g)
	}
	// The deliberately overlong borrowed view is a bounded physical-guard
	// fixture; source early-one skipped it above. A read at the actual object boundary fails.
	view.Count = 1
	view.Offset = uint64(l.requested)
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
			g := util.NewInt(19)
			view := util.IntegerArrayView{Allocator: foreign.Allocator, Count: 0}
			serviceMust(t, z.GCDManyIndexed(view, &g))
			if g != util.NewInt(0) {
				t.Fatal("source count-zero RESET", name, g)
			}
			integerPhysicalAudit(t, z, &g)
		})
	}
}

func TestPolynomialIntegerArrayReachedRuntimeCohortAndUnreadZero(t *testing.T) {
	z, _ := integerManager(t)
	foreign, _ := integerManager(t)
	raw, e := foreign.RuntimeOwner().Allocate(12)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, foreign.RuntimeOwner().Deallocate(&raw)) })
	for offset, value := range []uint32{6, 0, 0} {
		serviceMust(t, foreign.RuntimeOwner().WriteUint32(raw, uint64(offset)*4, value))
	}
	view := util.IntegerArrayView{Runtime: foreign.RuntimeOwner(), Raw: raw, Stride: 12}
	g := util.NewInt(19)
	serviceMust(t, z.GCDManyIndexed(view, &g))
	if g != util.NewInt(0) {
		t.Fatal("empty view must not touch borrowed Runtime", g)
	}
	view.Count = 1
	g = util.NewInt(19)
	e = z.GCDManyIndexed(view, &g)
	if !errors.Is(e, umemory.ErrRuntimeOwner) || g != util.NewInt(19) {
		t.Fatal("reached foreign Runtime must reject before publication", e, g)
	}
	integerPhysicalAudit(t, z, &g)
	integerPhysicalAuditWithRaw(t, foreign, []umemory.Block{raw})
	own, e := z.RuntimeOwner().Allocate(24)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, z.RuntimeOwner().Deallocate(&own)) })
	for offset, value := range []uint32{1, 0, 0, 7, 0, 0} {
		serviceMust(t, z.RuntimeOwner().WriteUint32(own, uint64(offset)*4, value))
	}
	serviceMust(t, z.GCDManyIndexed(util.IntegerArrayView{Runtime: z.RuntimeOwner(), Raw: own, Count: 64, Stride: 12}, &g))
	if g != util.NewInt(1) {
		t.Fatal("same Runtime source early-one must skip physical tail", g)
	}
	integerPhysicalAuditWithRaw(t, z, []umemory.Block{own}, &g)
}
