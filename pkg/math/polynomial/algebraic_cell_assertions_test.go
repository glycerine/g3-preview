package polynomial

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestAlgebraicCellAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			polynomialAssertionActionsFixture(t, action)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			body := c.assertCellLowerSign(0)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(body, ErrUPolynomialAssertion) || !errors.Is(body, ErrAlgebraicAssertion) || !linearThrownAssertion(body) {
					t.Fatal("cell assertion lost source Throw", body)
				}
			} else {
				serviceMust(t, body)
			}
		})
	}
}
func TestAlgebraicCellAssertionDisabledErasesOperands(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	for _, call := range []func() error{func() error { return c.assertCellUpperSign(UPolynomialView{}, nil, 1) }, func() error { return c.assertCellNormalizeDegree(nil) }, func() error { return c.assertNewCellFlags(nil) }, func() error { return c.assertAssignedCellSign(nil) }} {
		serviceMust(t, call())
	}
}
func TestAlgebraicCellConstructorActualSignActions(t *testing.T) {
	for _, lowerZero := range []bool{true, false} {
		for _, action := range []string{"T", "CCC", "disabled"} {
			t.Run(fmt.Sprint(lowerZero, "/", action), func(t *testing.T) {
				c, p, _ := algebraicCompareFixtureProfile(t, true)
				poly := algebraicPolynomial(t, p, -1, 0, 1)
				v, e := p.wholeView(poly)
				serviceMust(t, e)
				lower, upper := unumeral.NewBinaryInt(1), unumeral.NewBinaryInt(2)
				if !lowerZero {
					lower = unumeral.NewBinaryInt(2)
					upper = unumeral.NewBinaryInt(3)
				}
				polynomialAssertionActionsFixture(t, action)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				var out Algebraic
				body := c.SetRoot(&out, v, &lower, &upper, false)
				if polynomialSourceDebug && action == "T" {
					if !linearThrownAssertion(body) || out.slot != 0 || c.pending.slot == 0 {
						t.Fatal("cell source assertion crossed constructor publication", body, out, c.pending)
					}
				} else {
					serviceMust(t, body)
				}
				serviceMust(t, c.CleanupRetained())
				serviceMust(t, c.Del(&out))
			})
		}
	}
}

func TestAlgebraicCellHornerActualReservationAndPublication(t *testing.T) {
	succeeded := false
	var reached [3]bool
	for prefix := int64(0); prefix < 1024 && !succeeded; prefix++ {
		if !t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, pm, _ := algebraicCompareFixtureProfile(t, true)
			seed := algebraicTinyBinaryBoundsRoot(t, c)
			loc, e := c.location(&seed)
			serviceMust(t, e)
			record, e := c.load(loc)
			serviceMust(t, e)
			v, e := c.sourcePolynomial(&record)
			serviceMust(t, e)
			z := c.config.Rationals.Integers()
			var out Algebraic
			z.Budget().FailAfter(prefix)
			body := c.SetRoot(&out, v, record.interval.BorrowLower(), record.interval.BorrowUpper(), true)
			z.Budget().FailAfter(-1)
			if body == nil {
				succeeded = true
			} else if !numericPhysicalReservationRefusal(body) {
				t.Fatal("constructor left numerical allocation path", body)
			}
			pending := c.cellEvaluationPending()
			if pending {
				for i := range c.cellEvaluation {
					if pm.matchesBinaryEvaluationReceipt(&c.cellEvaluation[i]) && pm.binaryEvaluation.active {
						reached[i] = true
						if i < 2 && (out.slot != 0 || c.pending.slot == 0) {
							t.Fatal("constructor Horner crossed publication", i, out, c.pending)
						}
						if i == 2 && (out.slot == 0 || c.pending.slot != 0) {
							t.Fatal("post-assignment Horner preceded caller publication", out, c.pending)
						}
					}
				}
				if e := c.Del(&seed); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("pending cell evaluation admitted caller deletion", e)
				}
			} else {
				serviceMust(t, z.CleanupPendingAllocation())
			}
			serviceMust(t, c.CleanupRetained())
			if c.cellEvaluationPending() {
				t.Fatal("owning cell cleanup retained actual Horner")
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&seed))
		}) {
			return
		}
	}
	if !succeeded || !reached[0] || polynomialSourceDebug && (!reached[1] || !reached[2]) {
		t.Fatal("cell campaign missed actual reached numerical children", succeeded, reached)
	}
}

func TestAlgebraicCellClosedAssertionSwitchBeforeOperands(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	polynomialAssertionActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	for _, call := range []func() error{func() error { return c.assertCellUpperSign(UPolynomialView{}, nil, 1) }, func() error { return c.assertCellNormalizeDegree(nil) }, func() error { return c.assertNewCellFlags(nil) }, func() error { return c.assertAssignedCellSign(nil) }} {
		if e := call(); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("cell assertion read operand before its source switch", e)
		}
	}
	if e := c.SetRoot(nil, UPolynomialView{count: 3}, nil, nil, false); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("setter read caller before entry assertion switch", e)
	}
}

func TestAlgebraicCellActualHornerTransportProtectsOwner(t *testing.T) {
	var c *AlgebraicRootCore
	var seed Algebraic
	armed, calls := false, 0
	transport := func(f umemory.Failure) error {
		if !armed {
			return f
		}
		calls++
		if !c.cellEvaluationRunning {
			t.Error("transport left actual cell Horner call")
		}
		before, callers := seed, c.callers
		z := c.config.Rationals.Integers()
		used, physical := z.Budget().Used(), z.RuntimeOwner().Stats()
		for _, call := range []func() error{c.Close, c.CleanupRetained, func() error { return c.Del(&seed) }, func() error { return c.SetRoot(nil, UPolynomialView{}, nil, nil, false) }} {
			if e := call(); !errors.Is(e, ErrAlgebraicPending) {
				t.Error("cell Horner transport acquired algebraic owner", e)
			}
		}
		if seed != before || c.callers != callers || used != z.Budget().Used() || physical != z.RuntimeOwner().Stats() {
			t.Error("refused cell callback changed owner or physical graph")
		}
		return f
	}
	var rt *umemory.Runtime
	c, _, rt = algebraicNegTLSFixture(t, transport)
	seed = algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	l, e := c.location(&seed)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	// A valid binary Horner operand forces a direct large numeral allocation.
	// This component vector claims no constructor completion at these bounds.
	b := unumeral.NewBinaryIntPower(1, 32768)
	held, e := rt.Allocate(uint64(int64(100000) - rt.State().LocalSize - 12))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	_, body := c.evalCellSign(v, &b, 0)
	armed = false
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, rt.Deallocate(&held))
	if body == nil || calls == 0 || !c.cellEvaluationPending() {
		t.Fatal("actual cell Horner transport not retained", body, calls)
	}
	serviceMust(t, c.CleanupRetained())
	if c.cellEvaluationPending() {
		t.Fatal("owning cell cleanup retained Horner child")
	}
	serviceMust(t, c.Del(&seed))
}
