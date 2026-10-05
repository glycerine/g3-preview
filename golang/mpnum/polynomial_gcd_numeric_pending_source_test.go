// Source: src/math/polynomial/polynomial.cpp:5356-5480;
// src/util/mpz.cpp:1186-1205.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialResultantPhysicalDescriptorRefusalComposedCleanup(t *testing.T) {
	t.Run("owned", func(t *testing.T) { polynomialResultantDescriptorRefusal(t, false) })
	t.Run("foreign-replacement", func(t *testing.T) { polynomialResultantDescriptorRefusal(t, true) })
}
func polynomialResultantDescriptorRefusal(t *testing.T, replacement bool) {
	budget := arena.NewBudget(1 << 25)
	q, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 8, SlotLimit: 12, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, e)
	t.Cleanup(func() {
		q.Close()
		if budget.Used() != 0 {
			t.Errorf("actual retained owning bytes %d", budget.Used())
		}
	})
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, full.CloseChecked()) })
	c := full.core
	z := q.z
	for i := 0; i < 11; i++ {
		_, e = c.monomials.MkVar()
		serviceMust(t, e)
	}
	ms := make([]Monomial, 4)
	powers := [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil}
	for i := range ms {
		ms[i], e = c.monomials.MkMonomial(powers[i])
		serviceMust(t, e)
	}
	p, e := c.MkTerms([]Int{z.two64, NewInt(1), NewInt(1), NewInt(2)}, ms)
	serviceMust(t, e)
	polynomialCaller(t, c, p)
	b := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, full.CleanupGCD()); serviceMust(t, r.Close()) })
	cause := full.Resultant(p, b, 1, r)

	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, b, r.root.value}, nil, r.header)
	if !errors.Is(cause, arena.ErrSlotLimit) || z.pendingCell.object == (memory.ObjectRef{}) {
		t.Fatal("actual bounded descriptor-after-physical prefix not reached", cause, z.pendingCell)
	}
	root := z.pendingCell
	var foreign Int
	var fillers [16]Int
	var filled int
	t.Cleanup(func() {
		for i := 0; i < filled; i++ {
			serviceMust(t, z.Del(&fillers[i]))
		}
	})
	if replacement {
		serviceMust(t, z.CleanupPendingAllocation())
		for filled < len(fillers) {
			foreign = NewInt(19)
			e = z.Copy(&foreign, &z.two64)
			if e != nil {
				break
			}
			fillers[filled] = foreign
			filled++
		}
		if !errors.Is(e, arena.ErrSlotLimit) || z.pendingCell.object == (memory.ObjectRef{}) {
			t.Fatal("actual foreign replacement refusal", e, z.pendingCell)
		}
		if z.pendingCell.object == root.object {
			t.Fatal("replacement must carry a distinct physical generation")
		}
		root = z.pendingCell
		named := []*Int{&foreign}
		for i := 0; i < filled; i++ {
			named = append(named, &fillers[i])
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, b, r.root.value}, named, r.header)
	}
	serviceMust(t, full.CleanupGCD())
	if replacement {
		if z.pendingCell != root {
			t.Fatal("matched ExactDiv receipt swept foreign replacement", root, z.pendingCell)
		}
		serviceMust(t, z.CleanupPendingAllocation())
		serviceMust(t, z.Del(&foreign))
	} else if z.pendingCell.object != (memory.ObjectRef{}) {
		t.Fatal("composed cleanup failed to drain exact invocation numeric object", root, z.pendingCell)
	}
	for i := 0; i < filled; i++ {
		serviceMust(t, z.Del(&fillers[i]))
	}
	filled = 0
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, b, r.root.value}, nil, r.header)
}
