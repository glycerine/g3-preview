package polynomial

import (
	"errors"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"testing"
)

// (2^64*x-1)*(x-2), with an interval whose midpoint is 1/2^64.
// Retain a nonminimal degree-two source cell until refinement finds that root.
func algebraicDyadicPromotionRoot(t *testing.T, c *AlgebraicRootCore) Algebraic {
	t.Helper()
	p, z := c.config.Polynomials, c.config.Rationals.Integers()
	coefficient := [3]unumeral.Int{unumeral.NewInt(2), unumeral.NewInt(0), unumeral.NewInt(0)}
	serviceMust(t, z.SetString(&coefficient[1], "-36893488147419103233"))
	serviceMust(t, z.SetString(&coefficient[2], "18446744073709551616"))
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(poly, coefficient[:]))
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	lower, upper := unumeral.NewBinaryInt(0), unumeral.NewBinaryIntPower(1, 63)
	var root Algebraic
	serviceMust(t, c.SetRoot(&root, v, &lower, &upper, false))
	serviceMust(t, p.DelPolynomial(&poly))
	for i := range coefficient {
		serviceMust(t, z.Del(&coefficient[i]))
	}
	return root
}

func TestAlgebraicRefinementPromotionReachedReservations(t *testing.T) {
	success, issued := false, false
	for ordinal := int64(0); ordinal < 2048 && !success; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			c, _ := algebraicFixture(t)
			root := algebraicDyadicPromotionRoot(t, c)
			q := c.config.Rationals
			q.Budget().FailAfter(ordinal)
			body := c.Refine(&root)
			q.Budget().FailAfter(-1)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("unexpected promotion refusal", body)
			}
			if c.refinementConversion.pending(c) {
				issued = true
				if e := c.Refine(&root); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("pending promotion converter replaced", e)
				}
			}
			serviceMust(t, c.CleanupRetained())
			if body == nil {
				l, e := c.location(&root)
				serviceMust(t, e)
				if l == nil || l.kind != 0 {
					t.Fatal("exact root was not promoted to a basic cell")
				}
				want := unumeral.NewRational()
				serviceMust(t, q.SetString(&want, "1/18446744073709551616"))
				cmp, e := c.CompareRational(&root, &want)
				serviceMust(t, e)
				if cmp != 0 {
					t.Fatal("independent promoted value", cmp)
				}
				serviceMust(t, q.Del(&want))
			}
			serviceMust(t, c.Del(&root))
			success = body == nil
		})
		if t.Failed() {
			return
		}
	}
	if !success {
		t.Fatal("promotion reservation campaign never completed")
	}
	t.Log("budget campaign retained promotion converter", issued)
}

func TestAlgebraicRefinementPromotionActualPowerRefusal(t *testing.T) {
	issued := false
	for offset := int64(0); offset < 2048 && !issued; offset += 4 {
		t.Run(strconv.FormatInt(offset, 10), func(t *testing.T) {
			var c *AlgebraicRootCore
			var root Algebraic
			callback := false
			transport := func(f umemory.Failure) error {
				if c != nil && c.refinementConversion.executing {
					callback = true
					handle, active, callers := root, c.localActive, c.callers
					for _, call := range []func() error{c.Close, c.CleanupRetained, func() error { return c.Del(&root) }, func() error { return c.Refine(&root) }} {
						if e := call(); !errors.Is(e, ErrAlgebraicPending) {
							t.Fatal("failure callback obtained promotion owner", e)
						}
					}
					if root != handle || c.localActive != active || c.callers != callers || c.closed {
						t.Fatal("refused callback changed promotion lifecycle")
					}
				}
				return f
			}
			c, _, _ = algebraicNegTLSFixture(t, transport)
			root = algebraicDyadicPromotionRoot(t, c)
			// Execute the actual source bisection first. Its transient Horner
			// peak exceeds the converter's peak, so a constant limit armed before
			// refinement would fail there instead of reaching promotion power.
			l, e := c.location(&root)
			serviceMust(t, e)
			r, e := c.load(l)
			serviceMust(t, e)
			v, e := c.sourcePolynomial(&r)
			serviceMust(t, e)
			sign := 1
			if r.flags&2 != 0 {
				sign = -1
			}
			interval, e := c.refineRecord(l, &r, v, sign, 0, false)
			serviceMust(t, e)
			if interval {
				t.Fatal("fixture did not find actual dyadic root")
			}
			runtime := c.config.Rationals.Integers().RuntimeOwner()
			padSize := int64(100000) - runtime.State().LocalSize - offset - 8
			if padSize <= 0 {
				t.Fatal("unexpected TLS extent", runtime.State())
			}
			pad, e := runtime.Allocate(uint64(padSize))
			serviceMust(t, e)
			serviceMust(t, runtime.SetMaxSize(1))
			body := c.promoteRefinedRoot(&root, &r)
			serviceMust(t, runtime.SetMaxSize(0))
			serviceMust(t, runtime.Deallocate(&pad))
			if c.config.Rationals.MatchesRawPowerInvocation(&c.refinementConversion.invocation.Power) {
				issued = true
				if !callback {
					t.Fatal("actual promotion failure transport not observed")
				}
				if body == nil {
					t.Fatal("pending converter without refused source call")
				}
				if c.bounds.conversion.owner != nil {
					t.Fatal("promotion borrowed bounds conversion authority")
				}
				if e := c.Refine(&root); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("replacement consumed promotion child", e)
				}
				l, e := c.location(&root)
				serviceMust(t, e)
				if l == nil || l.kind != 1 {
					t.Fatal("source root deleted before conversion completed")
				}
				r, e := c.load(l)
				serviceMust(t, e)
				if r.interval.BorrowLower().BorrowNumerator().Val != 1 || r.interval.BorrowLower().K() != 64 {
					t.Fatal("exact root publication lost before conversion refusal")
				}
			}
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, c.Del(&root))
		})
		if t.Failed() {
			return
		}
	}
	if !issued {
		t.Fatal("no actual retained promotion conversion reached")
	}
}

