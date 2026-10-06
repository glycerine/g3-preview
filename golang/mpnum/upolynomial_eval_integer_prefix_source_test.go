// Source: src/math/polynomial/upolynomial.cpp:1801-1820.
// Licensed under the MIT license.
package mpnum

import (
	"fmt"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
	"testing"
)

func TestUPolynomialEvalSignIntEveryActualReservationPrefix(t *testing.T) {
	success, acquired := false, false
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
			var b Int
			serviceMust(t, z.SetInt32(&b, math.MaxInt32))
			baseline := z.budget.Used() - integerActualOwnedBytes(z) - p.integerEvaluationHistoryBytes
			z.budget.FailAfter(prefix)
			sign, body := p.EvalSignInt(v, &b)
			q := &p.integerEvaluation
			r := p.integerEvaluationReceipt()
			context := q.context()
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal(body)
			}
			extra := []*Int{&b, &q.values[0]}
			if q.addMul.frame != nil {
				extra = append(extra, &q.addMul.frame.tmp)
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{poly, constant}, extra, nil)
			if want := baseline + integerActualOwnedBytes(z) + p.integerEvaluationHistoryBytes; z.budget.Used() != want {
				t.Fatal("before-lift cold graph", z.budget.Used(), want)
			}
			if q.numeric.cell.object != (memory.ObjectRef{}) || polynomialNegNumericPending(&q.numeric) {
				acquired = true
			}
			if q.history != nil {
				copiedOwner := *p
				copiedOwner.self = &copiedOwner
				copiedOwner.integerEvaluation.owner = &copiedOwner
				copiedReceipt := copiedOwner.integerEvaluationReceipt()
				if e := copiedOwner.cleanupIntegerEvaluationWithReceipt(&copiedReceipt, true); e != ErrUPolynomialOwner {
					t.Fatal("copied owner obtained exact allocating scope cleanup", e)
				}
			}
			before := *q
			z.budget.FailAfter(0)
			if got, e := p.EvalSignInt(UPolynomialView{}, nil); got != 0 || e != nil {
				t.Fatal("erased zero", got, e)
			}
			if got, e := p.EvalSignInt(cv, nil); got != -1 || e != nil {
				t.Fatal("erased constant", got, e)
			}
			if !reflect.DeepEqual(before, *q) {
				t.Fatal("erased calls changed actual older invocation")
			}
			z.budget.FailAfter(-1)
			serviceMust(t, p.cleanupIntegerEvaluationWithReceipt(&r, false))
			serviceMust(t, z.Del(&b))
			serviceMust(t, p.cleanupIntegerEvaluationWithReceipt(&r, true))
			if q.history != nil {
				old := *q.history
				// A real later inline two-coefficient body reuses the actual embedded frame.
				serviceMust(t, z.SetInt32(&b, 1))
				linear := algebraicPolynomial(t, p, 1, 1)
				lv, e := p.wholeView(linear)
				serviceMust(t, e)
				z.budget.FailAfter(0)
				if sign, e := p.EvalSignInt(lv, &b); sign != 1 || e != nil {
					t.Fatal("later inline Horner", sign, e)
				}
				z.budget.FailAfter(-1)
				if *r.history != old {
					t.Fatal("later body changed charged first diagnostic/child tuples")
				}
				current := p.integerEvaluation
				serviceMust(t, p.cleanupIntegerEvaluationWithReceipt(&r, true))
				if !reflect.DeepEqual(current, p.integerEvaluation) {
					t.Fatal("old ACK touched later invocation")
				}
				z.integerEvaluationCaller = p
				if context.roots(z) != nil {
					t.Fatal("saved context retargeted actual reused member")
				}
				z.integerEvaluationCaller = nil
				serviceMust(t, z.Del(&b))
			}
			if body == nil {
				success = true
				if sign != 1 {
					t.Fatal("independent positive Horner sign", sign)
				}
			}
		})
	}
	if !success || !acquired {
		t.Fatal("actual campaign did not reach success/pending leaf", success, acquired)
	}
}

func TestUPolynomialEvalSignIntLiteralHornerBranches(t *testing.T) {
	c, p := algebraicFixture(t)
	z := c.config.Rationals.z
	poly := algebraicPolynomial(t, p, -2, 0, 1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	for _, tc := range []struct {
		n    int32
		want int
	}{{0, -1}, {1, -1}, {2, 1}, {-2, 1}} {
		b := NewInt(tc.n)
		z.budget.FailAfter(0)
		got, e := p.EvalSignInt(v, &b)
		z.budget.FailAfter(-1)
		if e != nil || got != tc.want {
			t.Fatal(tc, got, e)
		}
		serviceMust(t, p.CleanupIntegerEvaluation())
		serviceMust(t, z.Del(&b))
	}
}
