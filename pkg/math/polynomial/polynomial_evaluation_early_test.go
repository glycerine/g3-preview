// Source: src/math/polynomial/polynomial.cpp:6509-6550.
// Licensed under the MIT license. Author: Leonardo de Moura.
package polynomial

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialEvaluationEarlyRoutesLeaveRetainedInvocationUntouched(t *testing.T) {
	found := false
	for ordinal := int64(0); ordinal < 30 && !found; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			s, _ := serviceNew(t)
			q := s.Manager
			m, p, _ := polynomialEvalFixtureWithRationals(t, q, []polynomialEvalTerm{{1, [3]uint32{3, 2, 0}}, {2, [3]uint32{1, 0, 0}}, {7, [3]uint32{}}})
			bounds, e := util.NewBinaryRationalManager(q.Z, util.BinaryRationalConfig{Diagnostics: s})
			serviceMust(t, e)
			t.Cleanup(bounds.Close)
			vm, e := util.NewBinaryIntervalManager(bounds)
			serviceMust(t, e)
			t.Cleanup(vm.Close)
			v := polynomialEvalRationalMap{manager: q}
			for i := range v.values {
				v.values[i] = util.NewRational()
				serviceMust(t, q.SetInt32(&v.values[i], 2))
			}
			output := util.NewRational()
			q.BudgetField.FailAfter(ordinal)
			e = m.EvaluateRational(p, &v, &output)
			// Census the endangered prefix before the fault epoch is lifted.
			iv := polynomialEvalIntervalMap{manager: vm}
			io := util.NewBinaryInterval()
			defer func() {
				q.BudgetField.FailAfter(-1)
				serviceMust(t, m.CleanupEvaluation())
				serviceMust(t, vm.Del(&io))
				for i := range v.values {
					serviceMust(t, q.Del(&v.values[i]))
				}
				serviceMust(t, q.Del(&output))
			}()
			polynomialEvaluationIntervalAudit(t, m, s, vm, p, &iv, &io)
			q.BudgetField.FailAfter(-1)
			if e != nil && m.evaluation.active {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(e)
				}
				found = true
				lex, groups, cell, raw := m.evaluation.lexGeneration, m.evaluation.groups, m.evaluation.numericPending, m.evaluation.allocatorPending
				oldCause := m.evaluation.bodyCause
				early := util.NewRational()
				serviceMust(t, q.SetInt32(&early, 19))
				serviceMust(t, m.EvaluateRational(m.core.zero, &v, &early))
				if early != util.NewRational() {
					t.Fatal("source zero reset", early)
				}
				serviceMust(t, m.EvaluateRational(m.core.unit, &v, &early))
				if (*early.BorrowNumerator()) != util.NewInt(1) || (*early.BorrowDenominator()) != util.NewInt(1) {
					t.Fatal("source constant set", early)
				}
				serviceMust(t, vm.SetInteger(&io, &(*early.BorrowNumerator())))
				serviceMust(t, m.EvaluateInterval(m.core.zero, &iv, &io))
				if (*(*io.BorrowLower()).BorrowNumerator()) != util.NewInt(0) || (*(*io.BorrowUpper()).BorrowNumerator()) != util.NewInt(0) {
					t.Fatal("source interval zero reset")
				}
				serviceMust(t, m.EvaluateInterval(m.core.unit, &iv, &io))
				if (*(*io.BorrowLower()).BorrowNumerator()) != util.NewInt(1) || (*(*io.BorrowUpper()).BorrowNumerator()) != util.NewInt(1) {
					t.Fatal("source interval constant set")
				}
				if !m.evaluation.active || m.evaluation.lexGeneration != lex || m.evaluation.groups != groups || m.evaluation.numericPending != cell || m.evaluation.allocatorPending != raw || m.evaluation.bodyCause != oldCause {
					t.Fatal("early route changed unrelated retained invocation")
				}
				serviceMust(t, q.Del(&early))
			}
			serviceMust(t, m.CleanupEvaluation())
			serviceMust(t, vm.Del(&io))
			for i := range v.values {
				serviceMust(t, q.Del(&v.values[i]))
			}
			serviceMust(t, q.Del(&output))
		})
	}
	if !found {
		t.Fatal("real retained invocation not reached")
	}
}

func TestPolynomialEvaluationEarlyCopyCapturesOnlyNewPhysicalRoots(t *testing.T) {
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
			oldOutput, out := util.NewRational(), util.NewRational()
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
				serviceMust(t, q.Del(&out))
				for i := range v.values {
					serviceMust(t, q.Del(&v.values[i]))
				}
				serviceMust(t, m.core.CleanupValue(&constant))
			}()
			inputs := []*util.Int{&holder, &(*oldOutput.BorrowNumerator()), &(*oldOutput.BorrowDenominator()), &(*out.BorrowNumerator()), &(*out.BorrowDenominator())}
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
			e = m.EvaluateRational(constant, &v, &out)
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
