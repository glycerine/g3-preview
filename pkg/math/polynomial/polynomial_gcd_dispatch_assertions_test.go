package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialGCDDispatcherSourceAssertionActions(t *testing.T) {
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
			e := polynomialGCDDispatchSourceAssert(4448, "max_var(u) == max_var(v)", func() (bool, error) {
				reads++
				return false, nil
			})
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) || reads != 1 {
					t.Fatal("GCD dispatcher assertion did not throw", e, reads)
				}
			} else if polynomialSourceDebug && action == "disabled" {
				if e != nil || reads != 0 {
					t.Fatal("disabled GCD dispatcher assertion evaluated predicate", e, reads)
				}
			} else if e != nil || (polynomialSourceDebug && reads != 1) || (!polynomialSourceDebug && reads != 0) {
				t.Fatal("Continue/release GCD dispatcher assertion behavior", e, reads)
			}
		})
	}
}
