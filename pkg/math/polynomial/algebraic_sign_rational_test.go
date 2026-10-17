package polynomial

import (
	"errors"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type algebraicSignTestValues struct {
	values  [3]Algebraic
	reads   []uint32
	failure error
}

func (v *algebraicSignTestValues) Contains(x uint32) (bool, error) { return x < 3, nil }
func (v *algebraicSignTestValues) Algebraic(x uint32) (*Algebraic, error) {
	v.reads = append(v.reads, x)
	if v.failure != nil {
		return nil, v.failure
	}
	if x >= 3 {
		return nil, ErrAlgebraicOwner
	}
	return &v.values[x], nil
}

func TestAlgebraicOptimisticSignExactRationalGrid(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{3, [3]uint32{2, 1, 0}}, {-2, [3]uint32{1, 0, 1}}, {-1, [3]uint32{}}})
	values := algebraicSignTestValues{}
	q := unumeral.NewRational()
	defer func() {
		for i := range values.values {
			serviceMust(t, c.Del(&values.values[i]))
		}
		serviceMust(t, c.config.Rationals.Del(&q))
	}()
	for x := int32(-2); x <= 2; x++ {
		for y := int32(-2); y <= 2; y++ {
			for z := int32(-2); z <= 2; z++ {
				for i, n := range []int32{x, y, z} {
					serviceMust(t, c.config.Rationals.SetFraction(&q, int64(n), 2))
					serviceMust(t, c.SetRational(&values.values[i], &q))
				}
				sign, all, e := c.tryRationalSign(pm, p, &values)
				serviceMust(t, e)
				// Multiply by eight: 3*x*x*y - 4*x*z - 8.
				exact := 3*x*x*y - 4*x*z - 8
				// qm().sign returns the full inline numerator, rather than a
				// normalized -1/+1. Reduce the independent exact value /8.
				magnitude := exact
				if magnitude < 0 {
					magnitude = -magnitude
				}
				denominator := int32(8)
				for denominator != 0 {
					magnitude, denominator = denominator, magnitude%denominator
				}
				want := exact / magnitude

				if !all || sign != int(want) {
					t.Fatal(x, y, z, sign, all, want)
				}
			}
		}
	}
}

func TestAlgebraicOptimisticSignRootSentinelAndLookupFailure(t *testing.T) {
	c, p, _ := algebraicControlledFixture(t)
	pm, poly, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}})
	a, b := algebraicResultantInputs(t, c, p)
	values := algebraicSignTestValues{}
	values.values[0] = a
	sign, all, e := c.tryRationalSign(pm, poly, &values)
	serviceMust(t, e)
	if all || sign != 0 || len(values.reads) != 1 || values.reads[0] != 0 {
		t.Fatal("nonbasic mapping did not stop at actual lookup", sign, all, values.reads)
	}
	marker := errors.New("assignment lookup stopped")
	values.failure = marker
	_, all, e = c.tryRationalSign(pm, poly, &values)
	if !errors.Is(e, marker) || all {
		t.Fatal("arbitrary error mistaken for nonbasic sentinel", e)
	}
	// Constants never ask the assignment for unused root cells or failed
	// lookups; the optimistic path must not pre-scan the map.
	reads := len(values.reads)
	for _, constant := range []Polynomial{pm.core.zero, pm.core.unit} {
		_, all, e = c.tryRationalSign(pm, constant, &values)
		serviceMust(t, e)
		if !all || len(values.reads) != reads {
			t.Fatal("constant touched unused assignment")
		}
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.Del(&b))
	serviceMust(t, c.Del(&a))
}

func TestAlgebraicOptimisticSignCancellationAndForeignPendingScope(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	pm, p, control := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{2, 1, 0}}, {1, [3]uint32{}}})
	values := algebraicSignTestValues{}
	control.limit = 0
	_, _, e := c.tryRationalSign(pm, p, &values)
	if !errors.Is(e, ErrUPolynomialCanceled) {
		t.Fatal(e)
	}
	control.limit = 100000
	serviceMust(t, c.CleanupRetained())
	sign, all, e := c.tryRationalSign(pm, p, &values)
	serviceMust(t, e)
	if !all || sign != 1 {
		t.Fatal(sign, all)
	}
	// A real LexSort bounds stop belongs to an older independent caller.
	wrapped := polynomialLexWrappedPolynomial(t, pm)
	oldValues := polynomialEvalRationalMap{manager: c.config.Rationals}
	for i := range oldValues.values {
		oldValues.values[i] = unumeral.NewRational()
	}
	oldOut := unumeral.NewRational()
	older := pm.EvaluateRational(wrapped, &oldValues, &oldOut)
	if older == nil || !pm.evaluation.active || !pm.lexPending {
		t.Fatal("older bounds stop not reached", older)
	}
	oldLex := pm.evaluation.lexGeneration
	_, _, e = c.tryRationalSign(pm, p, &values)
	if !errors.Is(e, ErrPolynomialPending) {
		t.Fatal(e)
	}
	if !pm.evaluation.active {
		t.Fatal("foreign pending evaluation was consumed")
	}
	serviceMust(t, c.CleanupRetained())
	if !pm.evaluation.active || pm.evaluation.lexGeneration != oldLex {
		t.Fatal("foreign invocation changed during parent cleanup")
	}
	serviceMust(t, pm.CleanupEvaluation())
	serviceMust(t, c.config.Rationals.Del(&oldOut))
	for i := range oldValues.values {
		serviceMust(t, c.config.Rationals.Del(&oldValues.values[i]))
	}
}
