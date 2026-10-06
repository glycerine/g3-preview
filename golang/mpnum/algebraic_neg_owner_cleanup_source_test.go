// Source: src/math/polynomial/algebraic_numbers.cpp:1565-1577;
// src/math/polynomial/upolynomial.cpp:1734-1764. Explicit core owning cleanup.
package mpnum

import (
	"errors"
	"fmt"
	"math"
	"testing"
)

func TestAlgebraicNegCoreCloseOwnsActualRetainedHornerChild(t *testing.T) {
	reached := false
	for prefix := int64(0); prefix < 96 && !reached; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, p, _ := algebraicCompareFixtureProfile(t, true)
			z := c.config.Rationals.z
			t.Cleanup(func() { z.budget.FailAfter(-1); serviceMust(t, c.CleanupNegation()) })
			original := algebraicPolynomial(t, p, 1, math.MinInt32, 1)
			v, e := p.wholeView(original)
			serviceMust(t, e)
			var lower, upper BinaryRational
			var upperInt Int
			serviceMust(t, c.config.Binary.SetInt32(&lower, math.MaxInt32))
			serviceMust(t, z.SetUint32(&upperInt, uint32(math.MaxInt32)+3))
			serviceMust(t, c.config.Binary.SetInteger(&upper, &upperInt))
			var a Algebraic
			serviceMust(t, c.SetRoot(&a, v, &lower, &upper, true))
			serviceMust(t, c.config.Binary.Del(&upper))
			serviceMust(t, c.config.Binary.Del(&lower))
			serviceMust(t, z.Del(&upperInt))
			z.budget.FailAfter(prefix)
			body := c.Neg(&a)
			q := &c.negation
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal(body)
			}
			var extra []*Int
			if q.eval.constructed == 2 {
				extra = append(extra, &q.eval.values[0], &q.eval.values[1])
			}
			if q.eval.addMul.frame != nil {
				extra = append(extra, &q.eval.addMul.frame.tmp)
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{original}, extra, nil, &a)
			// A new scoped setter cannot silently abort this prior call.
			retained := body != nil && q.active && q.eval.addMul.frame != nil && q.eval.addMul.frame.tmp.cell != 0 && !z.isFailureRef(q.eval.addMul.frame.tmp.cell)
			if retained {
				value := NewRational()
				before := q.diagnostic()
				tmp := q.eval.addMul.frame.tmp
				used := z.budget.Used()
				physical := z.memory.Stats()
				if e := c.SetRational(&a, &value); e != ErrAlgebraicPending {
					t.Fatal("later scoped setter consumed pending child", e)
				}
				if before != q.diagnostic() || tmp != q.eval.addMul.frame.tmp || used != z.budget.Used() || physical != z.memory.Stats() {
					t.Fatal("rejected setter changed exact child/graph")
				}
				if e := c.CleanupRetained(); e != ErrAlgebraicLive {
					t.Fatal("own abort crossed live caller-value DT", e)
				}
				if tmp != q.eval.addMul.frame.tmp || q.bodyCause != before.bodyCause || q.sourceCause != before.sourceCause {
					t.Fatal("false acknowledgment changed RAW/first cause")
				}
				algebraicCellSourcePhysical(t, c, []UPolynomial{original}, extra, nil, &a)
			}
			z.budget.FailAfter(-1)
			// Del is the actual caller value DT, not an implicit child acknowledgment.
			serviceMust(t, c.Del(&a))
			if body != nil && q.active && q.eval.addMul.frame != nil && q.eval.addMul.frame.tmp.cell != 0 && !z.isFailureRef(q.eval.addMul.frame.tmp.cell) {
				reached = true
				before := q.diagnostic()
				if e := c.Close(); e != nil {
					t.Fatal("core owning close refused exact child", e)
				}
				if q.active || q.eval.addMul.frame.tmp.cell != 0 || polynomialNegNumericPending(&q.numeric) || polynomialNegNumericPending(&q.eval.numeric) {
					t.Fatal("core refunded history/header while exact child roots remain")
				}
				algebraicCellSourcePhysical(t, c, []UPolynomial{original}, nil, nil)
				if !errors.Is(body, q.bodyCause) {
					t.Fatal("returned body cause changed after owner end")
				}
				if q.bodyCause != before.bodyCause || q.sourceCause != before.sourceCause {
					t.Fatal("core own erased original cause")
				}
			} else {
				serviceMust(t, c.CleanupNegation())
			}
		})
	}
	if !reached {
		t.Fatal("actual retained BIG RAW child frontier absent")
	}
}
