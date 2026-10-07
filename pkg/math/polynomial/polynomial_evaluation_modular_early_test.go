// Source: src/math/polynomial/polynomial.cpp:6509-6543.
// Licensed under the MIT license. Author: Leonardo de Moura.
package polynomial

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialEvaluationModularEarlyCopyCapturesPhysicalRoots(t *testing.T) {
	found, success := false, false
	for ordinal := int64(0); ordinal < 50; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{3, 2, 0}}, {2, [3]uint32{1, 0, 0}}, {7, [3]uint32{}}})
			q := m.core.config.Rationals
			digits := make([]uint32, 80)
			digits[0], digits[79] = 3, 1
			coefficient := util.Int{}
			serviceMust(t, q.Integers().SetLimbs(&coefficient, digits))
			constant, e := m.core.MkConstantTake(&coefficient)
			serviceMust(t, e)
			serviceMust(t, m.core.IncRef(constant))
			v := polynomialEvalRationalMap{manager: q}
			for i := range v.values {
				v.values[i] = util.NewRational()
				serviceMust(t, q.SetInt32(&v.values[i], 2))
			}
			oldOutput := util.NewRational()
			out, value := util.Int{}, util.NewInt(2)
			// This real caller coefficient occupies the remaining descriptor on
			// the two-slot cold page, so a later Copy reaches page reservation
			// after its physical numeric object has actually been allocated.
			holder := util.Int{}
			serviceMust(t, q.Integers().SetLimbs(&holder, digits))
			defer func() {
				q.Budget().FailAfter(-1)
				serviceMust(t, m.CleanupEvaluation())
				serviceMust(t, q.Integers().Del(&holder))
				serviceMust(t, q.Del(&oldOutput))
				serviceMust(t, q.Integers().Del(&out))
				for i := range v.values {
					serviceMust(t, q.Del(&v.values[i]))
				}
				serviceMust(t, m.core.CleanupValue(&constant))
			}()
			inputs := []*util.Int{&holder, &(*oldOutput.BorrowNumerator()), &(*oldOutput.BorrowDenominator()), &out}
			for i := range v.values {
				inputs = append(inputs, &(*v.values[i].BorrowNumerator()), &(*v.values[i].BorrowDenominator()))
			}
			q.Budget().FailAfter(0)
			e = m.EvaluateRational(p, &v, &oldOutput)
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, constant}, inputs, 0)
			q.Budget().FailAfter(-1)
			if !numericPhysicalReservationRefusal(e) || !m.lexPending || !m.evaluation.active {
				t.Fatal("actual earlier lexical stop", e)
			}
			oldLex, oldCause := m.evaluation.lexGeneration, m.evaluation.bodyCause
			q.Budget().FailAfter(ordinal)
			e = m.UnivariateEvaluate(constant, 0, &value, &out)
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, constant}, inputs, 0)
			if e == nil {
				success = true
			} else if !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual early Copy refusal", e)
			}
			if m.evaluation.lexGeneration != oldLex || m.evaluation.bodyCause != oldCause {
				t.Fatal("early Copy replaced older invocation")
			}
			raw, inspect := q.Integers().Allocator.PendingRoots()
			serviceMust(t, inspect)
			if q.Integers().PendingCell.Object != (util.ObjectRef{}) {
				found = true
				if m.evaluation.numericPending != q.Integers().PendingCell.Object {
					t.Fatal("actual new numeric object receipt")
				}
			}
			if raw != ([3]util.Block{}) {
				found = true
				if m.evaluation.allocatorPending != raw {
					t.Fatal("actual new allocator roots receipt")
				}
			}
			q.Budget().FailAfter(-1)
			serviceMust(t, m.CleanupEvaluation())
			if q.Integers().PendingCell.Object != (util.ObjectRef{}) {
				t.Fatal("operation cleanup omitted early Copy object")
			}
			raw, inspect = q.Integers().Allocator.PendingRoots()
			serviceMust(t, inspect)
			if raw != ([3]util.Block{}) {
				t.Fatal("operation cleanup omitted early Copy raw")
			}
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, constant}, inputs, 0)
		})
		if success {
			break
		}
	}
	if !found || !success {
		t.Fatal("complete early Copy physical-prefix campaign", found, success)
	}
}
