// Source: src/math/polynomial/upolynomial.cpp:1767-1797.
// Licensed under the MIT license.
package mpnum

import (
	"fmt"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
	"testing"
)

func TestUPolynomialEvalSignRationalEveryActualReservationPrefix(t *testing.T) {
	success, acquired, denRefusal := false, false, false
	for prefix := int64(0); prefix < 64 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, p := algebraicFixture(t)
			z := c.config.Rationals.z
			poly := algebraicPolynomial(t, p, 1, math.MaxInt32, 1)
			constant := algebraicPolynomial(t, p, -7)
			v, e := p.wholeView(poly)
			serviceMust(t, e)
			cv, e := p.wholeView(constant)
			serviceMust(t, e)
			var b Rational
			serviceMust(t, c.config.Rationals.SetFraction(&b, 1, (1<<40)+3))
			baseline := z.budget.Used() - integerActualOwnedBytes(z) - p.rationalEvaluationHistoryBytes
			z.budget.FailAfter(prefix)
			sign, body := p.EvalSignRational(v, &b)
			q := &p.rationalEvaluation
			if body != nil && q.constructed == 1 {
				denRefusal = true
				if q.values[1] != (Int{}) || q.values[2] != (Int{}) || q.sourceRemaining > 1 {
					t.Fatal("unreached r/ak constructor")
				}
			}
			r := p.rationalEvaluationReceipt()
			context := q.context()
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal(body)
			}
			extra := []*Int{&b.num, &b.den, &q.values[0], &q.values[1], &q.values[2]}
			if q.addMul.frame != nil {
				extra = append(extra, &q.addMul.frame.tmp)
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{poly, constant}, extra, nil)
			if want := baseline + integerActualOwnedBytes(z) + p.rationalEvaluationHistoryBytes; z.budget.Used() != want {
				t.Fatal("before-lift cold graph", z.budget.Used(), want)
			}
			if q.numeric.cell.object != (memory.ObjectRef{}) || polynomialNegNumericPending(&q.numeric) {
				acquired = true
			}
			if q.history != nil {
				copiedOwner := *p
				copiedOwner.self = &copiedOwner
				copiedOwner.rationalEvaluation.owner = &copiedOwner
				copiedReceipt := copiedOwner.rationalEvaluationReceipt()
				if e := copiedOwner.cleanupRationalEvaluationWithReceipt(&copiedReceipt, true); e != ErrUPolynomialOwner {
					t.Fatal("copied owner obtained exact allocating scope cleanup", e)
				}
			}
			before := *q
			z.budget.FailAfter(0)
			if got, e := p.EvalSignRational(UPolynomialView{}, nil); got != 0 || e != nil {
				t.Fatal("erased zero", got, e)
			}
			if got, e := p.EvalSignRational(cv, nil); got != -1 || e != nil {
				t.Fatal("erased constant", got, e)
			}
			if !reflect.DeepEqual(before, *q) {
				t.Fatal("erased calls changed actual older invocation")
			}
			z.budget.FailAfter(-1)
			serviceMust(t, p.cleanupRationalEvaluationWithReceipt(&r, false))
			serviceMust(t, c.config.Rationals.Del(&b))
			serviceMust(t, p.cleanupRationalEvaluationWithReceipt(&r, true))
			if q.history != nil {
				old := *q.history
				// A real later inline two-coefficient body reuses the actual embedded frame.
				serviceMust(t, c.config.Rationals.SetInt32(&b, 1))
				linear := algebraicPolynomial(t, p, 1, 1)
				lv, e := p.wholeView(linear)
				serviceMust(t, e)
				z.budget.FailAfter(0)
				if sign, e := p.EvalSignRational(lv, &b); sign != 1 || e != nil {
					t.Fatal("later inline Horner", sign, e)
				}
				z.budget.FailAfter(-1)
				if *r.history != old {
					t.Fatal("later body changed charged first diagnostic/child tuples")
				}
				current := p.rationalEvaluation
				serviceMust(t, p.cleanupRationalEvaluationWithReceipt(&r, true))
				if !reflect.DeepEqual(current, p.rationalEvaluation) {
					t.Fatal("old ACK touched later invocation")
				}
				z.rationalEvaluationCaller = p
				if context.roots(z) != nil {
					t.Fatal("saved context retargeted actual reused member")
				}
				z.rationalEvaluationCaller = nil
				serviceMust(t, c.config.Rationals.Del(&b))
			}
			if body == nil {
				success = true
				if sign != 1 {
					t.Fatal("independent positive Horner sign", sign)
				}
			}
		})
	}
	if !success || !acquired || !denRefusal {
		t.Fatal("actual campaign did not reach success/pending leaf", success, acquired)
	}
}

func TestUPolynomialEvalSignRationalLiteralHornerBranches(t *testing.T) {
	c, p := algebraicFixture(t)
	z := c.config.Rationals.z
	poly := algebraicPolynomial(t, p, -2, 0, 1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	for _, tc := range []struct {
		n    int64
		d    uint64
		want int
	}{{3, 2, 1}, {1, 1, -1}, {-1, 2, -1}} {
		var b Rational
		serviceMust(t, c.config.Rationals.SetFraction(&b, tc.n, tc.d))
		z.budget.FailAfter(0)
		got, e := p.EvalSignRational(v, &b)
		z.budget.FailAfter(-1)
		if e != nil || got != tc.want {
			t.Fatal(tc, got, e)
		}
		serviceMust(t, p.CleanupRationalEvaluation())
		serviceMust(t, c.config.Rationals.Del(&b))
	}
}
