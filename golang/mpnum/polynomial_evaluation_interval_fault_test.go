// Source: src/math/polynomial/polynomial.cpp:6436-6550; src/util/mpbqi.h;
// src/util/basic_interval.h. Licensed under the MIT license.
// Author: Leonardo de Moura.
package mpnum

import (
	"fmt"
	"math/big"
	"testing"
)

func polynomialEvaluationIntervalAudit(t *testing.T, m *MultivariatePolynomialManager, s *RationalService, vm *BinaryIntervalManager, p Polynomial, values *polynomialEvalIntervalMap, out *BinaryInterval) {
	t.Helper()
	b := vm.bounds
	inputs := []*Int{&values.value.lower.num, &values.value.upper.num, &out.lower.num, &out.upper.num, &vm.mulCurr.num, &vm.mulMax.num, &vm.mulMin.num, &b.tmp, &b.tmp2, &b.addMulTmp.num, &b.selectIntTmp1, &b.selectIntTmp2, &b.selectSmallTmp, &b.selectSmallTmp1.num, &b.selectSmallTmp2.num, &b.divTmp1, &b.divTmp2, &b.divTmp3}
	for i := range b.display.values {
		inputs = append(inputs, &b.display.values[i])
	}
	for _, r := range []*Rational{&s.zero, &s.one, &s.minusOne, &s.infZero.first, &s.infZero.second, &s.infOne.first, &s.infOne.second, &s.infMinusOne.first, &s.infMinusOne.second, &s.infIntZero.first, &s.infIntOne.first, &s.infIntMinusOne.first} {
		inputs = append(inputs, &r.num, &r.den)
	}
	for i := range s.cache {
		inputs = append(inputs, &s.cache[i].num, &s.cache[i].den)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, inputs, s.headerBytes+s.cacheBytes+b.headerBytes+vm.headerBytes)
}
func TestPolynomialEvaluationIntervalEveryReservation(t *testing.T) {
	failed := 0
	success := false
	for ordinal := int64(0); ordinal < 500; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			s, _ := serviceNew(t)
			q := s.manager
			m, p, _ := polynomialEvalFixtureWithRationals(t, q, []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {2, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
			bounds, e := NewBinaryRationalManager(q.z, BinaryRationalConfig{Diagnostics: s})
			serviceMust(t, e)
			t.Cleanup(bounds.Close)
			vm, e := NewBinaryIntervalManager(bounds)
			serviceMust(t, e)
			t.Cleanup(vm.Close)
			values := polynomialEvalIntervalMap{manager: vm}
			digits := make([]uint32, 80)
			digits[0], digits[79] = 3, 1
			serviceMust(t, q.z.SetLimbs(&values.value.lower.num, digits))
			serviceMust(t, bounds.Copy(&values.value.upper, &values.value.lower))
			out := NewBinaryInterval()
			q.budget.FailAfter(ordinal)
			e = m.EvaluateInterval(p, &values, &out)
			polynomialEvaluationIntervalAudit(t, m, s, vm, p, &values, &out)
			if e == nil {
				success = true
			} else {
				failed++
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("exact reached physical/metadata reservation", e)
				}
			}
			q.budget.FailAfter(-1)
			serviceMust(t, m.CleanupEvaluation())
			polynomialEvaluationIntervalAudit(t, m, s, vm, p, &values, &out)
			if e == nil {
				base := binaryRat(t, bounds, &values.value.lower)
				base.Add(base, new(big.Rat).SetInt64(1))
				base.Mul(base, base)
				if binaryRat(t, bounds, &out.lower).Cmp(base) != 0 || binaryRat(t, bounds, &out.upper).Cmp(base) != 0 {
					t.Fatal("independent point (x+1)^2")
				}
			}
			serviceMust(t, vm.Del(&values.value))
			serviceMust(t, vm.Del(&out))
		})
		if success {
			break
		}
	}
	if !success || failed == 0 {
		t.Fatal("real complete interval prefix sweep", failed, success)
	}
}
