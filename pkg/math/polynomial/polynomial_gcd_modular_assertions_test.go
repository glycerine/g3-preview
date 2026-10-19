package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialModGCDSourceAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			choices := action
			if action == "disabled" {
				choices = ""
			}
			polynomialAssertionActionsFixture(t, choices)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			reads := 0
			e := polynomialModGCDSourceAssert(4249, "u_var_degrees[i].get_var() == v_var_degrees[i].get_var()", func() (bool, error) {
				reads++
				return false, nil
			})
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) || reads != 1 {
					t.Fatal("modular GCD assertion did not throw", e, reads)
				}
			} else if polynomialSourceDebug && action == "disabled" {
				if e != nil || reads != 0 {
					t.Fatal("disabled modular GCD assertion evaluated predicate", e, reads)
				}
			} else if e != nil || (polynomialSourceDebug && reads != 1) || (!polynomialSourceDebug && reads != 0) {
				t.Fatal("Continue/release modular GCD assertion behavior", e, reads)
			}
		})
	}
}

func TestPolynomialModGCDRecursiveTerminalAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "CCCCCCCC"} {
		t.Run(action, func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			serviceMust(t, c.modular.SetZPUint64(11))
			u := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}})
			out, e := NewPolynomialRef(c)
			serviceMust(t, e)
			polynomialAssertionActionsFixture(t, action)
			e = full.modGCDRec(u, u, 11, 0, []uint32{0}, &out.root)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) {
					t.Fatal("recursive univariate assertion did not throw", e)
				}
			} else {
				serviceMust(t, e)
				equal, x := c.Equal(out.root.value, u)
				serviceMust(t, x)
				if !equal {
					t.Fatal("Continue/release did not reach terminal Euclid result")
				}
			}
			serviceMust(t, out.Close())
		})
	}
}
