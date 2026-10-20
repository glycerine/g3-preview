// Source: algebraic_numbers.cpp:1582-1618.
package polynomial

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// The first halving reaches +/-2^-32, an exact root of 2^64*x^2-1.
func algebraicInverseExactHalvingRoot(t *testing.T, c *AlgebraicRootCore, negative bool) Algebraic {
	t.Helper()
	z, p, b := c.config.Rationals.Integers(), c.config.Polynomials, c.config.Binary
	n := unumeral.NewInt(0)
	serviceMust(t, z.SetString(&n, "18446744073709551616"))
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(poly, []unumeral.Int{unumeral.NewInt(-1), unumeral.NewInt(0), n}))
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	var lower, upper unumeral.BinaryRational
	one := unumeral.NewInt(1)
	if negative {
		one = unumeral.NewInt(-1)
		serviceMust(t, b.SetIntegerPower(&lower, &one, 31))
	} else {
		serviceMust(t, b.SetIntegerPower(&upper, &one, 31))
	}
	var a Algebraic
	serviceMust(t, c.SetRoot(&a, v, &lower, &upper, false))
	serviceMust(t, p.DelPolynomial(&poly))
	serviceMust(t, z.Del(&n))
	serviceMust(t, b.Del(&lower))
	serviceMust(t, b.Del(&upper))
	return a
}

func TestAlgebraicInverseRefinementActualHornerAndConversionReservations(t *testing.T) {
	for _, negative := range []bool{false, true} {
		t.Run(fmt.Sprint(negative), func(t *testing.T) {
			succeeded, horner, conversion := false, false, false
			for prefix := int64(0); prefix < 512 && !succeeded; prefix++ {
				if !t.Run(fmt.Sprint(prefix), func(t *testing.T) {
					c, p, _ := algebraicCompareFixtureProfile(t, true)
					a := algebraicInverseExactHalvingRoot(t, c, negative)
					budget := c.config.Rationals.Budget()
					budget.FailAfter(prefix)
					body := c.refineNonzeroBounds(&a)
					budget.FailAfter(-1)
					if body == nil {
						succeeded = true
					} else {
						if !numericPhysicalReservationRefusal(body) {
							t.Fatal("refinement left actual allocator", body)
						}
						if c.inverseRefinementEvaluationPending() {
							horner = true
							if !p.matchesBinaryEvaluationReceipt(&c.inversion.refineEvaluation) || c.localActive || c.inversion.remaining != 0 {
								t.Fatal("Horner failure crossed rational declaration", body)
							}
							l, e := c.locationCleanup(&a)
							serviceMust(t, e)
							r, e := c.load(l)
							serviceMust(t, e)
							changed, other := r.interval.BorrowLower(), r.interval.BorrowUpper()
							numeral := int32(1)
							if negative {
								changed, other = other, changed
								numeral = -1
							}
							if changed.K() != 32 || changed.BorrowNumerator().Val != numeral || other.K() != 31 || other.BorrowNumerator().Val != numeral {
								t.Fatal("halved bound was not saved before Horner failure")
							}
							if e := c.Del(&a); !errors.Is(e, ErrAlgebraicPending) {
								t.Fatal("pending refinement admitted ordinary deletion", e)
							}
						}
						if c.refinementConversion.pending(c) {
							conversion = true
						}
					}
					serviceMust(t, c.CleanupRetained())
					algebraicExpectNoPendingAllocation(t, c.config.Rationals.Integers())
					if c.inverseRefinementEvaluationPending() || p.binaryEvaluation.active || c.refinementConversion.pending(c) || c.localActive {
						t.Fatal("owning refinement cleanup left actual child", body)
					}
					if body == nil {
						want := unumeral.NewRational()
						n := int64(1)
						if negative {
							n = -1
						}
						serviceMust(t, c.config.Rationals.SetFraction(&want, n, 4294967296))
						cmp, e := c.CompareRational(&a, &want)
						serviceMust(t, e)
						if cmp != 0 {
							t.Fatal("exact halving root", cmp)
						}
						serviceMust(t, c.config.Rationals.Del(&want))
					}
					serviceMust(t, c.Del(&a))
				}) {
					return
				}
			}
			if !succeeded || !horner || !conversion {
				t.Fatal("refinement sweep missed actual child", succeeded, horner, conversion)
			}
		})
	}
}

func TestAlgebraicInverseRefinementActualTransportProtectsOwner(t *testing.T) {
	var c *AlgebraicRootCore
	var a Algebraic
	armed, calls := false, 0
	transport := func(f umemory.Failure) error {
		if !armed {
			return f
		}
		calls++
		before := c.inversion
		original := a
		z := c.config.Rationals.Integers()
		used, physical := z.Budget().Used(), z.RuntimeOwner().Stats()
		for _, call := range []func() error{
			func() error { return c.Del(&a) },
			func() error { return c.Inv(&a) },
			func() error { return c.CleanupRetained() },
			func() error { return c.Close() },
		} {
			if e := call(); !errors.Is(e, ErrAlgebraicPending) {
				t.Error("actual nonzero-bound transport acquired owner", e)
			}
		}
		if !reflect.DeepEqual(before, c.inversion) || a != original || used != z.Budget().Used() || physical != z.RuntimeOwner().Stats() {
			t.Error("refused refinement callback changed source owner or allocation graph")
		}
		return f
	}
	var p *UPolynomialManager
	var rt *umemory.Runtime
	c, p, rt = algebraicNegTLSFixture(t, transport)
	a = algebraicInverseExactHalvingRoot(t, c, false)
	held, e := rt.Allocate(uint64(int64(100000) - rt.State().LocalSize - 12))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	body := c.Inv(&a)
	armed = false
	if body == nil || calls == 0 || !c.inverseRefinementEvaluationPending() || c.localActive || c.inversion.remaining != 0 || !p.matchesBinaryEvaluationReceipt(&c.inversion.refineEvaluation) {
		t.Fatal("actual refinement transport not retained before rational declaration", body, calls)
	}
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, rt.Deallocate(&held))
	if c.inverseRefinementEvaluationPending() || p.binaryEvaluation.active {
		t.Fatal("owning refinement acknowledgement retained actual Horner")
	}
	serviceMust(t, c.Del(&a))
}

