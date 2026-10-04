// Source: src/math/polynomial/polynomial.cpp:6509-6543.
// Licensed under the MIT license. Author: Leonardo de Moura.
package mpnum

import (
	"fmt"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialEvaluationModularEarlyCopyCapturesPhysicalRoots(t *testing.T) {
	found, success := false, false
	for ordinal := int64(0); ordinal < 50; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{3, 2, 0}}, {2, [3]uint32{1, 0, 0}}, {7, [3]uint32{}}})
			q := m.core.config.Rationals
			digits := make([]uint32, 80)
			digits[0], digits[79] = 3, 1
			coefficient := Int{}
			serviceMust(t, q.z.SetLimbs(&coefficient, digits))
			constant, e := m.core.MkConstantTake(&coefficient)
			serviceMust(t, e)
			serviceMust(t, m.core.IncRef(constant))
			v := polynomialEvalRationalMap{manager: q}
			for i := range v.values {
				v.values[i] = NewRational()
				serviceMust(t, q.SetInt32(&v.values[i], 2))
			}
			oldOutput := NewRational()
			out, value := Int{}, NewInt(2)
			// This real caller coefficient occupies the remaining descriptor on
			// the two-slot cold page, so a later Copy reaches page reservation
			// after its physical numeric object has actually been allocated.
			holder := Int{}
			serviceMust(t, q.z.SetLimbs(&holder, digits))
			defer func() {
				q.budget.FailAfter(-1)
				serviceMust(t, m.CleanupEvaluation())
				serviceMust(t, q.z.Del(&holder))
				serviceMust(t, q.Del(&oldOutput))
				serviceMust(t, q.z.Del(&out))
				for i := range v.values {
					serviceMust(t, q.Del(&v.values[i]))
				}
				serviceMust(t, m.core.CleanupValue(&constant))
			}()
			inputs := []*Int{&holder, &oldOutput.num, &oldOutput.den, &out}
			for i := range v.values {
				inputs = append(inputs, &v.values[i].num, &v.values[i].den)
			}
			q.budget.FailAfter(0)
			e = m.EvaluateRational(p, &v, &oldOutput)
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, constant}, inputs, 0)
			q.budget.FailAfter(-1)
			if !numericPhysicalReservationRefusal(e) || !m.lexPending || !m.evaluation.active {
				t.Fatal("actual earlier lexical stop", e)
			}
			oldLex, oldCause := m.evaluation.lexGeneration, m.evaluation.bodyCause
			q.budget.FailAfter(ordinal)
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
			raw, inspect := q.z.allocator.PendingRoots()
			serviceMust(t, inspect)
			if q.z.pendingCell.object != (memory.ObjectRef{}) {
				found = true
				if m.evaluation.numericPending != q.z.pendingCell.object {
					t.Fatal("actual new numeric object receipt")
				}
			}
			if raw != ([3]memory.Block{}) {
				found = true
				if m.evaluation.allocatorPending != raw {
					t.Fatal("actual new allocator roots receipt")
				}
			}
			q.budget.FailAfter(-1)
			serviceMust(t, m.CleanupEvaluation())
			if q.z.pendingCell.object != (memory.ObjectRef{}) {
				t.Fatal("operation cleanup omitted early Copy object")
			}
			raw, inspect = q.z.allocator.PendingRoots()
			serviceMust(t, inspect)
			if raw != ([3]memory.Block{}) {
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
