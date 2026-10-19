package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialGCDEuclidFieldAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			out, e := NewPolynomialRef(c)
			serviceMust(t, e)
			choices := action
			if action == "disabled" {
				choices = ""
			}
			polynomialAssertionActionsFixture(t, choices)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			e = full.gcdEuclid(c.zero, c.zero, &out.root)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) {
					t.Fatal("nonmodular assertion did not throw", e)
				}
			} else {
				serviceMust(t, e)
			}
			serviceMust(t, out.Close())
		})
	}
}

func TestPolynomialGCDEuclidUnivariateAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C"} {
		t.Run(action, func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			serviceMust(t, c.modular.SetZPUint64(11))
			u := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}})
			out, e := NewPolynomialRef(c)
			serviceMust(t, e)
			polynomialAssertionActionsFixture(t, action)
			e = full.gcdEuclid(u, c.zero, &out.root)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) {
					t.Fatal("nonunivariate assertion did not throw", e)
				}
			} else {
				serviceMust(t, e)
				got, x := out.Get()
				serviceMust(t, x)
				equal, x := c.Equal(got, u)
				serviceMust(t, x)
				if !equal {
					t.Fatal("Continue/release did not reach zero shortcut")
				}
			}
			serviceMust(t, out.Close())
		})
	}
}
