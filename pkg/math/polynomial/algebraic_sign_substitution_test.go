package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"strconv"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestAlgebraicSignSubstitutionMixedIdentityAndReferenceLifetime(t *testing.T) {
	c, p, _ := algebraicControlledFixture(t)
	pm, poly, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{3, [3]uint32{2, 1, 0}}, {-2, [3]uint32{1, 0, 1}}, {-1, [3]uint32{}}})
	a, b := algebraicResultantInputs(t, c, p)
	values := algebraicSignTestValues{}
	values.values[1] = a
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 1, 2))
	serviceMust(t, c.SetRational(&values.values[0], &q))
	result, _, constant, e := c.prepareSignPolynomial(pm, poly, &values)
	serviceMust(t, e)
	if constant || !c.signSubstitution.active {
		t.Fatal("nonconstant reference did not remain live")
	}
	// Clearing x=1/2's squared denominator yields 3*y-4. Zero z
	// removes the x*z term while the irrational y is retained symbolically.
	polynomialQueryExpect(t, pm.core, result, map[[3]uint32]int64{{0, 1, 0}: 3, {0, 0, 0}: -4})
	if _, _, _, e = c.prepareSignPolynomial(pm, poly, &values); !errors.Is(e, ErrAlgebraicPending) {
		t.Fatal("live p_prime was overwritten", e)
	}
	serviceMust(t, c.finishSignSubstitution(false))
	serviceMust(t, c.Del(&values.values[0]))
	serviceMust(t, c.Del(&b))
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.config.Rationals.Del(&q))
}

func TestAlgebraicSignSubstitutionReservationPrefixesAndRetry(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 256 && !success; prefix++ {
		t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			c, p, _ := algebraicControlledFixture(t)
			pm, poly, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{3, [3]uint32{2, 1, 0}}, {-2, [3]uint32{1, 0, 1}}, {-1, [3]uint32{}}})
			a, b := algebraicResultantInputs(t, c, p)
			values := algebraicSignTestValues{}
			values.values[1] = a
			q := unumeral.NewRational()
			serviceMust(t, c.config.Rationals.SetFraction(&q, 4294967297, 11))
			serviceMust(t, c.SetRational(&values.values[0], &q))
			budget := c.config.Rationals.Budget()
			budget.FailAfter(prefix)
			_, _, _, body := c.prepareSignPolynomial(pm, poly, &values)
			budget.FailAfter(-1)
			if body == nil {
				success = true
			} else if !numericPhysicalReservationRefusal(body) {
				t.Fatal("non-reservation failure", body)
			}
			serviceMust(t, c.CleanupRetained())
			result, _, constant, e := c.prepareSignPolynomial(pm, poly, &values)
			// Source multiplication publishes its result before allocating a
			// replacement scratch cell. A failed replacement leaves a literal
			// scratch hole; owning cleanup must not manufacture a replacement.
			if c.config.Rationals.Integers().Tmp[0] == 0 {
				if !errors.Is(e, arena.ErrInvalidSlot) {
					t.Fatal("scratch hole was silently repaired", e)
				}
				serviceMust(t, c.CleanupRetained())
				serviceMust(t, c.Del(&values.values[0]))
				serviceMust(t, c.Del(&b))
				serviceMust(t, c.Del(&a))
				serviceMust(t, c.config.Rationals.Del(&q))
				return
			}
			serviceMust(t, e)
			if constant {
				t.Fatal("retry lost irrational variable")
			}
			// Evaluate the rebuilt polynomial independently at y=0: denominator
			// clearing leaves the constant -11^2, regardless of the BIG numerator.
			point := polynomialEvalRationalMap{manager: c.config.Rationals}
			for i := range point.values {
				point.values[i] = unumeral.NewRational()
			}
			out := unumeral.NewRational()
			serviceMust(t, pm.EvaluateRational(result, &point, &out))
			if text := rationalString(t, c.config.Rationals, &out); text != "-121" {
				t.Fatal(text)
			}
			serviceMust(t, c.config.Rationals.Del(&out))
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, c.Del(&values.values[0]))
			serviceMust(t, c.Del(&b))
			serviceMust(t, c.Del(&a))
			serviceMust(t, c.config.Rationals.Del(&q))
		})
	}
	if !success {
		t.Fatal("no successful reservation endpoint")
	}
}

func TestAlgebraicSignSubstitutionVanishingConstantAndBorrowedIdentity(t *testing.T) {
	c, p, _ := algebraicControlledFixture(t)
	pm, poly, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 1, 0}}, {5, [3]uint32{1, 0, 0}}})
	a, b := algebraicResultantInputs(t, c, p)
	values := algebraicSignTestValues{}
	values.values[1] = a
	result, sign, constant, e := c.prepareSignPolynomial(pm, poly, &values)
	serviceMust(t, e)
	if !constant || sign != 0 {
		t.Fatal("vanishing polynomial", sign, constant)
	}
	polynomialQueryExpect(t, pm.core, result, map[[3]uint32]int64{})
	serviceMust(t, c.finishSignSubstitution(false))
	// With only x in the input, replacing x by 2/3 clears denominator
	// and primitive-normalizes the constant polynomial to coefficient 1.
	x, e := pm.core.MkVariable(0, 1)
	serviceMust(t, e)
	serviceMust(t, pm.core.IncRef(x))
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 2, 3))
	serviceMust(t, c.SetRational(&values.values[0], &q))
	_, sign, constant, e = c.prepareSignPolynomial(pm, x, &values)
	serviceMust(t, e)
	if !constant || sign != 1 {
		t.Fatal("source constant sign", sign, constant)
	}
	serviceMust(t, c.finishSignSubstitution(false))
	serviceMust(t, c.Del(&values.values[0]))
	values.values[0] = b
	result, _, constant, e = c.prepareSignPolynomial(pm, poly, &values)
	serviceMust(t, e)
	if constant || result != poly {
		t.Fatal("all-root map should preserve the original polynomial")
	}
	serviceMust(t, c.finishSignSubstitution(false))
	serviceMust(t, pm.core.DecRef(x))
	serviceMust(t, c.Del(&b))
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.config.Rationals.Del(&q))
}

func TestAlgebraicSignSubstitutionGetterSourceAssertion(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	c, p, _ := algebraicControlledFixture(t)
	a, b := algebraicResultantInputs(t, c, p)
	values := algebraicSignTestValues{}
	values.values[0] = a
	mapping := algebraicBasicRationals{algebraicOptimisticRationals: algebraicOptimisticRationals{owner: c, values: &values}}
	polynomialAssertionActionsFixture(t, "T")
	_, e := mapping.Rational(0)
	var exception z3exception.DefaultException
	if !errors.As(e, &exception) {
		t.Fatal("getter did not use source assertion action", e)
	}
	serviceMust(t, c.Del(&b))
	serviceMust(t, c.Del(&a))
}
