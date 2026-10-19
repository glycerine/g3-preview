// Source: algebraic_numbers.cpp:392-398; actual neg(INT_MIN) allocation.
package polynomial

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// A real existing cell is replaced by the negative-leading source polynomial.
// This component fixture stops immediately before normalize_coeffs; it does
// not claim a completed constructor with unnormalized coefficients.
func algebraicCellNegativeNormalizationInput(t *testing.T, c *AlgebraicRootCore, p *UPolynomialManager) (Algebraic, *algebraicLocation, algebraicRecord) {
	t.Helper()
	a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	poly := algebraicPolynomial(t, p, 1, math.MinInt32, -1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	l, e := c.location(&a)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	serviceMust(t, c.deletePolynomial(l, &r))
	serviceMust(t, c.copyPolynomial(l, &r, v))
	lo, hi := unumeral.NewBinaryInt(0), unumeral.NewBinaryInt(1)
	serviceMust(t, c.config.Intervals.SetBounds(&r.interval, &lo, &hi))
	serviceMust(t, c.store(l, &r))
	serviceMust(t, c.updateSign(l, &r))
	return a, l, r
}
func TestAlgebraicCellNegativeNormalizationActualTransport(t *testing.T) {
	var c *AlgebraicRootCore
	var p *UPolynomialManager
	var a Algebraic
	var operand *unumeral.Int
	armed, calls := false, 0
	transport := func(f umemory.Failure) error {
		if !armed {
			return f
		}
		calls++
		before, callers := a, c.callers
		z := c.config.Rationals.Integers()
		used, physical := z.Budget().Used(), z.RuntimeOwner().Stats()
		if e := c.CleanupRetained(); !errors.Is(e, ErrAlgebraicPending) {
			t.Error("coefficient negation callback admitted owning cleanup", e)
			return f
		}
		for _, call := range []func() error{c.Close, func() error { return c.Del(&a) }, func() error { return c.SetRoot(nil, UPolynomialView{}, nil, nil, false) }} {
			if e := call(); !errors.Is(e, ErrAlgebraicPending) {
				t.Error("coefficient negation callback acquired core", e)
			}
		}
		if e := p.Close(); !errors.Is(e, ErrUPolynomialCleanupPending) {
			t.Error("coefficient negation callback closed UP manager", e)
		}
		if e := p.modular.Close(); !errors.Is(e, ErrPolynomialPending) {
			t.Error("coefficient negation callback closed modular owner", e)
		}
		if e := z.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
			t.Error("coefficient negation callback closed integer owner", e)
		}
		if e := z.Del(operand); !errors.Is(e, unumeral.ErrNumericPending) {
			t.Error("coefficient negation callback deleted borrowed coefficient", e)
		}
		if a != before || c.callers != callers || used != z.Budget().Used() || physical != z.RuntimeOwner().Stats() {
			t.Error("refused coefficient callback changed graph")
		}
		return f
	}
	var rt *umemory.Runtime
	c, p, rt = algebraicNegTLSFixture(t, transport)
	var r algebraicRecord
	a, _, r = algebraicCellNegativeNormalizationInput(t, c, p)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	operand, e = p.read(v, 1)
	serviceMust(t, e)
	// Warm only the source content normalization; the sign-changing negation
	// remains unreached and its INT_MIN coefficient is still inline.
	serviceMust(t, p.normalizeViewWithReceipt(v, &c.coefficientNormalization))
	pad := algebraicCellSourceTLSBoundary(t, c)
	armed = true
	// The actual negation subcall follows the already completed content normalization.
	body := c.negateCellCoefficients(v)
	armed = false
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, rt.Deallocate(&pad))
	if body == nil || calls == 0 || c.cellNeg.cause != body {
		t.Fatal("actual coefficient negation allocation not retained", body, calls)
	}
	if c.cellNeg.pending() {
		if e := c.Del(&a); !errors.Is(e, ErrAlgebraicPending) {
			t.Fatal("pending coefficient allocation admitted caller deletion", e)
		}
	}
	serviceMust(t, c.CleanupRetained())
	if c.cellNeg.pending() {
		t.Fatal("owning cleanup retained actual negation allocation")
	}
	serviceMust(t, c.Del(&a))
}

func TestAlgebraicCellNegativeNormalizationConstructorReservations(t *testing.T) {
	succeeded, reached := false, false
	for prefix := int64(0); prefix < 1024 && !succeeded; prefix++ {
		if !t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, p, _ := algebraicCompareFixtureProfile(t, true)
			poly := algebraicPolynomial(t, p, 1, math.MinInt32, -1)
			v, e := p.wholeView(poly)
			serviceMust(t, e)
			lo, hi := unumeral.NewBinaryInt(0), unumeral.NewBinaryInt(1)
			z := c.config.Rationals.Integers()
			var out Algebraic
			z.Budget().FailAfter(prefix)
			body := c.SetRoot(&out, v, &lo, &hi, false)
			z.Budget().FailAfter(-1)
			if body == nil {
				succeeded = true
				l, e := c.location(&out)
				serviceMust(t, e)
				r, e := c.load(l)
				serviceMust(t, e)
				cv, e := c.sourcePolynomial(&r)
				serviceMust(t, e)
				for i, want := range []string{"-1", "2147483648", "1"} {
					n, e := p.read(cv, uint32(i))
					serviceMust(t, e)
					text, e := z.String(n)
					serviceMust(t, e)
					if text != want {
						t.Fatal("normalized coefficient", i, text, want)
					}
				}
				if r.flags&2 == 0 {
					t.Fatal("negative leading normalization did not flip lower sign")
				}
			} else if !numericPhysicalReservationRefusal(body) {
				t.Fatal("constructor left numerical refusal path", body)
			}
			if c.cellNeg.cause != nil {
				reached = true
				if out.slot != 0 || c.pending.slot == 0 {
					t.Fatal("normalization refusal crossed constructor publication")
				}
			} else if !c.cellEvaluationPending() && !c.coefficientNormalizationPending() && !c.cellNeg.pending() {
				serviceMust(t, z.CleanupPendingAllocation())
			}
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, c.Del(&out))
		}) {
			return
		}
	}
	if !succeeded || !reached {
		t.Fatal("negative normalization sweep missed refusal or success", succeeded, reached)
	}
}

