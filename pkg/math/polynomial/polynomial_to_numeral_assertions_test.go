package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialToNumeralSourceAssertionActions(t *testing.T) {
	for _, line := range []int32{404, 415} {
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
				e := polynomialToNumeralSourceAssert(line, "converter assertion", func() (bool, error) {
					reads++
					return false, nil
				})
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrPolynomialAssertion) || reads != 1 {
						t.Fatal("converter assertion did not throw", line, e, reads)
					}
				} else if polynomialSourceDebug && action == "disabled" {
					if e != nil || reads != 0 {
						t.Fatal("disabled converter assertion evaluated predicate", line, e, reads)
					}
				} else if e != nil || (polynomialSourceDebug && reads != 1) || (!polynomialSourceDebug && reads != 0) {
					t.Fatal("Continue/release converter assertion behavior", line, e, reads)
				}
			})
		}
	}
}
