// Source: src/math/polynomial/upolynomial.cpp:1734-1764.
// Licensed under the MIT license.
package mpnum

import (
	"fmt"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
	"testing"
)

func TestUPolynomialEvalSignBinaryEveryActualReservationPrefix(t *testing.T) {
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
			var b BinaryRational
			serviceMust(t, c.config.Binary.SetInt32(&b, math.MaxInt32))
			baseline := z.budget.Used() - integerActualOwnedBytes(z) - p.binaryEvaluationHistoryBytes
			z.budget.FailAfter(prefix)
			sign, body := p.EvalSignBinary(v, &b)
			q := &p.binaryEvaluation
			r := p.binaryEvaluationReceipt()
			context := q.context()
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal(body)
			}
			extra := []*Int{&b.num, &q.values[0], &q.values[1]}
			if q.addMul.frame != nil {
				extra = append(extra, &q.addMul.frame.tmp)
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{poly, constant}, extra, nil)
			if want := baseline + integerActualOwnedBytes(z) + p.binaryEvaluationHistoryBytes; z.budget.Used() != want {
				t.Fatal("before-lift cold graph", z.budget.Used(), want)
			}
			if q.numeric.cell.object != (memory.ObjectRef{}) || polynomialNegNumericPending(&q.numeric) {
				acquired = true
			}
			if q.history != nil {
				copiedOwner := *p
				copiedOwner.self = &copiedOwner
				copiedOwner.binaryEvaluation.owner = &copiedOwner
				copiedReceipt := copiedOwner.binaryEvaluationReceipt()
				if e := copiedOwner.cleanupBinaryEvaluationWithReceipt(&copiedReceipt, true); e != ErrUPolynomialOwner {
					t.Fatal("copied owner obtained exact allocating scope cleanup", e)
				}
			}
			before := *q
			z.budget.FailAfter(0)
			if got, e := p.EvalSignBinary(UPolynomialView{}, nil); got != 0 || e != nil {
				t.Fatal("erased zero", got, e)
			}
			if got, e := p.EvalSignBinary(cv, nil); got != -1 || e != nil {
				t.Fatal("erased constant", got, e)
			}
			if !reflect.DeepEqual(before, *q) {
				t.Fatal("erased calls changed actual older invocation")
			}
			z.budget.FailAfter(-1)
			serviceMust(t, p.cleanupBinaryEvaluationWithReceipt(&r, false))
			serviceMust(t, c.config.Binary.Del(&b))
			serviceMust(t, p.cleanupBinaryEvaluationWithReceipt(&r, true))
			if q.history != nil {
				old := *q.history
				// A real later inline two-coefficient body reuses the actual embedded frame.
				serviceMust(t, c.config.Binary.SetInt32(&b, 1))
				linear := algebraicPolynomial(t, p, 1, 1)
				lv, e := p.wholeView(linear)
				serviceMust(t, e)
				z.budget.FailAfter(0)
				if sign, e := p.EvalSignBinary(lv, &b); sign != 1 || e != nil {
					t.Fatal("later inline Horner", sign, e)
				}
				z.budget.FailAfter(-1)
				if *r.history != old {
					t.Fatal("later body changed charged first diagnostic/child tuples")
				}
				current := p.binaryEvaluation
				serviceMust(t, p.cleanupBinaryEvaluationWithReceipt(&r, true))
				if !reflect.DeepEqual(current, p.binaryEvaluation) {
					t.Fatal("old ACK touched later invocation")
				}
				z.binaryEvaluationCaller = p
				if context.roots(z) != nil {
					t.Fatal("saved context retargeted actual reused member")
				}
				z.binaryEvaluationCaller = nil
				serviceMust(t, c.config.Binary.Del(&b))
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

func TestUPolynomialEvalSignBinaryLiteralDyadicHornerBranches(t *testing.T) {
	c, p := algebraicFixture(t)
	poly := algebraicPolynomial(t, p, -2, 0, 1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	for _, test := range []struct {
		num  int32
		k    uint32
		sign int
	}{{3, 1, 1}, {1, 0, -1}, {-1, 1, -1}} {
		var b BinaryRational
		serviceMust(t, c.config.Binary.SetInt32Power(&b, test.num, test.k))
		c.config.Rationals.z.budget.FailAfter(0)
		sign, e := p.EvalSignBinary(v, &b)
		c.config.Rationals.z.budget.FailAfter(-1)
		if e != nil || sign != test.sign {
			t.Fatal("independent x²−2 dyadic sign", test, sign, e)
		}
		serviceMust(t, c.config.Binary.Del(&b))
	}
}
