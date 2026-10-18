// Source: src/math/polynomial/upolynomial_factorization.cpp:1284-1292.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialRecombinationConstantRemainderAssertionActions(t *testing.T) {
	for _, constant := range []string{"empty", "nonunit", "unit"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(constant+"/"+action, func(t *testing.T) {
				q, s, fs := recombinationPublicationFixture(t)
				m, c := s.manager, s.scope
				t.Cleanup(func() { serviceMust(t, s.CleanupRetained()); serviceMust(t, fs.Close()) })
				var coefficients []unumeral.Int
				if constant == "nonunit" {
					coefficients = []unumeral.Int{unumeral.NewInt(2)}
				}
				if constant == "unit" {
					coefficients = []unumeral.Int{unumeral.NewInt(1)}
				}
				serviceMust(t, m.set(&c.fpp, coefficients))
				choice := action
				if action == "disabled" || constant == "unit" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				before := q.Budget().Used()
				e := s.publishRecombinationRemainder(fs, 5)
				if polynomialSourceDebug && action == "T" && constant != "unit" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("remainder source Throw", e)
					}
				} else {
					serviceMust(t, e)
				}
				if c.fpp.size != uint32(len(coefficients)) || fs.count != 0 || fs.totalFactors != 0 || q.Budget().Used() != before {
					t.Fatal("constant branch changed source remainder, output or charged storage")
				}
				if len(coefficients) != 0 && m.values(&c.fpp)[0] != coefficients[0] {
					t.Fatal("constant remainder changed")
				}
				if m.factorTrace != nil || c.recombinationStep.active {
					t.Fatal("constant branch retained source call or debugger borrow")
				}
				primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, nil)
			})
		}
	}
}
