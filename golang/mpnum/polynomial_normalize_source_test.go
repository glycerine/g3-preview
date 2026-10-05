// Source: src/math/polynomial/polynomial.cpp:5904-5940;
// src/math/polynomial/polynomial.h:873.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialNormalizeIndependentSignedIntegerContent(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{-6, -10}, [][]MonomialPower{{{0, 1}}, nil})
	result, e := full.Normalize(p)
	serviceMust(t, e)
	result = polynomialCaller(t, c, result)
	polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{1, 0, 0}: -3, {}: -5})
	// This source operation removes content, preserving sign; it is distinct
	// from flip_sign_if_lm_neg and monic normalization.
	same, e := full.Normalize(result)
	serviceMust(t, e)
	if same != result {
		t.Fatal("source gcd-one exact pointer return", same, result)
	}
	zero, e := full.Normalize(c.zero)
	serviceMust(t, e)
	if zero != c.zero {
		t.Fatal("source zero exact pointer return")
	}
}

func TestPolynomialNormalizeIndependentModularResidues(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{9, -1}, [][]MonomialPower{{{0, 1}}, nil})
	serviceMust(t, c.modular.SetZPUint64(7))
	result, e := full.Normalize(p)
	serviceMust(t, e)
	result = polynomialCaller(t, c, result)
	polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{1, 0, 0}: 2, {}: -1})
	polynomialImpPhysicalAuditValues(t, full, []Polynomial{p, result})
}

func TestPolynomialNormalizeEveryActualReservationPrefix(t *testing.T) {
	for route := uint32(0); route < 3; route++ {
		success := false
		for ordinal := int64(0); ordinal < 128 && !success; ordinal++ {
			t.Run(fmt.Sprintf("%d/%d", route, ordinal), func(t *testing.T) {
				full := polynomialPSCFullFixture(t)
				c := full.core
				p := polynomialQueryTerms(t, c, []int32{-6, -10}, [][]MonomialPower{{{0, 1}}, nil})
				want := polynomialQueryTerms(t, c, []int32{-3, -5}, [][]MonomialPower{{{0, 1}}, nil})
				var large, unitLarge Int
				inputs := []*Int{}
				if route == 1 {
					p = polynomialQueryTerms(t, c, []int32{9, -1}, [][]MonomialPower{{{0, 1}}, nil})
					want = polynomialQueryTerms(t, c, []int32{2, -1}, [][]MonomialPower{{{0, 1}}, nil})
					serviceMust(t, c.modular.SetZPUint64(7))
				}
				if route == 2 {
					serviceMust(t, c.integer.SetLimbs(&large, []uint32{0, 0, 7}))
					serviceMust(t, c.integer.SetLimbs(&unitLarge, []uint32{0, 0, 1}))
					t.Cleanup(func() { serviceMust(t, c.integer.Del(&large)); serviceMust(t, c.integer.Del(&unitLarge)) })
					x, e := c.monomials.MkVariable(0, 1)
					serviceMust(t, e)
					p, e = c.MkTerms([]Int{large, NewInt(7)}, []Monomial{x, c.monomials.unit})
					serviceMust(t, e)
					p = polynomialCaller(t, c, p)
					want, e = c.MkTerms([]Int{unitLarge, NewInt(1)}, []Monomial{x, c.monomials.unit})
					serviceMust(t, e)
					want = polynomialCaller(t, c, want)
					inputs = []*Int{&large, &unitLarge}
				}
				c.integer.budget.FailAfter(ordinal)
				result, e := full.Normalize(p)
				if e != nil && !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual Normalize refusal", route, ordinal, e)
				}
				polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, want, result}, inputs, 0)
				c.integer.budget.FailAfter(-1)
				serviceMust(t, full.CleanupNormalization())
				polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, want, result}, inputs, 0)
				if result != (Polynomial{}) {
					result = polynomialCaller(t, c, result)
				}
				if e == nil {
					equal, e := c.Equal(result, want)
					serviceMust(t, e)
					if !equal {
						t.Fatal("independent exact signed/field/BIG normalization", route, ordinal)
					}
					success = true
					t.Log("actual refusal endpoint", route, ordinal)
				}
			})
		}
		if !success {
			t.Fatal("no completed source route", route)
		}
	}
}

func TestPolynomialNormalizeBigDescriptorRefusalAndReplacement(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		t.Run(fmt.Sprint(replacement), func(t *testing.T) {
			budget := arena.NewBudget(1 << 24)
			q, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 8, SlotLimit: 9, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
			serviceMust(t, e)
			t.Cleanup(func() {
				q.Close()
				if budget.Used() != 0 {
					t.Errorf("remaining actual owning bytes %d", budget.Used())
				}
			})
			full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, full.CleanupNormalization()); serviceMust(t, full.CloseChecked()) })
			c := full.core
			p, e := c.MkTerms([]Int{q.z.two64}, []Monomial{c.monomials.unit})
			serviceMust(t, e)
			p = polynomialCaller(t, c, p)
			_, e = full.Normalize(p)
			if !errors.Is(e, arena.ErrSlotLimit) || q.z.pendingCell.object == (memory.ObjectRef{}) {
				t.Fatal("actual GCD Copy physical allocation before descriptor refusal", e, q.z.pendingCell)
			}
			old := q.z.pendingCell
			polynomialImpPhysicalAuditValues(t, full, []Polynomial{p})
			var foreign Int
			if replacement {
				serviceMust(t, q.z.CleanupPendingAllocation())
				e = q.z.Copy(&foreign, &q.z.two64)
				if !errors.Is(e, arena.ErrSlotLimit) || q.z.pendingCell.object == (memory.ObjectRef{}) {
					t.Fatal("actual foreign replacement", e, q.z.pendingCell)
				}
				if q.z.pendingCell.object == old.object {
					t.Fatal("replacement must have distinct physical generation")
				}
			}
			newer := q.z.pendingCell
			polynomialImpPhysicalAuditValues(t, full, []Polynomial{p})
			serviceMust(t, full.CleanupNormalization())
			if replacement {
				if q.z.pendingCell != newer {
					t.Fatal("Normalize cleanup swept foreign numeric replacement", newer, q.z.pendingCell)
				}
				serviceMust(t, q.z.CleanupPendingAllocation())
				serviceMust(t, q.z.Del(&foreign))
			} else if q.z.pendingCell.object != (memory.ObjectRef{}) {
				t.Fatal("Normalize did not drain actual invocation-owned pending allocation", q.z.pendingCell)
			}
			polynomialImpPhysicalAuditValues(t, full, []Polynomial{p})
		})
	}
}
