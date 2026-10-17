package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/z3exception"
	"testing"
)

func TestAlgebraicCheckpointSourceExceptionAndResume(t *testing.T) {
	f := algebraicSourceFixture(t)
	before := f.limit.Count()
	cooperationCalls := 0
	original := f.core.config.Cooperation
	f.core.config.Cooperation = func() error { cooperationCalls++; return original() }
	f.limit.Cancel()
	e := f.core.checkpoint()
	var specific AlgebraicException
	var base z3exception.DefaultException
	if !errors.As(e, &specific) || !errors.As(e, &base) || e.Error() != "canceled" || base.Message() != "canceled" || specific.HasErrorCode() || !errors.Is(e, ErrUPolynomialCanceled) {
		t.Fatal("source algebraic cancellation class/message", e)
	}
	if f.limit.Count() != before+1 || f.algebraicTicks != 1 || cooperationCalls != 0 {
		t.Fatal("checkpoint order", f.limit.Count(), before, f.algebraicTicks)
	}
	f.limit.ResetCancel()
	algebraicSourceMust(t, f.core.checkpoint())
	if f.limit.Count() != before+2 || f.algebraicTicks != 2 || cooperationCalls != 1 {
		t.Fatal("resumed checkpoint", f.limit.Count(), f.algebraicTicks)
	}
	// A matching message alone does not classify an arbitrary root-object error
	// as resource cancellation.
	ordinary := newAlgebraicException("canceled")
	if errors.Is(ordinary, ErrUPolynomialCanceled) {
		t.Fatal("message classified as cancellation")
	}
}

func TestAlgebraicArithmeticSourceExceptionClasses(t *testing.T) {
	c, _, control := algebraicCompareFixture(t)
	polynomialAssertionActionsFixture(t, "CC")
	var negative, out, zero Algebraic
	minusOne := unumeral.NewInt(-1)
	one := unumeral.NewInt(1)
	serviceMust(t, c.SetInteger(&negative, &minusOne))
	serviceMust(t, c.SetInteger(&out, &one))
	defer func() { serviceMust(t, c.Del(&negative)); serviceMust(t, c.Del(&out)) }()
	for _, v := range []struct {
		name       string
		call       func() error
		want       error
		message    string
		diagnostic bool
	}{
		{"zeroth-root", func() error { return c.Root(nil, 0, &out) }, ErrAlgebraicZeroRoot, "0-th root is indeterminate", false},
		{"zero-power", func() error { return c.Power(&zero, 0, &out) }, ErrAlgebraicZeroPower, "0^0 is indeterminate", false},
		{"negative-even-root", func() error { return c.Root(&negative, 2, &out) }, ErrAlgebraicEvenNegativeRoot, "even root of negative number is not real", false},
		{"zero-inverse", func() error { return c.Inv(&zero) }, ErrAlgebraicDivisionByZero, "division by zero", true},
		{"zero-divisor", func() error { return c.Div(nil, &zero, &out) }, ErrAlgebraicDivisionByZero, "division by zero", true},
	} {
		t.Run(v.name, func(t *testing.T) {
			before, checks, prior := c.config.Rationals.Budget().Used(), control.checks, out
			c.config.Rationals.Budget().FailAfter(0)
			e := v.call()
			c.config.Rationals.Budget().FailAfter(-1)
			if v.diagnostic && !polynomialSourceDebug {
				if !errors.Is(e, diagnostic.ErrDebugNativeUnreachable) {
					t.Fatal("source unreachable must precede throw", e)
				}
			} else {
				var specific AlgebraicException
				var base z3exception.DefaultException
				if !errors.Is(e, v.want) || !errors.As(e, &specific) || !errors.As(e, &base) || e.Error() != v.message || base.Message() != v.message || specific.HasErrorCode() {
					t.Fatal("source arithmetic exception class/message", e)
				}
				if errors.Is(e, ErrUPolynomialCanceled) {
					t.Fatal("arithmetic exception classified as cancellation", e)
				}
			}
			if out != prior || c.config.Rationals.Budget().Used() != before || control.checks != checks || c.division.active || c.basicRootScope.active || c.inversion.remaining != 0 {
				t.Fatal("error performed numerical work or changed destination")
			}
		})
	}
}
