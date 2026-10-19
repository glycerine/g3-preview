// Source: algebraic_numbers.cpp:463-474,509-539. Actual same-core copy callers.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicCopyCallActualTransport(t *testing.T) {
	var c *AlgebraicRootCore
	var src, dst Algebraic
	var borrowed unumeral.Int
	armed, bad, calls := false, false, 0
	transport := func(f umemory.Failure) error {
		if !armed {
			return f
		}
		calls++
		beforeSrc, beforeDst, callers := src, dst, c.callers
		z := c.config.Rationals.Integers()
		used, stats := z.Budget().Used(), z.RuntimeOwner().Stats()
		if e := c.CleanupRetained(); !errors.Is(e, ErrAlgebraicPending) {
			t.Error("actual copy callback admitted owning cleanup", e)
			bad = true
			return f
		}
		for _, call := range []func() error{c.Close, func() error { return c.Del(&src) }, func() error { return c.Del(&dst) }, func() error { return c.Copy(nil, nil) }} {
			if e := call(); !errors.Is(e, ErrAlgebraicPending) {
				t.Error("copy callback acquired core", e)
			}
		}
		p := c.config.Polynomials
		if e := p.Close(); !errors.Is(e, ErrUPolynomialCleanupPending) {
			t.Error("copy callback closed UP owner", e)
		}
		if _, e := p.NewPolynomial(); !errors.Is(e, ErrUPolynomialCleanupPending) {
			t.Error("copy callback acquired source factory", e)
		}
		if e := p.modular.Close(); !errors.Is(e, ErrPolynomialPending) {
			t.Error("copy callback closed modular owner", e)
		}
		if e := z.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
			t.Error("copy callback closed integer owner", e)
		}
		for index, call := range []func() error{z.SourceFinalize, z.Natural.CloseChecked, c.config.Rationals.CloseChecked, c.config.Rationals.SourceFinalize} {
			expected := ErrPolynomialPending
			if index == 1 {
				expected = arena.ErrInvalidSlot
			}
			if e := call(); !errors.Is(e, expected) {
				t.Error("copy callback acquired numeric lifecycle", index, e)
			}
		}
		saved := borrowed
		if e := z.Del(&borrowed); !errors.Is(e, unumeral.ErrNumericPending) || saved != borrowed {
			t.Error("copy callback deleted borrowed source numeral", e)
		}
		if src != beforeSrc || dst != beforeDst || callers != c.callers || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
			t.Error("refused copy callback changed graph")
		}
		return f
	}
	var rt *umemory.Runtime
	c, _, rt = algebraicNegTLSFixture(t, transport)
	src = algebraicBigBinaryBoundsRoot(t, c)
	srcLocation, e := c.location(&src)
	serviceMust(t, e)
	srcRecord, e := c.load(srcLocation)
	serviceMust(t, e)
	borrowed = *srcRecord.interval.BorrowLower().BorrowNumerator()
	dst = algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	pad := algebraicCellSourceTLSBoundary(t, c)
	armed = true
	body := c.Copy(&dst, &src)
	armed = false
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, rt.Deallocate(&pad))
	if calls == 0 || body == nil {
		t.Fatal("actual same-core copy transport not reached", body, calls)
	}
	if bad {
		return
	}
	if c.copyCall.bounds.Frame == nil {
		t.Fatal("copy callback did not reach actual bound SET")
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.CleanupValue(&dst))
	serviceMust(t, c.Del(&src))
}

