package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestUPolynomialNonzeroBoundAssertionActions(t *testing.T) {
	for _, route := range []string{"empty", "all-zero"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(route+"/"+action, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				p := upolySourceNew(t, m, []int32{1})
				if route == "all-zero" {
					serviceMust(t, m.Reset(p))
					serviceMust(t, m.Reserve(p, 2))
				}
				v := mustUPolyView(t, m, p)
				if route == "empty" {
					v = v.WithCount(0)
				}
				choice := action
				if action == "C" && route == "empty" {
					choice = "CC"
				}
				if choice == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				bound, e := m.NonzeroRootLowerBound(v)
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("actual nonzero-bound Throw", e)
					}
				} else if !errors.Is(e, arena.ErrBounds) {
					t.Fatal("continuation did not reach guarded scan read", e)
				}
				if bound != 0 || m.factorTrace != nil {
					t.Fatal("invalid input published bound or retained borrow", bound)
				}
				if route == "all-zero" {
					upolyGCDExpect(t, m, p, []int64{0, 0})
				} else {
					upolyGCDExpect(t, m, p, []int64{1})
				}
				serviceMust(t, m.CleanupKnuthBounds())
			})
		}
	}
}
func TestUPolynomialNonzeroBoundAssertionProfiles(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "enabled", true: "disabled"}[disabled], func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{0, 0, -2, 0, 1})
			polynomialAssertionActionsFixture(t, "")
			if polynomialSourceDebug && disabled {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			bound, e := m.NonzeroRootLowerBound(mustUPolyView(t, m, p))
			serviceMust(t, e)
			if bound != 1 {
				t.Fatal("nonzero roots +-sqrt(2) require actual exponent1", bound)
			}
			upolyGCDExpect(t, m, p, []int64{0, 0, -2, 0, 1})
			serviceMust(t, m.CleanupKnuthBounds())
		})
	}
}
