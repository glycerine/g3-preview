// Source: src/math/polynomial/polynomial.cpp:6843-6862,7388-7390;
// src/math/polynomial/polynomial.h:986.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"errors"
	"fmt"
	"testing"
)

func TestPolynomialGlexMonicActualFieldAndIndependentCoefficients(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	if modularSourceDebug {
		_, e := full.GlexMonic(m.zero)
		if !errors.Is(e, ErrPolynomialAssertion) {
			t.Fatal("source field assertion before zero", e)
		}
	}
	serviceMust(t, m.modular.SetZPUint64(7))
	p := polynomialQueryTerms(t, m, []int32{2, -3}, [][]MonomialPower{{{0, 1}}, nil})
	result, e := full.GlexMonic(p)
	serviceMust(t, e)
	result = polynomialCaller(t, m, result)
	polynomialQueryExpect(t, m, result, map[[3]uint32]int64{{1, 0, 0}: 1, {}: 2})
	same, e := full.GlexMonic(result)
	serviceMust(t, e)
	if same != result {
		t.Fatal("source leading-one pointer short circuit", same, result)
	}
	zero, e := full.GlexMonic(m.zero)
	serviceMust(t, e)
	if zero != m.zero {
		t.Fatal("source zero exact pointer")
	}
}

func TestPolynomialGlexMonicEveryActualReservationPrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 96 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			m := full.core
			serviceMust(t, m.modular.SetZPUint64(7))
			p := polynomialQueryTerms(t, m, []int32{2, -3}, [][]MonomialPower{{{0, 1}}, nil})
			m.integer.budget.FailAfter(ordinal)
			result, e := full.GlexMonic(p)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual monic refusal", e)
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result}, nil, 0)
			m.integer.budget.FailAfter(-1)
			serviceMust(t, full.CleanupGlexMonic())
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result}, nil, 0)
			if e == nil {
				result = polynomialCaller(t, m, result)
				polynomialQueryExpect(t, m, result, map[[3]uint32]int64{{1, 0, 0}: 1, {}: 2})
				success = true
				t.Log("monic every actual refusal endpoint", ordinal)
			} else if result != (Polynomial{}) {
				serviceMust(t, m.CleanupValue(&result))
			}
		})
	}
	if !success {
		t.Fatal("no complete monic source body")
	}
}

func TestPolynomialGlexMonicBigFieldEveryNumericPrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 256 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			m := full.core
			z := m.integer
			var prime Int
			serviceMust(t, z.SetLimbs(&prime, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
			serviceMust(t, m.modular.SetZP(&prime))
			serviceMust(t, z.Del(&prime))
			x, e := m.monomials.MkVariable(0, 1)
			serviceMust(t, e)
			unit, e := m.monomials.Unit()
			serviceMust(t, e)
			p, e := m.MkTerms([]Int{z.two64, z.two64}, []Monomial{x, unit})
			serviceMust(t, e)
			p = polynomialCaller(t, m, p)
			t.Cleanup(func() { z.budget.FailAfter(-1); serviceMust(t, full.CleanupGlexMonic()) })
			z.budget.FailAfter(ordinal)
			result, e := full.GlexMonic(p)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual BIG monic allocation", e)
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result}, nil, 0)
			z.budget.FailAfter(-1)
			serviceMust(t, full.CleanupGlexMonic())
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result}, nil, 0)
			if e == nil {
				result = polynomialCaller(t, m, result)
				polynomialQueryExpect(t, m, result, map[[3]uint32]int64{{1, 0, 0}: 1, {}: 1})
				success = true
				t.Log("actual BIG field source completion after every refusal", ordinal)
			} else if result != (Polynomial{}) {
				serviceMust(t, m.CleanupValue(&result))
			}
		})
	}
	if !success {
		t.Fatal("no complete BIG monic route")
	}
}
