// Source: src/math/polynomial/upolynomial.cpp:1906-1951.
package mpnum

import (
	"fmt"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
	"testing"
)

func rootUpperActualGraph(t *testing.T, c *AlgebraicRootCore, p *UPolynomialManager, poly UPolynomial, out *Int, raw []memory.Block) {
	t.Helper()
	q := &p.rootUpperBound
	extra := []*Int{out}
	for i := range q.values {
		extra = append(extra, &q.values[i])
	}
	for _, r := range []*integerRawInvocationReceipt{&q.divisionRaw, &q.extendedRaw} {
		if r.frame != nil {
			for i := uint8(0); i < r.frame.count; i++ {
				extra = append(extra, &r.frame.values[i])
			}
		}
	}
	for _, r := range q.buffers {
		if r.pending != (memory.Block{}) {
			raw = append(raw, r.pending)
		}
	}
	if r := q.verification.frame; r != nil {
		if r.buffer.pending != (memory.Block{}) {
			raw = append(raw, r.buffer.pending)
		}
		if r.buffer.published != (memory.Block{}) {
			raw = append(raw, r.buffer.published)
		}
	}
	algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, extra, raw)
}
func TestUPolynomialRootUpperBoundEveryActualReservationPrefix(t *testing.T) {
	acquired, beforeC, division := false, false, false
	for _, route := range []string{"big-div", "min-abs"} {
		t.Run(route, func(t *testing.T) {
			success := false
			for prefix := int64(0); prefix < 256 && !success; prefix++ {
				t.Run(fmt.Sprint(prefix), func(t *testing.T) {
					c, p := algebraicFixture(t)
					z := c.config.Rationals.z
					poly, e := p.NewPolynomial()
					serviceMust(t, e)
					var max, top Int
					serviceMust(t, z.SetLimbs(&max, []uint32{1, 0, 0, 1}))

					if route == "big-div" {
						serviceMust(t, z.SetLimbs(&top, []uint32{1, 0, 1}))
					} else {
						top = NewInt(math.MinInt32)

					}
					serviceMust(t, p.SetInts(poly, []Int{max, NewInt(2), top}))
					serviceMust(t, z.Del(&max))
					serviceMust(t, z.Del(&top))
					v, e := p.wholeView(poly)
					serviceMust(t, e)
					var out Int
					baseline := z.budget.Used() - integerActualOwnedBytes(z) - p.rootUpperBoundHistoryBytes
					z.budget.FailAfter(prefix)
					body := p.RootUpperBound(v, &out)
					q := &p.rootUpperBound
					r := p.rootUpperBoundReceipt()
					context := q.context()
					if body != nil && !numericPhysicalReservationRefusal(body) {
						t.Fatal(body)
					}
					rootUpperActualGraph(t, c, p, poly, &out, nil)
					if expect := baseline + integerActualOwnedBytes(z) + p.rootUpperBoundHistoryBytes; expect != z.budget.Used() {
						t.Fatal("before-lift exact cold", expect, z.budget.Used())
					}
					if body != nil && q.constructed == 4 {
						beforeC = true
						if q.values[4] != (Int{}) || q.sourceRemaining > 4 {
							t.Fatal("c ctor was credited before leading SET/ABS")
						}
					}
					if polynomialNegNumericPending(&q.numeric) {
						acquired = true
					}
					if q.verification.frame != nil || q.buffers[0].owner != nil {
						division = true
					}
					if q.history != nil {
						copied := *p
						copied.self = &copied
						copied.rootUpperBound.owner = &copied
						receipt := copied.rootUpperBoundReceipt()
						if e := copied.cleanupRootUpperBoundWithReceipt(&receipt, true); e != ErrUPolynomialOwner {
							t.Fatal("copied owning frame", e)
						}
					}
					z.budget.FailAfter(-1)
					if body == nil {
						var expected Int
						digits := []uint32{1, 1}
						if route == "min-abs" {
							digits = []uint32{2, 0, 2}
						}
						serviceMust(t, z.SetLimbs(&expected, digits))
						eq, e := z.Equal(&out, &expected)
						serviceMust(t, e)
						if !eq {
							t.Fatal("independent exact Cauchy bound", out, expected)
						}
						serviceMust(t, z.Del(&expected))
					}
					serviceMust(t, p.cleanupRootUpperBoundWithReceipt(&r, false))
					serviceMust(t, z.Del(&out))
					serviceMust(t, p.cleanupRootUpperBoundWithReceipt(&r, true))
					if r.history != nil {
						h := *r.history
						small := algebraicPolynomial(t, p, -7)
						sv, e := p.wholeView(small)
						serviceMust(t, e)
						z.budget.FailAfter(0)
						e = p.RootUpperBound(sv, &out)
						z.budget.FailAfter(-1)
						if e != nil || out.val != 3 {
							t.Fatal("inline constant bound", out, e)
						}
						if *r.history != h {
							t.Fatal("later inline changed issued history")
						}
						current := p.rootUpperBound
						serviceMust(t, p.cleanupRootUpperBoundWithReceipt(&r, true))
						if !reflect.DeepEqual(current, p.rootUpperBound) {
							t.Fatal("old ACK touched later frame")
						}
						z.rootUpperBoundCaller = p
						if context.roots(z) != nil {
							t.Fatal("stale context retargeted")
						}
						z.rootUpperBoundCaller = nil
						serviceMust(t, p.CleanupRootUpperBound())
						serviceMust(t, z.Del(&out))
					}
					if body == nil {
						success = true
					}

					serviceMust(t, p.DelPolynomial(&poly))
				})
			}
			if !success {
				t.Fatal("no actual successful prefix")
			}
		})
	}
	if !acquired || !beforeC || !division {
		t.Fatal("actual required leaves not reached", acquired, beforeC, division)
	}
}

func TestUPolynomialRootUpperBoundIndependentCauchyCases(t *testing.T) {
	for _, sample := range []struct {
		coefficients []int32
		bound        int32
	}{
		{[]int32{-7}, 3},
		{[]int32{-2, 0, 1}, 4},
		{[]int32{-100, 0, 10}, 12},
		{[]int32{100, 1, 10}, 12},
	} {
		c, p := algebraicFixture(t)
		z := c.config.Rationals.z
		poly := algebraicPolynomial(t, p, sample.coefficients...)
		v, e := p.wholeView(poly)
		serviceMust(t, e)
		var out Int
		serviceMust(t, p.RootUpperBound(v, &out))
		if out != NewInt(sample.bound) {
			t.Fatal(sample.coefficients, out, sample.bound)
		}
		serviceMust(t, p.cleanupRootUpperBound(false))
		serviceMust(t, z.Del(&out))
		serviceMust(t, p.cleanupRootUpperBound(true))
	}
}
