package polynomial

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialConvertBoundPredicateActions(t *testing.T) {
	for _, line := range []int32{2833, 2844, 2845, 2849, 2879, 2887, 2888, 2894, 2914, 2915} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(fmt.Sprint(line)+"/"+action, func(t *testing.T) {
				m, q, budget := upolySourceOwner(t)
				bqm := upolyIsolationBinaryOwner(t, q, budget)
				y := unumeral.NewRational()
				serviceMust(t, q.SetFraction(&y, 3, 2))
				n := int32(2)
				if line == 2845 || line == 2887 || line == 2914 {
					n = 1
				}
				x := unumeral.NewBinaryIntPower(n, 0)
				t.Cleanup(func() { serviceMust(t, bqm.Del(&x)); serviceMust(t, q.Del(&y)) })
				choice := action
				if action == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				// Direct invariant adapter uses independent actual values, without replacing
				// conversion scratch or inventing a stopped parent construction frontier.
				e := m.convertBoundAssertion(line, bqm, &x, q, &y)
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("bound assertion Throw", e)
					}
				} else {
					serviceMust(t, e)
				}
				if upolyIsolationBinaryRat(t, m, bqm, &x).RatString() != fmt.Sprint(n) {
					t.Fatal("predicate changed caller bound")
				}
			})
		}
	}
}
func TestUPolynomialConvertBoundPredicateErasure(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	for _, line := range []int32{2833, 2844, 2845, 2849, 2879, 2887, 2888, 2894, 2914, 2915} {
		serviceMust(t, m.convertBoundAssertion(line, nil, nil, nil, nil))
	}
}
