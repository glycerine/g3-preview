package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestUPolynomialSquareFreeAliasAssertionActions(t *testing.T) {
	for _, route := range []string{"constant", "repeated"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(route+"/"+action, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				input := []int32{7}
				want := []int64{7}
				if route == "repeated" {
					input = []int32{1, -2, 1}
					want = []int64{-1, 1}
				}
				p := upolySourceNew(t, m, input)
				choice := action
				if choice == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				e := m.SquareFree(p, p)
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) || m.sqf1.size != 0 || m.sqf2.size != 0 {
						t.Fatal("square-free alias Throw crossed derivative/GCD frontier", e)
					}
					want = make([]int64, len(input))
					for i, n := range input {
						want[i] = int64(n)
					}
				} else {
					serviceMust(t, e)
				}
				upolyGCDExpect(t, m, p, want)
				if m.factorTrace != nil {
					t.Fatal("square-free debugger retained numeric borrow")
				}
			})
		}
	}
}

func TestUPolynomialZeroPowerAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			zero := upolySourceNew(t, m, nil)
			out := upolySourceNew(t, m, []int32{7, 8})
			choice := action
			if choice == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			e := m.Power(zero, 0, out)
			want := []int64{1}
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("zero-power debugger Throw", e)
				}
				want = []int64{7, 8}
			} else {
				serviceMust(t, e)
			}
			upolyGCDExpect(t, m, out, want)
			if m.pw.size != 0 || m.factorTrace != nil {
				t.Fatal("zero-power assertion entered iterative kernel or retained borrow")
			}
		})
	}
}

func TestUPolynomialSquareFreePowerClosedAssertionBeforeWrites(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions compiled out")
	}
	for _, route := range []string{"square-free", "power"} {
		t.Run(route, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{1})
			out := upolySourceNew(t, m, []int32{7, 8})
			v := mustUPolyView(t, m, p)
			o, e := m.descriptor(out)
			serviceMust(t, e)
			polynomialAssertionActionsFixture(t, "")
			serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
			if route == "square-free" {
				e = m.squareFree(v, &o.state)
			} else {
				e = m.powerState(v, 0, &o.state)
			}
			if !errors.Is(e, arena.ErrClosed) || o.state.size != 2 || m.values(&o.state)[0].Val != 7 || m.values(&o.state)[1].Val != 8 || m.sqf1.size != 0 || m.sqf2.size != 0 {
				t.Fatal("closed valid-predicate gate crossed output/derivative frontier", e)
			}
		})
	}
}
