// Source: src/math/polynomial/polynomial.cpp:3180; src/util/obj_ref.h;
// src/math/polynomial/polynomial.cpp:2396-2440.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialSparseOutputCallbackNumericPendingRemainsForeign(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	q, e := util.NewRationalManager(util.RationalConfig{Integers: util.IntegerConfig{Cells: arena.Config{PageSize: 8, SlotLimit: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, e)
	t.Cleanup(func() {
		q.Close()
		if budget.Used() != 0 {
			t.Error("actual owning bytes", budget.Used())
		}
	})
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, full.CloseChecked()) })
	m := full.core
	_, e = m.monomials.MkVar()
	serviceMust(t, e)
	serviceMust(t, m.modular.SetZPUint64(97))
	p := polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{0, 1}}})
	sk, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, sk.CloseChecked()) })
	s, e := NewPolynomialSparseInterpolator(full, sk)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.CloseChecked()) })
	two := util.NewInt(2)
	polyTwo := polynomialQueryTerms(t, m, []int32{2}, [][]MonomialPower{nil})
	ok, e := s.Add(&two, polyTwo)
	serviceMust(t, e)
	if !ok {
		t.Fatal("source sample")
	}
	unit, e := m.monomials.Unit()
	serviceMust(t, e)
	old, e := m.MkTerms([]util.Int{util.NewInt(5)}, []Monomial{unit})
	serviceMust(t, e)
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	serviceMust(t, out.root.assign(m, old))
	foreign := util.NewInt(19)
	var reached util.SourceCellBacking
	calls := 0
	handler, e := NewPolynomialDeletionHandler(budget, func(_ *PolynomialDeletionHandler, c *MultivariatePolynomialCore, a Polynomial) error {
		if a == old {
			calls++
			x := c.integer.Copy(&foreign, &c.integer.Two64)
			if !errors.Is(x, arena.ErrSlotLimit) {
				t.Fatal("actual callback Copy refusal", x)
			}
			reached = c.integer.PendingCell
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, m.AddDeletionHandler(handler))
	t.Cleanup(func() { serviceMust(t, m.RemoveDeletionHandler(handler)); serviceMust(t, handler.CloseChecked()) })
	ok, e = s.Interpolate(out)
	serviceMust(t, e)
	if !ok || calls != 1 || reached.Object == (util.ObjectRef{}) {
		t.Fatal("actual completed callback publication", ok, calls, reached)
	}
	polynomialSparsePhysical(t, s, out, []Polynomial{p, polyTwo}, &foreign)
	serviceMust(t, s.CleanupInterpolation())
	if q.Integers().PendingCell != reached {
		t.Fatal("sparse swept actual callback-owned pending", reached, q.Integers().PendingCell)
	}
	polynomialSparsePhysical(t, s, out, []Polynomial{p, polyTwo}, &foreign)
	serviceMust(t, q.Integers().CleanupPendingAllocation())
	serviceMust(t, q.Integers().Del(&foreign))
}

func TestPolynomialSparseOutputCallbackCreationRemainsForeign(t *testing.T) {
	placementSeen, factorySeen := false, false
	for ordinal := int64(0); ordinal < 12; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialSparseDirectFactoryFixture(t)
			m := full.core
			z := m.integer
			serviceMust(t, m.modular.SetZPUint64(97))
			p := polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{0, 1}}})
			sk, e := NewPolynomialSkeleton(m, p, 0)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, sk.CloseChecked()) })
			s, e := NewPolynomialSparseInterpolator(full, sk)
			serviceMust(t, e)
			t.Cleanup(func() { z.Budget().FailAfter(-1); serviceMust(t, s.CloseChecked()) })
			in := util.NewInt(2)
			q := polynomialQueryTerms(t, m, []int32{2}, [][]MonomialPower{nil})
			ok, e := s.Add(&in, q)
			serviceMust(t, e)
			if !ok {
				t.Fatal("source sample")
			}
			unit, e := m.monomials.Unit()
			serviceMust(t, e)
			old, e := m.MkTerms([]util.Int{util.NewInt(5)}, []Monomial{unit})
			serviceMust(t, e)
			out, e := NewPolynomialRef(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, out.Close()) })
			serviceMust(t, out.root.assign(m, old))
			var callbackResult Polynomial
			var callbackError error
			handler, e := NewPolynomialDeletionHandler(z.Budget(), func(_ *PolynomialDeletionHandler, c *MultivariatePolynomialCore, a Polynomial) error {
				if a == old {
					z.Budget().FailAfter(ordinal)
					callbackResult, callbackError = c.MkVariable(0, 17)
				}
				return nil
			})
			serviceMust(t, e)
			serviceMust(t, m.AddDeletionHandler(handler))
			t.Cleanup(func() { serviceMust(t, m.RemoveDeletionHandler(handler)); serviceMust(t, handler.CloseChecked()) })
			ok, e = s.Interpolate(out)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual source unwind", e)
			}
			if callbackError != nil && !numericPhysicalReservationRefusal(callbackError) {
				t.Fatal("actual callback allocation category", callbackError)
			}
			before := m.pending
			generation := m.monomials.creation.generation
			active := m.generatedCalleePending
			placementSeen = placementSeen || before.generation != 0
			factorySeen = factorySeen || active
			polynomialSparsePhysical(t, s, out, []Polynomial{p, q, callbackResult})
			z.Budget().FailAfter(-1)
			serviceMust(t, s.CleanupInterpolation())
			if m.pending != before || m.monomials.creation.generation != generation || m.generatedCalleePending != active {
				t.Fatal("sparse swept callback-owned creation", before, m.pending, generation, m.monomials.creation.generation, active, m.generatedCalleePending)
			}
			polynomialSparsePhysical(t, s, out, []Polynomial{p, q, callbackResult})
			serviceMust(t, m.cleanupCreation())
			if callbackResult != (Polynomial{}) {
				serviceMust(t, m.CleanupValue(&callbackResult))
			}
		})
	}
	if !placementSeen || !factorySeen {
		t.Fatal("actual callback entered-placement and factory prefixes not covered", placementSeen, factorySeen)
	}
}

// Actual selected Direct allocation profile exposes each constructor request
// even in the release assertion profile; no allocator field is forged.
func polynomialSparseDirectFactoryFixture(t *testing.T) *MultivariatePolynomialManager {
	q, b := rationalManager(t, false)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.Integers().Memory, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	t.Cleanup(func() { serviceMust(t, mm.DecManagerRef()) })
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, full.CloseChecked()) })
	for i := 0; i < 11; i++ {
		_, e = mm.MkVar()
		serviceMust(t, e)
	}
	return full
}