func TestAlgebraicCellNegativeNormalizationPreservesOlderAllocation(t *testing.T) {
	c, p := algebraicCellSourceFixtureProfile(t, true)
	a, _, r := algebraicCellNegativeNormalizationInput(t, c, p)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	z := c.config.Rationals.Integers()
	rt := z.RuntimeOwner()
	pad := algebraicCellSourceTLSBoundary(t, c)
	var older unumeral.IntegerCopyAllocationReceipt
	var destination unumeral.Int
	body := z.CopyFromOwnerWithReceipt(&destination, z, &z.Two64, &older)
	if body == nil || !z.MatchesCopyAllocationInvocation(&older) {
		t.Fatal("older actual allocation refusal missing", body)
	}
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, rt.Deallocate(&pad))
	body = c.negateCellCoefficients(v)
	if body == nil || c.cellNeg.pending() {
		t.Fatal("refused negation acquired older allocation", body)
	}
	state, used, stats := z.CopyAllocation, z.Budget().Used(), rt.Stats()
	serviceMust(t, c.CleanupRetained())
	if !z.MatchesCopyAllocationInvocation(&older) || state != z.CopyAllocation || used != z.Budget().Used() || stats != rt.Stats() {
		t.Fatal("cell cleanup consumed unrelated numeric allocation")
	}
	serviceMust(t, z.CleanupCopyAllocationInvocation(&older))
	serviceMust(t, z.Del(&destination))
	serviceMust(t, c.Del(&a))
}

func TestAlgebraicCellNegativeNormalizationContextIdentity(t *testing.T) {
	c, p, _ := algebraicCompareFixtureProfile(t, true)
	a, _, r := algebraicCellNegativeNormalizationInput(t, c, p)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	serviceMust(t, c.negateCellCoefficients(v))
	first := c.cellNeg.invocation
	old := unumeral.NumericAllocationContext{}.WithCaller(&c.cellNeg.hooks)
	serviceMust(t, c.Del(&a))
	a, _, r = algebraicCellNegativeNormalizationInput(t, c, p)
	v, e = c.sourcePolynomial(&r)
	serviceMust(t, e)
	serviceMust(t, c.negateCellCoefficients(v))
	if c.cellNeg.invocation != first+1 {
		t.Fatal("actual replacement call did not advance numeric context")
	}
	z := c.config.Rationals.Integers()
	used, stats := z.Budget().Used(), z.RuntimeOwner().Stats()
	if e := z.WithAllocationContext(old).Neg(nil); !errors.Is(e, unumeral.ErrNumericOwner) {
		t.Fatal("old numeric context accepted an operand", e)
	}
	copied := c.cellNeg
	if copied.NumericBound(false) {
		t.Fatal("copied numeric scope acquired actual owner")
	}
	if used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
		t.Fatal("refused old context changed allocation graph")
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.Del(&a))
}

func TestAlgebraicCellNegativeNormalizationNumeralTraceBorrow(t *testing.T) {
	c, p, _ := algebraicCompareFixtureProfile(t, true)
	a, _, r := algebraicCellNegativeNormalizationInput(t, c, p)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	operand, e := p.read(v, 1)
	serviceMust(t, e)
	z := c.config.Rationals.Integers()
	calls := 0
	writer := &linearOperationProbeWriter{probe: func() {
		calls++
		before, callers := a, c.callers
		used, stats := z.Budget().Used(), z.RuntimeOwner().Stats()
		for _, call := range []func() error{c.CleanupRetained, c.Close, func() error { return c.Del(&a) }} {
			if e := call(); !errors.Is(e, ErrAlgebraicPending) {
				t.Error("numeral trace acquired coefficient owner", e)
			}
		}
		if e := p.Close(); !errors.Is(e, ErrUPolynomialCleanupPending) {
			t.Error("numeral trace closed UP owner", e)
		}
		if e := z.Del(operand); !errors.Is(e, unumeral.ErrNumericPending) {
			t.Error("numeral trace deleted actual coefficient", e)
		}
		if a != before || c.callers != callers || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
			t.Error("refused numeral trace changed graph")
		}
	}}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("mpz")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	serviceMust(t, c.negateCellCoefficients(v))
	if tags.ShouldTrace("mpz") && calls == 0 {
		t.Fatal("coefficient negation did not reach numeral trace")
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.Del(&a))
}
