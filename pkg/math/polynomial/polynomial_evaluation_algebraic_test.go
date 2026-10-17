package polynomial

import (
	"errors"
	"fmt"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialEvaluationAlgebraicRationalGrid(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{3, [3]uint32{2, 1, 0}}, {-2, [3]uint32{1, 0, 1}}, {-1, [3]uint32{}}})
	values := algebraicSignTestValues{}
	q := unumeral.NewRational()
	var out Algebraic
	defer func() {
		serviceMust(t, c.Del(&out))
		for i := range values.values {
			serviceMust(t, c.Del(&values.values[i]))
		}
		serviceMust(t, c.config.Rationals.Del(&q))
	}()
	for x := int64(-2); x <= 2; x++ {
		for y := int64(-2); y <= 2; y++ {
			for z := int64(-2); z <= 2; z++ {
				for i, n := range []int64{x, y, z} {
					serviceMust(t, c.config.Rationals.SetFraction(&q, n, 2))
					serviceMust(t, c.SetRational(&values.values[i], &q))
				}
				serviceMust(t, pm.EvaluateAlgebraic(p, c, &values, &out))
				serviceMust(t, c.config.Rationals.SetFraction(&q, 3*x*x*y-4*x*z-8, 8))
				cmp, e := c.CompareRational(&out, &q)
				serviceMust(t, e)
				if cmp != 0 {
					t.Fatal(x, y, z, cmp)
				}
			}
		}
	}
	if pm.algebraicEvaluation.active || c.polynomialEvaluationBorrowers != 0 {
		t.Fatal("successful evaluation retained owner")
	}
}

func TestPolynomialEvaluationAlgebraicRootsAndEarlyRoutes(t *testing.T) {
	c, up, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{3, [3]uint32{2, 1, 0}}, {-2, [3]uint32{1, 0, 1}}, {-1, [3]uint32{}}})
	values := algebraicSignTestValues{}
	values.values[0], values.values[1] = algebraicResultantInputs(t, c, up)
	var out Algebraic
	defer func() {
		serviceMust(t, c.Del(&out))
		for i := range values.values {
			serviceMust(t, c.Del(&values.values[i]))
		}
	}()
	for repeat := 0; repeat < 2; repeat++ {
		serviceMust(t, pm.EvaluateAlgebraic(p, c, &values, &out))
		algebraicExpectBetween(t, c, &out, 9, 10)
	}
	zero := pm.core.zero
	n := unumeral.NewInt(-7)
	constant, e := pm.core.MkConstantTakeCopy(&n)
	serviceMust(t, e)
	serviceMust(t, pm.core.IncRef(constant))
	defer func() { serviceMust(t, pm.core.DecRef(constant)) }()
	values.failure = errors.New("early routes must not read mapping")
	reads := len(values.reads)
	serviceMust(t, pm.EvaluateAlgebraic(constant, c, &values, &out))
	algebraicExpectBetween(t, c, &out, -8, -6)
	serviceMust(t, pm.EvaluateAlgebraic(zero, c, &values, &out))
	if out != (Algebraic{}) || len(values.reads) != reads {
		t.Fatal("early route read mapping or failed to reset")
	}
}

func TestPolynomialEvaluationAlgebraicCancellationAndRetry(t *testing.T) {
	succeeded := false
	for allowance := uint64(0); allowance < 512 && !succeeded; allowance++ {
		t.Run(fmt.Sprint(allowance), func(t *testing.T) {
			c, up, control := algebraicControlledFixture(t)
			pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{0, 1, 0}}, {1, [3]uint32{}}})
			values := algebraicSignTestValues{}
			values.values[0], values.values[1] = algebraicResultantInputs(t, c, up)
			var out Algebraic
			defer func() {
				serviceMust(t, c.Del(&out))
				for i := range values.values {
					serviceMust(t, c.Del(&values.values[i]))
				}
			}()
			control.limit = control.visits + allowance
			body := pm.EvaluateAlgebraic(p, c, &values, &out)
			control.limit = 1000000
			succeeded = body == nil
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, pm.CleanupAlgebraicEvaluation())
			serviceMust(t, pm.EvaluateAlgebraic(p, c, &values, &out))
			algebraicExpectBetween(t, c, &out, 4, 5)
		})
	}
	if !succeeded {
		t.Fatal("did not exhaust successful evaluator checkpoints")
	}
}

