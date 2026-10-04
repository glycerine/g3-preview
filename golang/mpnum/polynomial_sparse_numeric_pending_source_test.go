// Source: src/math/polynomial/polynomial.cpp:3109-3131; src/util/mpz.cpp:1186-1205.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialSparseBigInputDescriptorRefusalRetainsActualRoot(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		t.Run(fmt.Sprint(replacement), func(t *testing.T) {
			budget := arena.NewBudget(1 << 24)
			q, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 8, SlotLimit: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
			serviceMust(t, e)
			t.Cleanup(func() {
				q.Close()
				if budget.Used() != 0 {
					t.Errorf("remaining actual owning bytes %d", budget.Used())
				}
			})
			full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, full.CloseChecked()) })
			m := full.core
			_, e = m.monomials.MkVar()
			serviceMust(t, e)
			serviceMust(t, m.modular.SetZPUint64(97))
			p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
			sk, e := NewPolynomialSkeleton(m, p, 0)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, sk.CloseChecked()) })
			s, e := NewPolynomialSparseInterpolator(full, sk)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, s.CloseChecked()) })
			_, e = s.Add(&q.z.two64, m.unit)
			if !errors.Is(e, arena.ErrSlotLimit) || q.z.pendingCell.object == (memory.ObjectRef{}) {
				t.Fatal("actual numeric physical allocation before descriptor refusal", e, q.z.pendingCell)
			}
			root := q.z.pendingCell
			polynomialSparsePhysical(t, s, nil, []Polynomial{p})
			if replacement {
				serviceMust(t, q.z.CleanupPendingAllocation())
				foreign := NewInt(19)
				e = q.z.Copy(&foreign, &q.z.two64)
				if !errors.Is(e, arena.ErrSlotLimit) || q.z.pendingCell.object == (memory.ObjectRef{}) {
					t.Fatal("actual later foreign pending", e)
				}
				newer := q.z.pendingCell
				if newer.object == root.object {
					t.Fatal("physical ObjectRef generation must distinguish replacement")
				}
				polynomialSparsePhysical(t, s, nil, []Polynomial{p})
				serviceMust(t, s.CloseChecked())
				if q.z.pendingCell != newer {
					t.Fatal("sparse receipt swept later foreign numeric allocation", newer, q.z.pendingCell)
				}
				serviceMust(t, q.z.CleanupPendingAllocation())
				serviceMust(t, q.z.Del(&foreign))
			} else {
				serviceMust(t, s.CloseChecked())
				if q.z.pendingCell.object != (memory.ObjectRef{}) {
					t.Fatal("owning sparse cleanup must drain only its actual numeric pending allocation", root, q.z.pendingCell)
				}
			}
			polynomialSparsePhysical(t, s, nil, []Polynomial{p})
		})
	}
}

func TestPolynomialSparseBigInputEveryActualCopyReservationPrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 64 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			m := full.core
			serviceMust(t, m.modular.SetZPUint64(97))
			p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
			sk, e := NewPolynomialSkeleton(m, p, 0)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, sk.CloseChecked()) })
			s, e := NewPolynomialSparseInterpolator(full, sk)
			serviceMust(t, e)
			t.Cleanup(func() { m.integer.budget.FailAfter(-1); serviceMust(t, s.CloseChecked()) })
			m.integer.budget.FailAfter(ordinal)
			ok, e := s.Add(&m.integer.two64, m.unit)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual reached numeric allocation", e)
			}
			polynomialSparsePhysical(t, s, nil, []Polynomial{p})
			m.integer.budget.FailAfter(-1)
			if e == nil {
				if !ok {
					t.Fatal("matching constant skeleton entry")
				}
				got, x := s.inputs.coefficient(m, 0)
				serviceMust(t, x)
				expected := NewInt(-36) // independent balanced residue: 2^64 mod 97 = 61 = -36
				equal, x := m.integer.Equal(&got, &expected)
				serviceMust(t, x)
				if !equal {
					t.Fatal("literal input Copy before output traversal")
				}
				success = true
				t.Log("BIG input copy reservation endpoint", ordinal)
			}
			serviceMust(t, s.CloseChecked())
			if m.integer.pendingCell.object != (memory.ObjectRef{}) {
				t.Fatal("owned pending numeric root remains", m.integer.pendingCell)
			}
			polynomialSparsePhysical(t, s, nil, []Polynomial{p})
		})
	}
	if !success {
		t.Fatal("no real BIG copy completion")
	}
}

func TestPolynomialSparseBigFieldInterpolationEveryNumericPrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 160 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			m := full.core
			z := m.integer
			var modulus Int
			serviceMust(t, z.SetLimbs(&modulus, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
			serviceMust(t, m.modular.SetZP(&modulus))
			serviceMust(t, z.Del(&modulus))
			p := polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{0, 1}}})
			unit, e := m.monomials.Unit()
			serviceMust(t, e)
			q, e := m.MkTerms([]Int{z.two64}, []Monomial{unit})
			serviceMust(t, e)
			q = polynomialCaller(t, m, q)
			sk, e := NewPolynomialSkeleton(m, p, 0)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, sk.CloseChecked()) })
			s, e := NewPolynomialSparseInterpolator(full, sk)
			serviceMust(t, e)
			t.Cleanup(func() { z.budget.FailAfter(-1); serviceMust(t, s.CloseChecked()) })
			ok, e := s.Add(&z.two64, q)
			serviceMust(t, e)
			if !ok {
				t.Fatal("same skeleton support")
			}
			out, e := NewPolynomialRef(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, out.Close()) })
			z.budget.FailAfter(ordinal)
			ok, e = s.Interpolate(out)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual numeric refusal", ordinal, e)
			}
			polynomialSparsePhysical(t, s, out, []Polynomial{p, q})
			z.budget.FailAfter(-1)
			serviceMust(t, s.CleanupInterpolation())
			polynomialSparsePhysical(t, s, out, []Polynomial{p, q})
			if z.pendingCell.object != (memory.ObjectRef{}) {
				t.Fatal("actual interpolation numeric pending", z.pendingCell)
			}
			if e == nil {
				if !ok {
					t.Fatal("single BIG field sample nonsingular")
				}
				equal, x := m.Equal(out.root.value, p)
				serviceMust(t, x)
				if !equal {
					t.Fatal("independent x coefficient 1")
				}
				success = true
				t.Log("BIG field numeric prefix endpoint", ordinal)
			}
		})
	}
	if !success {
		t.Fatal("no BIG field complete prefix")
	}
}