func TestAlgebraicInverseRefinementPreservesForeignHorner(t *testing.T) {
	c, p := algebraicCellSourceFixtureProfile(t, true)
	a := algebraicInverseExactHalvingRoot(t, c, false)
	l, e := c.location(&a)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	bound := unumeral.NewBinaryIntPower(1, 32)
	held := algebraicCellSourceTLSBoundary(t, c)
	_, body := p.EvalSignBinary(v, &bound)
	if body == nil || !p.binaryEvaluation.active {
		t.Fatal("missing foreign accepted Horner", body)
	}
	older := p.binaryEvaluationReceipt()
	rt := c.config.Rationals.Integers().RuntimeOwner()
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, rt.Deallocate(&held))
	body = c.refineNonzeroBounds(&a)
	if !errors.Is(body, ErrUPolynomialCleanupPending) || c.inversion.refineEvaluation.owner != nil {
		t.Fatal("refused refinement adopted unrelated Horner", body)
	}
	before, used, physical := p.binaryEvaluation, p.config.Integer.Budget().Used(), rt.Stats()
	serviceMust(t, c.CleanupRetained())
	if !reflect.DeepEqual(before, p.binaryEvaluation) || used != p.config.Integer.Budget().Used() || physical != rt.Stats() {
		t.Fatal("refinement cleanup consumed foreign invocation")
	}
	serviceMust(t, p.cleanupBinaryEvaluationWithReceipt(&older, false))
	serviceMust(t, p.cleanupBinaryEvaluationWithReceipt(&older, true))
	serviceMust(t, c.Del(&a))
}

func TestAlgebraicInverseRefinementActualBoundCopyReceipt(t *testing.T) {
	reached := false
	for prefix := int64(0); prefix < 128 && !reached; prefix++ {
		if !t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			a := algebraicLargeDyadicPromotionRoot(t, c)
			l, e := c.location(&a)
			serviceMust(t, e)
			r, e := c.load(l)
			serviceMust(t, e)
			serviceMust(t, c.config.Binary.SetInt32(r.interval.BorrowLower(), 0))
			serviceMust(t, c.store(l, &r))
			z := c.config.Rationals.Integers()
			var fillers [16]unumeral.Int
			filled := 0
			for z.Cells.Stats().Slots > z.Cells.Stats().Live+z.Cells.Stats().Retired || z.Cells.Stats().Slots < z.Cells.Stats().Pages*2 {
				if filled == len(fillers) {
					t.Fatal("ordinary descriptor-fill bound")
				}
				serviceMust(t, z.Copy(&fillers[filled], &z.Two64))
				filled++
			}
			c.config.Rationals.Budget().FailAfter(prefix)
			body := c.refineNonzeroBounds(&a)
			c.config.Rationals.Budget().FailAfter(-1)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("bound copy left actual allocation path", body)
			}
			if c.inverseRefinementCopyPending() {
				reached = true
				if body == nil || c.inversion.refineEvaluation.owner != nil || c.localActive {
					t.Fatal("bound copy allocation crossed Horner declaration", body)
				}
				if e := c.Del(&a); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("pending bound copy admitted ordinary deletion", e)
				}
			}
			serviceMust(t, c.CleanupRetained())
			if c.inverseRefinementCopyPending() {
				t.Fatal("owning refinement cleanup retained bound SET allocation")
			}
			serviceMust(t, c.Del(&a))
			for i := 0; i < filled; i++ {
				serviceMust(t, z.Del(&fillers[i]))
			}
		}) {
			return
		}
	}
	if !reached {
		t.Fatal("no actual bound-copy allocation suffix retained")
	}
}

func TestAlgebraicInverseRefinementAssertionGateBeforeBoundCopy(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertion erased")
	}
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	a := algebraicInverseExactHalvingRoot(t, c, false)
	l, e := c.location(&a)
	serviceMust(t, e)
	before, e := c.load(l)
	serviceMust(t, e)
	budget := arena.NewBudget(1 << 20)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	serviceMust(t, e)
	d, e := diagnostic.NewDebugContext(region, budget)
	serviceMust(t, e)
	previous, e := diagnostic.BindProcessDebugContext(d)
	serviceMust(t, e)
	serviceMust(t, d.Close())
	body := c.refineNonzeroBounds(&a)
	_, e = diagnostic.BindProcessDebugContext(previous)
	serviceMust(t, e)
	serviceMust(t, region.Close())
	after, e := c.load(l)
	serviceMust(t, e)
	if !errors.Is(body, arena.ErrClosed) || !reflect.DeepEqual(before, after) || c.inversion.refineCopy.Frame != nil || c.inversion.refineEvaluation.owner != nil || c.localActive {
		t.Fatal("assertion switch did not precede bound SET and Horner", body)
	}
	serviceMust(t, c.Del(&a))
}
