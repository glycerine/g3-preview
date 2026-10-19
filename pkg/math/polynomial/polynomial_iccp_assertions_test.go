package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialICCPSourceAssertionActions(t *testing.T) {
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
			e := polynomialICCPSourceAssert(3501, "!iccp_powers.empty()", func() (bool, error) {
				reads++
				return false, nil
			})
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) || reads != 1 {
					t.Fatal("ICCP assertion did not throw", e, reads)
				}
			} else if polynomialSourceDebug && action == "disabled" {
				if e != nil || reads != 0 {
					t.Fatal("disabled ICCP assertion evaluated predicate", e, reads)
				}
			} else if e != nil || (polynomialSourceDebug && reads != 1) || (!polynomialSourceDebug && reads != 0) {
				t.Fatal("Continue/release ICCP assertion behavior", e, reads)
			}
		})
	}
}
