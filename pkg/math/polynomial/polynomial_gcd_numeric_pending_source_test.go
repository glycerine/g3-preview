// Source: src/math/polynomial/polynomial.cpp:5356-5480;
// src/util/mpz.cpp:1186-1205.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialResultantPhysicalDescriptorRefusalComposedCleanup(t *testing.T) {
	t.Run("owned", func(t *testing.T) { polynomialResultantDescriptorRefusal(t, false) })
	t.Run("foreign-replacement", func(t *testing.T) { polynomialResultantDescriptorRefusal(t, true) })
}
func polynomialResultantDescriptorRefusal(t *testing.T, replacement bool) {
	budget := arena.NewBudget(1 << 25)
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{PageSize: 8, SlotLimit: 12, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, e)
	t.Cleanup(func() {
		q.Close()
		if budget.Used() != 0 {
			t.Errorf("actual retained owning bytes %d", budget.Used())
		}
	})
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, full.CloseChecked()) })
	c := full.core
	z := q.Integers()
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
	p, e := c.MkTerms([]unumeral.Int{z.Two64, unumeral.NewInt(1), unumeral.NewInt(1), unumeral.NewInt(2)}, ms)
	serviceMust(t, e)
	polynomialCaller(t, c, p)
	b := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, full.CleanupGCD()); serviceMust(t, r.Close()) })
	cause := full.Resultant(p, b, 1, r)

	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, b, r.root.value}, nil, r.header)
	if !errors.Is(cause, arena.ErrSlotLimit) || z.PendingCell.Object == (umemory.ObjectRef{}) {
		t.Fatal("actual bounded descriptor-after-physical prefix not reached", cause, z.PendingCell)
	}
	root := z.PendingCell
	var foreign unumeral.Int
	var fillers [16]unumeral.Int
	var filled int
	t.Cleanup(func() {
		for i := 0; i < filled; i++ {
			serviceMust(t, z.Del(&fillers[i]))
		}
	})
	if replacement {
		serviceMust(t, z.CleanupPendingAllocation())
		for filled < len(fillers) {
			foreign = unumeral.NewInt(19)
			e = z.Copy(&foreign, &z.Two64)
			if e != nil {
				break
			}
			fillers[filled] = foreign
			filled++
		}
		if !errors.Is(e, arena.ErrSlotLimit) || z.PendingCell.Object == (umemory.ObjectRef{}) {
			t.Fatal("actual foreign replacement refusal", e, z.PendingCell)
		}
		if z.PendingCell.Object == root.Object {
			t.Fatal("replacement must carry a distinct physical generation")
		}
		root = z.PendingCell
		named := []*unumeral.Int{&foreign}
		for i := 0; i < filled; i++ {
			named = append(named, &fillers[i])
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, b, r.root.value}, named, r.header)
	}
	serviceMust(t, full.CleanupGCD())
	if replacement {
		if z.PendingCell != root {
			t.Fatal("matched ExactDiv receipt swept foreign replacement", root, z.PendingCell)
		}
		serviceMust(t, z.CleanupPendingAllocation())
		serviceMust(t, z.Del(&foreign))
	} else if z.PendingCell.Object != (umemory.ObjectRef{}) {
		t.Fatal("composed cleanup failed to drain exact invocation numeric object", root, z.PendingCell)
	}
	for i := 0; i < filled; i++ {
		serviceMust(t, z.Del(&fillers[i]))
	}
	filled = 0
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, b, r.root.value}, nil, r.header)
}
