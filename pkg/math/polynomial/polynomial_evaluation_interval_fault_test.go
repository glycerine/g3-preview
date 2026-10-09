// Source: src/math/polynomial/polynomial.cpp:6436-6550; src/util/mpbqi.h;
// src/util/basic_interval.h.
package polynomial

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func polynomialEvaluationIntervalAudit(t *testing.T, m *MultivariatePolynomialManager, s *util.RationalService, vm *util.BinaryIntervalManager, p Polynomial, values *polynomialEvalIntervalMap, out *util.BinaryInterval) {
	t.Helper()
	b, e := vm.Bounds()
	serviceMust(t, e)
	inputs := []*util.Int{&(*(*values.value.BorrowLower()).BorrowNumerator()), &(*(*values.value.BorrowUpper()).BorrowNumerator()), &(*(*out.BorrowLower()).BorrowNumerator()), &(*(*out.BorrowUpper()).BorrowNumerator()), &vm.MulCurr.Num, &vm.MulMax.Num, &vm.MulMin.Num, &b.Tmp, &b.Tmp2, &b.AddMulTmp.Num, &b.SelectIntTmp1, &b.SelectIntTmp2, &b.SelectSmallTmp, &b.SelectSmallTmp1.Num, &b.SelectSmallTmp2.Num, &b.DivTmp1, &b.DivTmp2, &b.DivTmp3}
	for i := range b.Display.Values {
		inputs = append(inputs, &b.Display.Values[i])
	}
	for _, r := range []*util.Rational{&s.ZeroField, &s.OneField, &s.MinusOneField, &s.InfZeroField.First, &s.InfZeroField.Second, &s.InfOneField.First, &s.InfOneField.Second, &s.InfMinusOneField.First, &s.InfMinusOneField.Second, &s.InfIntZeroField.First, &s.InfIntOneField.First, &s.InfIntMinusOneField.First} {
		inputs = append(inputs, &(*r.BorrowNumerator()), &(*r.BorrowDenominator()))
	}
	for i := range s.Cache {
		inputs = append(inputs, &s.Cache[i].Num, &s.Cache[i].Den)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, inputs, s.HeaderBytes+b.HeaderBytes+vm.HeaderBytes)
}
func TestPolynomialEvaluationIntervalEveryReservation(t *testing.T) {
	failed := 0
	success := false
	for ordinal := int64(0); ordinal < 500; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			s, _ := serviceNew(t)
			q := s.Manager
			m, p, _ := polynomialEvalFixtureWithRationals(t, q, []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {2, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
			bounds, e := util.NewBinaryRationalManager(q.Z, util.BinaryRationalConfig{Diagnostics: s})
			serviceMust(t, e)
			t.Cleanup(bounds.Close)
			vm, e := util.NewBinaryIntervalManager(bounds)
			serviceMust(t, e)
			t.Cleanup(vm.Close)
			values := polynomialEvalIntervalMap{manager: vm}
			digits := make([]uint32, 80)
			digits[0], digits[79] = 3, 1
			serviceMust(t, q.Z.SetLimbs(&(*(*values.value.BorrowLower()).BorrowNumerator()), digits))
			serviceMust(t, bounds.Copy(&(*values.value.BorrowUpper()), &(*values.value.BorrowLower())))
			out := util.NewBinaryInterval()
			q.BudgetField.FailAfter(ordinal)
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
			q.BudgetField.FailAfter(-1)
			serviceMust(t, m.CleanupEvaluation())
			polynomialEvaluationIntervalAudit(t, m, s, vm, p, &values, &out)
			if e == nil {
				base := binaryRat(t, bounds, &(*values.value.BorrowLower()))
				base.Add(base, new(big.Rat).SetInt64(1))
				base.Mul(base, base)
				if binaryRat(t, bounds, &(*out.BorrowLower())).Cmp(base) != 0 || binaryRat(t, bounds, &(*out.BorrowUpper())).Cmp(base) != 0 {
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
