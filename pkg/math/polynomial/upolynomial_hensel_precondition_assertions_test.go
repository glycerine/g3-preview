package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialHenselPairPreconditionSourceActions(t *testing.T) {
	for _, route := range []string{"product-degree", "bezout-degree"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(route+"/"+action, func(t *testing.T) {
				_, s := preparationPhysicalFixture(t)
				m := s.manager
				a, b := upolySourceNew(t, m, []int32{0, 1}), upolySourceNew(t, m, []int32{1, 1})
				uc, cc := []int32{-1}, []int32{1, 1}
				if route == "bezout-degree" {
					uc, cc = []int32{0, 1}, []int32{3, 1, 1}
				}
				u, v, c := upolySourceNew(t, m, uc), upolySourceNew(t, m, []int32{1}), upolySourceNew(t, m, cc)
				al, bl := upolySourceNew(t, m, []int32{71}), upolySourceNew(t, m, []int32{72})
				choice := action
				if action == "C" && route == "product-degree" {
					choice = "CCC"
				}
				if action == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				one, three := unumeral.NewInt(1), unumeral.NewInt(3)
				e := s.HenselLiftPair(&one, &three, &three, mustUPolyView(t, m, u), mustUPolyView(t, m, a), mustUPolyView(t, m, v), mustUPolyView(t, m, b), mustUPolyView(t, m, c), al, bl)
				// Source mpz::div rounds a negative non-exact coefficient downward:
				// f=-x^2, hence T=-x and B_lifted=1-2*x.
				wantA, wantB := []int32{0, 1}, []int32{1, -2}
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("source precondition Throw", e)
					}
					wantA, wantB = []int32{71}, []int32{72}
				} else {
					serviceMust(t, e)
					if route == "bezout-degree" {
						wantA, wantB = []int32{3, 1}, []int32{1, 4}
					}
				}
				pa, pb := upolySourceNew(t, m, wantA), upolySourceNew(t, m, wantB)
				for _, pair := range [2][2]UPolynomial{{al, pa}, {bl, pb}} {
					same, e := m.Equal(pair[0], pair[1])
					serviceMust(t, e)
					if !same {
						t.Fatal("precondition action changed source output frontier", route, action, upolyDivisionValues(t, m, al), upolyDivisionValues(t, m, bl))
					}
				}
				serviceMust(t, s.CleanupRetained())
				if m.factorTrace != nil {
					t.Fatal("precondition retained assertion borrow")
				}
			})
		}
	}
}