func algebraicCopyCallValue(t *testing.T, c *AlgebraicRootCore, kind int) Algebraic {
	t.Helper()
	switch kind {
	case 0:
		return Algebraic{}
	case 1:
		q := unumeral.NewRational()
		serviceMust(t, c.config.Rationals.SetString(&q, "1267650600228229401496703205377/18446744073709551619"))
		var a Algebraic
		serviceMust(t, c.SetRational(&a, &q))
		serviceMust(t, c.config.Rationals.Del(&q))
		return a
	case 2:
		return algebraicLargeDyadicPromotionRoot(t, c)
	default:
		return algebraicBigBinaryBoundsRoot(t, c)
	}
}
func TestAlgebraicCopyCallActualReservationMatrix(t *testing.T) {
	for sk := 0; sk < 4; sk++ {
		for dk := 0; dk < 3; dk++ {
			t.Run(fmt.Sprintf("%d-to-%d", sk, dk), func(t *testing.T) {
				success := false
				for prefix := int64(0); prefix < 256 && !success; prefix++ {
					if !t.Run(fmt.Sprint(prefix), func(t *testing.T) {
						c, _, _ := algebraicCompareFixtureProfile(t, true)
						src, dst := algebraicCopyCallValue(t, c, sk), algebraicCopyCallValue(t, c, dk)
						z := c.config.Rationals.Integers()
						z.Budget().FailAfter(prefix)
						body := c.Copy(&dst, &src)
						z.Budget().FailAfter(-1)
						if body == nil {
							success = true
							equal, e := c.Equal(&dst, &src)
							serviceMust(t, e)
							if !equal {
								t.Fatal("copy changed actual value")
							}
						} else if !numericPhysicalReservationRefusal(body) {
							t.Fatal("copy left numerical refusal path", body)
						}
						// The actual caller acknowledges only accepted numerical children. No
						// global allocator acknowledgement or failed scratch repair is used.
						serviceMust(t, c.CleanupRetained())
						serviceMust(t, c.CleanupValue(&dst))
						serviceMust(t, c.Del(&src))
						if c.copyCall.pending() {
							t.Fatal("copy acknowledgement retained a numerical child")
						}
					}) {
						return
					}
				}
				if !success {
					t.Fatal("copy matrix did not reach success")
				}
			})
		}
	}
}

func TestAlgebraicCopyCallActualBoundReceipt(t *testing.T) {
	success, issued := false, false
	for prefix := int64(0); prefix < 1024 && !success; prefix++ {
		if !t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, _, _ := algebraicCompareFixture(t)
			src := algebraicBigBinaryBoundsRoot(t, c)
			values, e := NewSimpleAlgebraicValues(c)
			serviceMust(t, e)
			z := c.config.Rationals.Integers()
			z.Budget().FailAfter(prefix)
			var body error
			for i := uint32(0); i < 32; i++ {
				body = values.Push(i, &src)
				if body != nil {
					break
				}
			}
			z.Budget().FailAfter(-1)
			if body == nil {
				success = true
			} else if !numericPhysicalReservationRefusal(body) {
				t.Fatal("mapping copy left numerical path", body)
			}
			if z.MatchesCopyAllocationInvocation(&c.copyCall.bounds) {
				issued = true
				var untouched Algebraic
				if e := c.Copy(&untouched, &src); !errors.Is(e, ErrAlgebraicPending) || untouched != (Algebraic{}) {
					t.Fatal("pending bound SET was replaced", e)
				}
			}
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, values.CleanupRetained())
			serviceMust(t, values.Close())
			serviceMust(t, c.Del(&src))
		}) {
			return
		}
	}
	if !success || !issued {
		t.Fatal("bound copy campaign missed actual receipt or success", success, issued)
	}
}

func TestAlgebraicCopyCallPreservesOlderNumericAllocation(t *testing.T) {
	c, _ := algebraicCellSourceFixtureProfile(t, true)
	src := algebraicLargeDyadicPromotionRoot(t, c)
	z := c.config.Rationals.Integers()
	rt := z.RuntimeOwner()
	pad := algebraicCellSourceTLSBoundary(t, c)
	var old unumeral.IntegerCopyAllocationReceipt
	var oldDestination unumeral.Int
	body := z.CopyFromOwnerWithReceipt(&oldDestination, z, &z.Two64, &old)
	if body == nil || !z.MatchesCopyAllocationInvocation(&old) {
		t.Fatal("older numeric allocation refusal missing", body)
	}
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, rt.Deallocate(&pad))
	var dst Algebraic
	body = c.Copy(&dst, &src)
	if !errors.Is(body, ErrPolynomialPending) || c.copyCall.pending() {
		t.Fatal("refused copy acquired older numeric allocation", body)
	}
	state, used, stats := z.CopyAllocation, z.Budget().Used(), rt.Stats()
	serviceMust(t, c.CleanupRetained())
	if !z.MatchesCopyAllocationInvocation(&old) || state != z.CopyAllocation || used != z.Budget().Used() || stats != rt.Stats() {
		t.Fatal("copy cleanup consumed older numeric owner")
	}
	serviceMust(t, z.CleanupCopyAllocationInvocation(&old))
	serviceMust(t, z.Del(&oldDestination))
	serviceMust(t, c.CleanupValue(&dst))
	serviceMust(t, c.Del(&src))
}
