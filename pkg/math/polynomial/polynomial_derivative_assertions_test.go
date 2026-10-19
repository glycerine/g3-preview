package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialDerivativeSourceAssertionActions(t *testing.T) {
	for _, site := range []struct {
		line       int32
		expression string
	}{{4512, "is_valid(x)"}, {4513, "m_cheap_som_buffer.empty()"}} {
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
				e := polynomialDerivativeSourceAssert(site.line, site.expression, func() (bool, error) {
					reads++
					return false, nil
				})
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrPolynomialAssertion) || reads != 1 {
						t.Fatal("derivative assertion did not throw", e, reads)
					}
				} else if polynomialSourceDebug && action == "disabled" {
					if e != nil || reads != 0 {
						t.Fatal("disabled derivative assertion evaluated predicate", e, reads)
					}
				} else if e != nil || (polynomialSourceDebug && reads != 1) || (!polynomialSourceDebug && reads != 0) {
					t.Fatal("Continue/release derivative assertion behavior", e, reads)
				}
			})
		}
	}
}
