// Source: src/math/polynomial/algebraic_numbers.cpp:383-388,1565-1577,2933-2935.
// Source caller: src/test/algebraic.cpp:376-379 (negation precedes multivariate sign evaluation).
package mpnum

import (
	"fmt"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"testing"
)

func TestAlgebraicNegLiteralNullBasicAndRoot(t *testing.T) {
	c, p := algebraicFixture(t)
	var zero Algebraic
	c.config.Rationals.budget.FailAfter(0)
	serviceMust(t, c.Neg(&zero))
	c.config.Rationals.budget.FailAfter(-1)
	q := NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 3, 2))
	var a Algebraic
	serviceMust(t, c.SetRational(&a, &q))
	serviceMust(t, c.Neg(&a))
	serviceMust(t, c.config.Rationals.Neg(&q))
	cmp, e := c.CompareRational(&a, &q)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("basic negation")
	}
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.config.Rationals.Del(&q))
	poly := algebraicPolynomial(t, p, -2, 0, 0, 1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	var lower, upper BinaryRational
	serviceMust(t, c.config.Binary.SetInt32(&lower, 1))
	serviceMust(t, c.config.Binary.SetInt32(&upper, 2))
	serviceMust(t, c.SetRoot(&a, v, &lower, &upper, true))
	serviceMust(t, c.Neg(&a))
	negative, e := c.IsNeg(&a)
	serviceMust(t, e)
	if !negative {
		t.Fatal("root interval negation")
	}
	l, e := c.location(&a)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	view, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	lead, e := p.read(view, 3)
	serviceMust(t, e)
	if lead.val != -1 {
		t.Fatal("literal p(-x) was renormalized")
	}
	if r.flags&2 != 0 {
		t.Fatal("lower sign not updated from transformed polynomial")
	}
	serviceMust(t, c.Neg(&a))
	positive, e := c.IsPos(&a)
	serviceMust(t, e)
	if !positive {
		t.Fatal("root repeated negation")
	}
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.config.Binary.Del(&upper))
	serviceMust(t, c.config.Binary.Del(&lower))
}

func TestAlgebraicNegMinimumEveryActualReservationPrefix(t *testing.T) {
	success, pending := false, false
	for prefix := int64(0); prefix < 32 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, _ := algebraicFixture(t)
			z := c.config.Rationals.z
			t.Cleanup(func() { z.budget.FailAfter(-1) })
			q := NewRational()
			serviceMust(t, c.config.Rationals.SetInt32(&q, math.MinInt32))
			var a Algebraic
			serviceMust(t, c.SetRational(&a, &q))
			z.budget.FailAfter(prefix)
			body := c.Neg(&a)
			frame := &c.negation
			r := algebraicNegInvocationReceipt{c, frame, frame.invocation}
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal(body)
			}
			if z.pendingCell.object != (memory.ObjectRef{}) {
				pending = true
				if frame.numeric.cell != z.pendingCell {
					t.Fatal("actual MIN leaf not stamped into its executing caller")
				}
			}
			algebraicCellSourcePhysical(t, c, nil, nil, nil, &a)
			if body == nil {
				success = true
				positive, e := c.IsPos(&a)
				serviceMust(t, e)
				if !positive {
					t.Fatal("promoted sign")
				}
			}
			if e := c.cleanupNegation(&r, false); e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(e)
				}
				algebraicCellSourcePhysical(t, c, nil, nil, nil, &a)
			}
			savedBody, savedSource := frame.bodyCause, frame.sourceCause
			z.budget.FailAfter(-1)
			serviceMust(t, c.cleanupNegation(&r, false))
			// Caller-owned cell DT follows completed child acknowledgment and precedes
			// owning retirement of the exact unjoined allocation.
			serviceMust(t, c.Del(&a))
			serviceMust(t, c.cleanupNegation(&r, true))
			if frame.active || frame.bodyCause != savedBody || frame.sourceCause != savedSource {
				t.Fatal("owning acknowledgment changed source history")
			}
			serviceMust(t, c.config.Rationals.Del(&q))
		})
	}
	if !success || !pending {
		t.Fatal("actual MIN success and published pending leaf", success, pending)
	}
}

func TestAlgebraicNegRootHornerEveryActualReservationPrefix(t *testing.T) {
	success, child, retainedTMP := false, false, false
	for prefix := int64(0); prefix < 96 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, p, _ := algebraicCompareFixtureProfile(t, true)
			z := c.config.Rationals.z
			t.Cleanup(func() { z.budget.FailAfter(-1) })
			original := algebraicPolynomial(t, p, 1, math.MinInt32, 1)
			v, e := p.wholeView(original)
			serviceMust(t, e)
			var lower, upper BinaryRational
			serviceMust(t, c.config.Binary.SetInt32(&lower, math.MaxInt32))
			var upperInt Int
			serviceMust(t, z.SetInt64(&upperInt, int64(math.MaxInt32)+3))
			serviceMust(t, c.config.Binary.SetInteger(&upper, &upperInt))
			var a Algebraic
			serviceMust(t, c.SetRoot(&a, v, &lower, &upper, true))
			serviceMust(t, c.config.Binary.Del(&upper))
			serviceMust(t, c.config.Binary.Del(&lower))
			serviceMust(t, z.Del(&upperInt))
			z.budget.FailAfter(prefix)
			body := c.Neg(&a)
			q := &c.negation
			r := algebraicNegInvocationReceipt{c, q, q.invocation}
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal(body)
			}
			if q.eval.constructed == 2 {
				child = true
			}
			if z.pendingCell.object != (memory.ObjectRef{}) {
				if q.numeric.cell != z.pendingCell && q.eval.numeric.cell != z.pendingCell && (!z.matchesAddMulInvocation(&q.eval.addMul) || q.eval.addMul.frame.numeric.cell != z.pendingCell) {
					t.Fatal("actual root/Horner numeric leaf has no concrete caller")
				}
			}
			var extra []*Int
			if q.eval.constructed == 2 {
				extra = append(extra, &q.eval.values[0], &q.eval.values[1])
			}
			if z.matchesAddMulInvocation(&q.eval.addMul) {
				if body != nil && q.eval.addMul.frame.tmp.cell != 0 && !z.isFailureRef(q.eval.addMul.frame.tmp.cell) {
					retainedTMP = true
				}
				extra = append(extra, &q.eval.addMul.frame.tmp)
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{original}, extra, nil, &a)
			if body == nil {
				success = true
				negative, e := c.IsNeg(&a)
				serviceMust(t, e)
				if !negative {
					t.Fatal("negative transformed root")
				}
			}
			// Completed younger source phases are acknowledged before the caller's
			// older root cell and its polynomial/bounds. Unjoined leaves follow them.
			if e := c.cleanupNegation(&r, false); e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(e)
				}
				algebraicCellSourcePhysical(t, c, []UPolynomial{original}, extra, nil, &a)
			}
			z.budget.FailAfter(-1)
			savedBody, savedSource, savedOwn := q.bodyCause, q.sourceCause, q.cleanupCause
			serviceMust(t, c.cleanupNegation(&r, false))
			serviceMust(t, c.Del(&a))
			serviceMust(t, c.cleanupNegation(&r, true))
			if q.active || q.bodyCause != savedBody || q.sourceCause != savedSource || q.cleanupCause != savedOwn {
				t.Fatal("exact owning retirement changed first histories")
			}
		})
	}
	if !success || !retainedTMP || !child {
		t.Fatal("actual root/Horner reached source branches", success, retainedTMP, child)
	}
}
