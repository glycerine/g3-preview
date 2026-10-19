package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialFactorSourceAssertionActions(t *testing.T) {
	for _, line := range []int32{6592, 6593, 6594, 6728, 6729, 6730} {
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
				e := polynomialFactorSourceAssert(line, "factor precondition", func() (bool, error) {
					reads++
					return false, nil
				})
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrPolynomialAssertion) || reads != 1 {
						t.Fatal("factor assertion did not throw", line, e, reads)
					}
				} else if polynomialSourceDebug && action == "disabled" {
					if e != nil || reads != 0 {
						t.Fatal("disabled factor assertion evaluated predicate", line, e, reads)
					}
				} else if e != nil || (polynomialSourceDebug && reads != 1) || (!polynomialSourceDebug && reads != 0) {
					t.Fatal("Continue/release factor assertion behavior", line, e, reads)
				}
			})
		}
	}
}
