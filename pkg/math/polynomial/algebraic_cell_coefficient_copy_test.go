// Source: algebraic_numbers.cpp:405-410,445-455,494.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestAlgebraicCellCoefficientCopyActualTransport(t *testing.T) {
	var c *AlgebraicRootCore
	var src, dst Algebraic
	armed := false
	calls := 0
	transport := func(f umemory.Failure) error {
		if !armed {
			return f
		}
		calls++
		beforeSrc, beforeDst := src, dst
		z := c.config.Rationals.Integers()
		used, stats := z.Budget().Used(), z.RuntimeOwner().Stats()
		if e := c.Del(&src); !errors.Is(e, ErrAlgebraicPending) {
			t.Error("coefficient callback admitted source destruction", e)
			return f
		}
		if e := c.CleanupRetained(); !errors.Is(e, ErrAlgebraicPending) {
			t.Error("setter callback admitted owning cleanup", e)
			return f
		}
		for _, call := range []func() error{c.Close, func() error { return c.Del(&dst) }} {
			if e := call(); !errors.Is(e, ErrAlgebraicPending) {
				t.Error("setter callback admitted core operation", e)
			}
		}
		if e := c.config.Polynomials.Close(); !errors.Is(e, ErrUPolynomialCleanupPending) {
			t.Error("setter callback closed UP owner", e)
		}
		if _, e := c.config.Polynomials.NewPolynomial(); !errors.Is(e, ErrUPolynomialCleanupPending) {
			t.Error("coefficient callback acquired source factory", e)
		}
		if e := z.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
			t.Error("setter callback closed numeric owner", e)
		}
		if src != beforeSrc || dst != beforeDst || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
			t.Error("callback changed owned graph")
		}
		return f
	}
	c, _, rt := algebraicNegTLSFixture(t, transport)
	src = algebraicLargeDyadicPromotionRoot(t, c)
	dst = algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	l, e := c.location(&src)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	pad := algebraicCellSourceTLSBoundary(t, c)
	armed = true
	body := c.SetRoot(&dst, v, r.interval.BorrowLower(), r.interval.BorrowUpper(), true)
	armed = false
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, rt.Deallocate(&pad))
	if calls == 0 || body == nil {
		t.Fatal("transport not reached", calls, body)
	}
	if c.coefficientCopy.Frame == nil {
		t.Fatal("actual coefficient SET receipt missing")
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.CleanupValue(&dst))
	serviceMust(t, c.CleanupValue(&src))
}

func TestAlgebraicCellCoefficientCopyReservationMatrix(t *testing.T) {
	for kind := 0; kind < 3; kind++ {
		t.Run(fmt.Sprint(kind), func(t *testing.T) {
			success := false
			for prefix := int64(0); prefix < 128 && !success; prefix++ {
				if !t.Run(fmt.Sprint(prefix), func(t *testing.T) {
					c, _, _ := algebraicCompareFixtureProfile(t, true)
					src := algebraicLargeDyadicPromotionRoot(t, c)
					dst := algebraicCopyCallValue(t, c, kind)
					l, e := c.location(&src)
					serviceMust(t, e)
					r, e := c.load(l)
					serviceMust(t, e)
					v, e := c.sourcePolynomial(&r)
					serviceMust(t, e)
					z := c.config.Rationals.Integers()
					z.Budget().FailAfter(prefix)
					body := c.SetRoot(&dst, v, r.interval.BorrowLower(), r.interval.BorrowUpper(), true)
					z.Budget().FailAfter(-1)
					if body == nil {
						success = true
						equal, e := c.Equal(&src, &dst)
						serviceMust(t, e)
						if !equal {
							t.Fatal("setter changed value")
						}
					} else if !numericPhysicalReservationRefusal(body) {
						t.Fatal(body)
					}
					serviceMust(t, c.CleanupRetained())
					serviceMust(t, c.CleanupValue(&dst))
					serviceMust(t, c.Del(&src))
					if z.MatchesCopyAllocationInvocation(&c.coefficientCopy) {
						t.Fatal("coefficient child remains")
					}
				}) {
					return
				}
			}
			if !success {
				t.Fatal("reservation matrix never reached success")
			}
		})
	}
}
