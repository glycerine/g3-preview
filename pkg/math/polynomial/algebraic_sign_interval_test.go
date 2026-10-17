package polynomial

import (
	"errors"
	"strconv"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestAlgebraicIntervalSignRefinementIndependentBounds(t *testing.T) {
	for _, negative := range []bool{false, true} {
		c, p, control := algebraicControlledFixture(t)
		coefficient, constant := int32(1), int32(-3)
		if negative {
			coefficient, constant = -1, 3
		}
		pm, poly, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{coefficient, [3]uint32{1, 0, 0}}, {coefficient, [3]uint32{0, 1, 0}}, {constant, [3]uint32{}}})
		a, b := algebraicResultantInputs(t, c, p)
		values := algebraicSignTestValues{}
		values.values[0], values.values[1] = a, b
		sign, determined, restart, e := c.intervalSignPrefix(pm, poly, &values)
		serviceMust(t, e)
		want := 1
		if negative {
			want = -1
		}
		// sqrt(2)+sqrt(3)>3 since its square is 5+2*sqrt(6)>9.
		if !determined || restart || sign != want || control.visits < 2 {
			t.Fatal("refined sign", sign, determined, restart, control.visits)
		}
		if c.signInterval.active {
			t.Fatal("determined interval stayed live")
		}
		serviceMust(t, c.Del(&b))
		serviceMust(t, c.Del(&a))
	}
}

func TestAlgebraicIntervalSignUnresolvedAndRationalRestart(t *testing.T) {
	c, p, _ := algebraicControlledFixture(t)
	pm, poly, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {-2, [3]uint32{}}})
	a, b := algebraicResultantInputs(t, c, p)
	values := algebraicSignTestValues{}
	values.values[0] = a
	_, determined, restart, e := c.intervalSignPrefix(pm, poly, &values)
	serviceMust(t, e)
	if determined || restart || !c.signInterval.active || !c.signInterval.constructed {
		t.Fatal("exact-zero candidate did not preserve resultant input")
	}
	contains, e := c.config.Intervals.ContainsZero(&c.signInterval.interval)
	serviceMust(t, e)
	if !contains {
		t.Fatal("zero was excluded")
	}
	serviceMust(t, c.CleanupRetained())
	// Preserve the source positive zero-accuracy branch separately from the
	// exact path: it accepts the remaining zero-containing interval as zero.
	c.zeroAccuracy = 1
	zero, determined, restart, e := c.intervalSignPrefix(pm, poly, &values)
	serviceMust(t, e)
	if zero != 0 || !determined || restart || c.signInterval.active {
		t.Fatal("zero-accuracy branch", zero, determined, restart)
	}
	c.zeroAccuracy = 0
	serviceMust(t, c.Del(&a))
	a = algebraicCompareRoot(t, c, []int32{-1, 0, 1}, 0, 0, 2, 0, false)
	values.values[0] = a
	// x^2-2 still crosses zero over (0,2), and midpoint refinement finds x=1.
	_, determined, restart, e = c.intervalSignPrefix(pm, poly, &values)
	serviceMust(t, e)
	if determined || !restart || c.signInterval.active {
		t.Fatal("exact-rational restart", determined, restart)
	}
	sign, all, e := c.tryRationalSign(pm, poly, &values)
	serviceMust(t, e)
	if !all || sign != -1 {
		t.Fatal("restart did not reach basic evaluation", sign, all)
	}
	serviceMust(t, c.Del(&b))
	serviceMust(t, c.Del(&values.values[0]))
}

func TestAlgebraicIntervalSignPreservesOlderPendingEvaluation(t *testing.T) {
	c, p, _ := algebraicControlledFixture(t)
	pm, poly, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
	a, b := algebraicResultantInputs(t, c, p)
	values := algebraicSignTestValues{}
	values.values[0] = a
	wrapped := polynomialLexWrappedPolynomial(t, pm)
	olderMap := polynomialEvalRationalMap{manager: c.config.Rationals}
	for i := range olderMap.values {
		olderMap.values[i] = unumeral.NewRational()
	}
	olderOutput := unumeral.NewRational()
	body := pm.EvaluateRational(wrapped, &olderMap, &olderOutput)
	if body == nil || !pm.evaluation.active || !pm.lexPending {
		t.Fatal("older bounds stop not reached", body)
	}
	lex := pm.evaluation.lexGeneration
	_, _, _, body = c.intervalSignPrefix(pm, poly, &values)
	if !errors.Is(body, ErrPolynomialPending) {
		t.Fatal("older scope refusal", body)
	}
	serviceMust(t, c.CleanupRetained())
	if !pm.evaluation.active || pm.evaluation.lexGeneration != lex {
		t.Fatal("older invocation consumed by interval parent")
	}
	serviceMust(t, pm.CleanupEvaluation())
	serviceMust(t, c.config.Rationals.Del(&olderOutput))
	for i := range olderMap.values {
		serviceMust(t, c.config.Rationals.Del(&olderMap.values[i]))
	}
	serviceMust(t, c.Del(&b))
	serviceMust(t, c.Del(&a))
}

func TestAlgebraicIntervalSignEveryCheckpointCancellationRetry(t *testing.T) {
	for _, outer := range []bool{true, false} {
		success := false
		for allowance := uint32(0); allowance < 128 && !success; allowance++ {
			t.Run(strconv.FormatBool(outer)+"/"+strconv.FormatUint(uint64(allowance), 10), func(t *testing.T) {
				c, p, control := algebraicControlledFixture(t)
				pm, poly, extControl := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{0, 1, 0}}, {-3, [3]uint32{}}})
				a, b := algebraicResultantInputs(t, c, p)
				values := algebraicSignTestValues{}
				values.values[0], values.values[1] = a, b
				if outer {
					control.limit = control.visits + uint64(allowance)
				} else {
					extControl.limit = extControl.checkpoints + allowance
				}
				sign, determined, restart, body := c.intervalSignPrefix(pm, poly, &values)
				control.limit = 1000000
				extControl.limit = 1000000
				if body == nil {
					success = true
				} else {
					if !errors.Is(body, ErrUPolynomialCanceled) {
						t.Fatal(body)
					}
					serviceMust(t, c.CleanupRetained())
					sign, determined, restart, body = c.intervalSignPrefix(pm, poly, &values)
					serviceMust(t, body)
				}
				if !determined || restart || sign != 1 {
					t.Fatal(sign, determined, restart)
				}
				serviceMust(t, c.CleanupRetained())
				serviceMust(t, c.Del(&b))
				serviceMust(t, c.Del(&a))
			})
		}
		if !success {
			t.Fatal("no successful cancellation endpoint")
		}
	}
}

func TestAlgebraicIntervalSignGetterAndEmptyVariablesAssertions(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	c, _, _ := algebraicControlledFixture(t)
	pm, _, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{}}})
	values := algebraicSignTestValues{}
	polynomialAssertionActionsFixture(t, "TT")
	mapping := algebraicRootIntervals{owner: c, values: &values}
	_, e := mapping.Interval(0)
	var exception z3exception.DefaultException
	if !errors.As(e, &exception) {
		t.Fatal("interval getter source assertion", e)
	}
	_, _, _, e = c.intervalSignPrefix(pm, pm.core.unit, &values)
	if !errors.As(e, &exception) || c.signInterval.constructed {
		t.Fatal("empty vars assertion after interval constructor", e)
	}
	serviceMust(t, c.CleanupRetained())
}
