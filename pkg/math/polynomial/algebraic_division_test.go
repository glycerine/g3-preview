package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"testing"
)

func TestAlgebraicDivisionRootsAndAliases(t *testing.T) {
	for _, alias := range []int{0, 1, 2} {
		c, p, _ := algebraicControlledFixture(t)
		a, b := algebraicResultantInputs(t, c, p)
		var result Algebraic
		out := &result
		if alias == 1 {
			out = &a
		}
		if alias == 2 {
			out = &b
		}
		serviceMust(t, c.Div(&a, &b, out))
		algebraicExpectBetween(t, c, out, 0, 1)
		if c.division.active || c.division.inverse != (Algebraic{}) {
			t.Fatal("division left its scoped inverse alive")
		}
		serviceMust(t, c.Del(&result))
		serviceMust(t, c.Del(&b))
		serviceMust(t, c.Del(&a))
	}
}

func TestAlgebraicDivisionSelfAndZeroNumerator(t *testing.T) {
	c, p, _ := algebraicControlledFixture(t)
	a, b := algebraicResultantInputs(t, c, p)
	var zero, out Algebraic
	serviceMust(t, c.Div(&zero, &b, &out))
	if out != (Algebraic{}) {
		t.Fatal("zero quotient is nonzero")
	}
	serviceMust(t, c.Div(&a, &a, &a))
	q := c.config.Rationals
	one := unumeral.NewRational()
	serviceMust(t, q.SetInt32(&one, 1))
	cmp, e := c.CompareRational(&a, &one)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("self quotient differs from one")
	}
	serviceMust(t, q.Del(&one))
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.Del(&b))
	serviceMust(t, c.Del(&out))
}

func TestAlgebraicDivisionZeroDivisorBeforeNumerator(t *testing.T) {
	c, _ := algebraicFixture(t)
	polynomialAssertionActionsFixture(t, "C")
	var zero, out Algebraic
	want := error(diagnostic.ErrDebugNativeUnreachable)
	if polynomialSourceDebug {
		want = ErrAlgebraicDivisionByZero
	}
	if e := c.Div(nil, &zero, &out); !errors.Is(e, want) {
		t.Fatal("zero divisor", e)
	}
	if c.division.active || out != (Algebraic{}) {
		t.Fatal("zero divisor constructed locals or changed output")
	}
}

func TestAlgebraicDivisionCancellationCleanupAndRetry(t *testing.T) {
	success := false
	for allowance := uint64(0); allowance < 512 && !success; allowance++ {
		t.Run(strconv.FormatUint(allowance, 10), func(t *testing.T) {
			c, p, control := algebraicControlledFixture(t)
			a, b := algebraicResultantInputs(t, c, p)
			var out Algebraic
			control.limit = control.visits + allowance
			body := c.Div(&a, &b, &out)
			control.limit = 1000000
			if body == nil {
				success = true
			} else {
				if !errors.Is(body, ErrUPolynomialCanceled) {
					t.Fatal(body)
				}
				if !errors.Is(c.Div(&a, &b, &out), ErrAlgebraicPending) {
					t.Fatal("pending inverse overwritten")
				}
				serviceMust(t, c.CleanupRetained())
				serviceMust(t, c.Div(&a, &b, &out))
			}
			algebraicExpectBetween(t, c, &out, 0, 1)
			if c.division.active {
				t.Fatal("scoped inverse survived success")
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&b))
			serviceMust(t, c.Del(&a))
		})
	}
	if !success {
		t.Fatal("division never reached success")
	}
}
