package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialSubstitutionSourceAssertionActions(t *testing.T) {
	for _, site := range []struct {
		line       int32
		expression string
	}{{6313, "position empty"}, {6320, "position present"}, {6388, "variable absent"}, {6414, "degree order"}} {
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
				e := polynomialSubstitutionSourceAssert(site.line, site.expression, func() (bool, error) {
					reads++
					return false, nil
				})
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrPolynomialAssertion) || reads != 1 {
						t.Fatal("substitution assertion did not throw", e, reads)
					}
				} else if polynomialSourceDebug && action == "disabled" {
					if e != nil || reads != 0 {
						t.Fatal("disabled substitution assertion evaluated predicate", e, reads)
					}
				} else if e != nil || (polynomialSourceDebug && reads != 1) || (!polynomialSourceDebug && reads != 0) {
					t.Fatal("Continue/release substitution assertion behavior", e, reads)
				}
			})
		}
	}
}