func TestAlgebraicRefinementPromotionPreservesOlderPower(t *testing.T) {
	issued := false
	for offset := int64(0); offset < 2048 && !issued; offset += 4 {
		t.Run(strconv.FormatInt(offset, 10), func(t *testing.T) {
			c, _ := algebraicCellSourceFixtureProfile(t, true)
			root := algebraicDyadicPromotionRoot(t, c)
			l, e := c.location(&root)
			serviceMust(t, e)
			r, e := c.load(l)
			serviceMust(t, e)
			v, e := c.sourcePolynomial(&r)
			serviceMust(t, e)
			sign := 1
			if r.flags&2 != 0 {
				sign = -1
			}
			interval, e := c.refineRecord(l, &r, v, sign, 0, false)
			serviceMust(t, e)
			if interval {
				t.Fatal("fixture did not find exact root")
			}
			q, z := c.config.Rationals, c.config.Rationals.Integers()
			runtime := z.RuntimeOwner()
			n := int64(100000) - runtime.State().LocalSize - offset - 8
			if n <= 0 {
				t.Fatal("TLS extent", runtime.State())
			}
			pad, e := runtime.Allocate(uint64(n))
			serviceMust(t, e)
			serviceMust(t, runtime.SetMaxSize(1))
			oldOut := unumeral.NewRational()
			var older unumeral.BinaryToRationalInvocation
			body := c.config.Binary.ToRationalWithInvocation(q, r.interval.BorrowLower(), &oldOut, unumeral.NumericAllocationContext{}, &older)
			serviceMust(t, runtime.SetMaxSize(0))
			serviceMust(t, runtime.Deallocate(&pad))
			if q.MatchesRawPowerInvocation(&older.Power) {
				issued = true
				if body == nil {
					t.Fatal("older power did not refuse")
				}
				body = c.promoteRefinedRoot(&root, &r)
				if !errors.Is(body, unumeral.ErrRationalRawArithmeticPending) || q.MatchesRawPowerInvocation(&c.refinementConversion.invocation.Power) {
					t.Fatal("promotion acquired refused older power", body)
				}
				state, used, stats := q.RawInvocationState(), z.Budget().Used(), runtime.Stats()
				serviceMust(t, c.CleanupRetained())
				if !q.MatchesRawPowerInvocation(&older.Power) || state != q.RawInvocationState() || used != z.Budget().Used() || stats != runtime.Stats() {
					t.Fatal("promotion cleanup consumed older power")
				}
			}
			// This independent caller owns the untracked utility invocation and its
			// allocation debt; core cleanup has no authority to acknowledge either.
			serviceMust(t, z.CleanupPendingAllocation())
			serviceMust(t, q.CleanupRawPowerInvocation(&older.Power))
			serviceMust(t, q.CleanupRawNormalizeInvocation(&older.Normalize))
			serviceMust(t, q.Del(&oldOut))
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, c.Del(&root))
		})
		if t.Failed() {
			return
		}
	}
	if !issued {
		t.Fatal("actual older RAW power frontier was not reached")
	}
}
