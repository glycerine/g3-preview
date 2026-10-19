package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialSqrtSourceAssertionActions(t *testing.T) {
	for _, site := range []struct {
		line       int32
		expression string
	}{{5965, "p != 0"}, {6006, "m1 != 0"}} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(action+"/"+site.expression, func(t *testing.T) {
				choices := action
				if action == "disabled" {
					choices = ""
				}
				polynomialAssertionActionsFixture(t, choices)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				reads := 0
				e := polynomialSqrtSourceAssert(site.line, site.expression, func() (bool, error) {
					reads++
					return false, nil
				})
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrPolynomialAssertion) || reads != 1 {
						t.Fatal("sqrt assertion did not throw", e, reads)
					}
				} else if polynomialSourceDebug && action == "disabled" {
					if e != nil || reads != 0 {
						t.Fatal("disabled sqrt assertion evaluated predicate", e, reads)
					}
				} else if e != nil || (polynomialSourceDebug && reads != 1) || (!polynomialSourceDebug && reads != 0) {
					t.Fatal("Continue/release sqrt assertion behavior", e, reads)
				}
			})
		}
	}
}