func TestPolynomialEvaluationAlgebraicLexOwnershipAndReadStop(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	pm, p, control := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
	values := algebraicSignTestValues{}
	var out Algebraic
	defer func() { serviceMust(t, c.Del(&out)) }()
	wrapped := polynomialLexWrappedPolynomial(t, pm)
	if e := pm.EvaluateAlgebraic(wrapped, c, &values, &out); e == nil || !pm.lexPending || !pm.algebraicEvaluation.active {
		t.Fatal("did not retain actual lex stop", e)
	}
	if e := c.Close(); !errors.Is(e, ErrAlgebraicPending) {
		t.Fatal("borrowed manager close", e)
	}
	rv := polynomialEvalRationalMap{manager: c.config.Rationals}
	ro := unumeral.NewRational()
	if e := pm.EvaluateRational(p, &rv, &ro); !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("other evaluator consumed pending owner", e)
	}
	generation := pm.algebraicEvaluation.lexGeneration
	serviceMust(t, pm.CleanupEvaluation())
	if !pm.lexPending || pm.algebraicEvaluation.lexGeneration != generation {
		t.Fatal("unrelated cleanup consumed lex owner")
	}
	serviceMust(t, pm.CleanupAlgebraicEvaluation())
	if c.polynomialEvaluationBorrowers != 0 {
		t.Fatal("retained borrow")
	}
	marker := errors.New("mapped value read stop")
	values.failure = marker
	if e := pm.EvaluateAlgebraic(p, c, &values, &out); !errors.Is(e, marker) {
		t.Fatal(e)
	}
	if pm.algebraicEvaluation.active {
		t.Fatal("read failure retained already destroyed auxiliaries")
	}
	values.failure = nil
	// The polynomial checkpoint itself is distinct from algebraic arithmetic's.
	control.limit = control.checkpoints
	if e := pm.EvaluateAlgebraic(p, c, &values, &out); e == nil {
		t.Fatal("missing polynomial cancellation")
	}
	control.limit = 100000
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, pm.CleanupAlgebraicEvaluation())
	serviceMust(t, pm.EvaluateAlgebraic(p, c, &values, &out))
	algebraicExpectBetween(t, c, &out, 0, 2)
}

type polynomialAlgebraicOwnerProbe struct {
	algebraicSignTestValues
	t      *testing.T
	pm     *MultivariatePolynomialManager
	vm     *AlgebraicRootCore
	probes uint32
}

func (v *polynomialAlgebraicOwnerProbe) Algebraic(x uint32) (*Algebraic, error) {
	v.probes++
	if e := v.pm.CleanupAlgebraicEvaluation(); !errors.Is(e, ErrPolynomialPending) {
		v.t.Fatal("executing auxiliary cleanup", e)
	}
	if e := v.pm.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
		v.t.Fatal("executing polynomial owner close", e)
	}
	if e := v.vm.CleanupRetained(); !errors.Is(e, ErrAlgebraicPending) {
		v.t.Fatal("executing algebraic owner cleanup", e)
	}
	if e := v.vm.Close(); !errors.Is(e, ErrAlgebraicPending) {
		v.t.Fatal("executing algebraic owner close", e)
	}
	return v.algebraicSignTestValues.Algebraic(x)
}
func TestPolynomialEvaluationAlgebraicExecutingOwnersAndSingleTerm(t *testing.T) {
	c, up, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{2, 2, 0}}})
	values := polynomialAlgebraicOwnerProbe{t: t, pm: pm, vm: c}
	values.values[0], values.values[1] = algebraicResultantInputs(t, c, up)
	var out Algebraic
	defer func() {
		serviceMust(t, c.Del(&out))
		for i := range values.values {
			serviceMust(t, c.Del(&values.values[i]))
		}
	}()
	serviceMust(t, pm.EvaluateAlgebraic(p, c, &values, &out))
	n := unumeral.NewInt(6)
	var expected Algebraic
	serviceMust(t, c.SetInteger(&expected, &n))
	cmp, e := c.Compare(&out, &expected)
	serviceMust(t, e)
	serviceMust(t, c.Del(&expected))
	if cmp != 0 || values.probes == 0 {
		t.Fatal("single-term evaluation", cmp, values.probes)
	}
}
