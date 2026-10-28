package polynomial

// Source: src/math/polynomial/polynomial.cpp:6509-6543.

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialEvaluationModularEarlyCopyCapturesPhysicalRoots(t *testing.T) {
	found, success := false, false
	for ordinal := int64(0); ordinal < 50; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, _, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{}}})
			p := polynomialLexWrappedPolynomial(t, m)
			q := m.core.config.Rationals
			digits := make([]uint32, 80)
			digits[0], digits[79] = 3, 1
			coefficient := unumeral.Int{}
			serviceMust(t, q.Integers().SetLimbs(&coefficient, digits))
			constant, e := m.core.MkConstantTake(&coefficient)
			serviceMust(t, e)
			serviceMust(t, m.core.IncRef(constant))
			v := polynomialEvalRationalMap{manager: q}
			for i := range v.values {
				v.values[i] = unumeral.NewRational()
				serviceMust(t, q.SetInt32(&v.values[i], 2))
			}
			oldOutput := unumeral.NewRational()
			out, value := unumeral.Int{}, unumeral.NewInt(2)
			// This real caller coefficient occupies the remaining descriptor on
			// the two-slot cold page, so a later Copy reaches page reservation
			// after its physical numeric object has actually been allocated.
			holder := unumeral.Int{}
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
			inputs := []*unumeral.Int{&holder, &(*oldOutput.BorrowNumerator()), &(*oldOutput.BorrowDenominator()), &out}
			for i := range v.values {
				inputs = append(inputs, &(*v.values[i].BorrowNumerator()), &(*v.values[i].BorrowDenominator()))
			}
			e = m.EvaluateRational(p, &v, &oldOutput)
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, constant}, inputs, 0)
			q.Budget().FailAfter(-1)
			if (e != arena.ErrBounds && e != ErrPolynomialAssertion) || !m.lexPending || !m.evaluation.active {
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
			if q.Integers().PendingCell.Object != (umemory.ObjectRef{}) {
				found = true
				if m.evaluation.numericPending != q.Integers().PendingCell.Object {
					t.Fatal("actual new numeric object receipt")
				}
			}
			if raw != ([3]umemory.Block{}) {
				found = true
				if m.evaluation.allocatorPending != raw {
					t.Fatal("actual new allocator roots receipt")
				}
			}
			q.Budget().FailAfter(-1)
			serviceMust(t, m.CleanupEvaluation())
			if q.Integers().PendingCell.Object != (umemory.ObjectRef{}) {
				t.Fatal("operation cleanup omitted early Copy object")
			}
			raw, inspect = q.Integers().Allocator.PendingRoots()
			serviceMust(t, inspect)
			if raw != ([3]umemory.Block{}) {
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
