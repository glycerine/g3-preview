package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestUPolynomialDisplaySMT2IndexAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{7})
			// Public Reserve creates a real initialized trailing zero. SetInts would
			// trim it, hiding the source's single-constant/multiple-slot assertion.
			serviceMust(t, m.Reserve(p, 2))
			v := mustUPolyView(t, m, p)
			choice := action
			if choice == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			out := &upolyDisplaySink{}
			e := m.DisplaySMT2Into(v, out, "x")
			want := "7(+ 7)"
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("actual SMT2 index Throw", e)
				}
				want = ""
			} else {
				serviceMust(t, e)
			}
			if out.text.String() != want {
				t.Fatal("source monomial/fallthrough frontier", out.text.String(), want)
			}
			upolyGCDExpect(t, m, p, []int64{7, 0})
			if m.currentDisplay != nil || m.display.active || m.factorTrace != nil {
				t.Fatal("display assertion retained completed source scope/borrow")
			}
		})
	}
}

func TestUPolynomialDisplayExponentAssertionProfiles(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "enabled", true: "disabled"}[disabled], func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{1, -2, 3})
			polynomialAssertionActionsFixture(t, "")
			if polynomialSourceDebug && disabled {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			out := &upolyDisplaySink{}
			serviceMust(t, m.DisplayInto(mustUPolyView(t, m, p), out, "y", true))
			if out.text.String() != "3*y^2 - 2*y + 1" {
				t.Fatal(out.text.String())
			}
		})
	}
}
