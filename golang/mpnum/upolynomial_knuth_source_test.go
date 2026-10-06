// Source: src/math/polynomial/upolynomial.cpp:1594-1602,2029-2074.
package mpnum

import (
	"errors"
	"fmt"
	"math"
	"testing"
)

func TestUPolynomialKnuthMinimumEveryActualPrefixAndDestructiveSuffix(t *testing.T) {
	acquired, reversed := false, false
	for _, route := range []string{"negative", "nonzero"} {
		t.Run(route, func(t *testing.T) {
			success := false
			for prefix := int64(0); prefix < 64 && !success; prefix++ {
				t.Run(fmt.Sprint(prefix), func(t *testing.T) {
					c, p := algebraicFixture(t)
					z := c.config.Rationals.z
					values := []int32{1, 7, 2, math.MinInt32, 1}
					if route == "nonzero" {
						values = []int32{0, 1, 3, math.MinInt32, 2}
					}
					poly, e := p.NewPolynomial()
					serviceMust(t, e)
					nums := make([]Int, len(values))
					for i, v := range values {
						nums[i] = NewInt(v)
					}
					serviceMust(t, p.SetInts(poly, nums))
					v, e := p.wholeView(poly)
					serviceMust(t, e)
					baseline := z.budget.Used() - integerActualOwnedBytes(z) - p.knuthHistoryBytes
					z.budget.FailAfter(prefix)
					var bound uint32
					var body error
					if route == "negative" {
						bound, body = p.KnuthNegativeRootUpperBound(v)
					} else {
						bound, body = p.NonzeroRootLowerBound(v)
					}
					q := &p.knuth
					r := p.knuthReceipt()
					context := q.context()
					if body != nil && !numericPhysicalReservationRefusal(body) {
						t.Fatal(body)
					}
					algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, nil, nil)
					if expected := baseline + integerActualOwnedBytes(z) + p.knuthHistoryBytes; z.budget.Used() != expected {
						t.Fatal("exact before-lift cold", expected, z.budget.Used())
					}
					if polynomialNegNumericPending(&q.numeric) {
						acquired = true
					}
					if body != nil {
						coefficient, e := p.read(v, 1)
						serviceMust(t, e)
						if route == "negative" {
							if coefficient.val != -7 {
								t.Fatal("source partial p(-x) restored earlier odd coefficient", *coefficient)
							}
						} else {
							coefficient, e = p.read(v, 4)
							serviceMust(t, e)
							if *coefficient != NewInt(1) {
								t.Fatal("source reversed suffix was restored on throw", *coefficient)
							}
							reversed = true
						}
					} else {
						expected := uint32(2)
						if route == "nonzero" {
							expected = 17
						}
						if bound != expected {
							t.Fatal("independent source bound", route, bound, expected)
						}
					}
					z.budget.FailAfter(-1)
					serviceMust(t, p.cleanupKnuthWithReceipt(&r, false))
					serviceMust(t, p.DelPolynomial(&poly))
					serviceMust(t, p.cleanupKnuthWithReceipt(&r, true))
					if r.history != nil {
						h := *r.history
						empty, e := p.NewPolynomial()
						serviceMust(t, e)
						ev, e := p.wholeView(empty)
						serviceMust(t, e)
						z.budget.FailAfter(0)
						b, e := p.KnuthNegativeRootUpperBound(ev)
						z.budget.FailAfter(-1)
						if e != nil || b != 0 {
							t.Fatal("erased repeated zero negative route allocated", b, e)
						}
						if *r.history != h {
							t.Fatal("old charged history overwritten")
						}
						serviceMust(t, p.cleanupKnuthWithReceipt(&r, true))
						if *r.history != h {
							t.Fatal("stale own changed old history")
						}
						if context.roots(z) != nil {
							t.Fatal("old allocation context retargeted later frame")
						}
						serviceMust(t, p.DelPolynomial(&empty))
					}
					success = body == nil
				})
			}
			if !success {
				t.Fatal("no actual complete prefix")
			}
		})
	}
	if !acquired || !reversed {
		t.Fatal("real MIN authority/reverse failure frontiers missing", acquired, reversed)
	}
}
func TestUPolynomialKnuthIndependentBoundsAndSourceAssertions(t *testing.T) {
	c, p := algebraicFixture(t)
	z := c.config.Rationals.z
	for _, sample := range []struct {
		coeff             []int32
		negative, nonzero uint32
	}{
		{[]int32{-2, 0, 1}, 2, 1}, {[]int32{0, 0, -2, 0, 1}, 2, 1}, {[]int32{1, -3, 2}, 1, 3},
	} {
		poly := algebraicPolynomial(t, p, sample.coeff...)
		v, e := p.wholeView(poly)
		serviceMust(t, e)
		b, e := p.KnuthNegativeRootUpperBound(v)
		serviceMust(t, e)
		if b != sample.negative {
			t.Fatal("negative", sample.coeff, b, sample.negative)
		}
		b, e = p.NonzeroRootLowerBound(v)
		serviceMust(t, e)
		if b != sample.nonzero {
			t.Fatal("nonzero", sample.coeff, b, sample.nonzero)
		}
		serviceMust(t, p.CleanupKnuthBounds())
	}
	if modularSourceDebug {
		empty := algebraicPolynomial(t, p)
		v, e := p.wholeView(empty)
		serviceMust(t, e)
		z.budget.FailAfter(0)
		_, e = p.NonzeroRootLowerBound(v)
		z.budget.FailAfter(-1)
		if !errors.Is(e, ErrUPolynomialAssertion) {
			t.Fatal("source nonzero size assertion", e)
		}
	}
}
