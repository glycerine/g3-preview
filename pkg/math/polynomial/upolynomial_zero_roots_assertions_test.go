package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestUPolynomialZeroRootsAssertionActions(t *testing.T) {
	for _, route := range []string{"has-empty", "remove-empty", "all-zero"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(route+"/"+action, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				p := upolySourceNew(t, m, []int32{1})
				out := upolySourceNew(t, m, []int32{7, 8})
				if route == "all-zero" {
					serviceMust(t, m.Reset(p))
					serviceMust(t, m.Reserve(p, 2))
				}
				v := mustUPolyView(t, m, p)
				if route != "all-zero" {
					v = v.WithCount(0)
				}
				choice := action
				if choice == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				var e error
				if route == "has-empty" {
					_, e = m.HasZeroRoots(v)
				} else {
					e = m.RemoveZeroRoots(v, out)
				}
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("actual zero-root debugger Throw", e)
					}
				} else if !errors.Is(e, arena.ErrBounds) {
					t.Fatal("literal continuation did not reach guarded raw read", e)
				}
				upolyGCDExpect(t, m, out, []int64{7, 8})
				if m.factorTrace != nil {
					t.Fatal("zero-root action retained numeric borrow")
				}
			})
		}
	}
}
func TestUPolynomialZeroRootsClosedAssertionBeforeRead(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions compiled out")
	}
	for _, route := range []string{"has", "remove"} {
		t.Run(route, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{1})
			out := upolySourceNew(t, m, []int32{7})
			v := mustUPolyView(t, m, p)
			o, e := m.descriptor(out)
			serviceMust(t, e)
			polynomialAssertionActionsFixture(t, "")
			serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
			if route == "has" {
				_, e = m.HasZeroRoots(v)
			} else {
				e = m.removeZeroRootsState(v, &o.state)
			}
			if !errors.Is(e, arena.ErrClosed) || o.state.size != 1 || m.values(&o.state)[0].Val != 7 {
				t.Fatal("closed valid-predicate gate crossed read/copy", e)
			}
		})
	}
}
func TestUPolynomialZeroRootsValidAssertionProfiles(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "enabled", true: "disabled"}[disabled], func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{0, 0, 6, -3})
			out := upolySourceNew(t, m, nil)
			polynomialAssertionActionsFixture(t, "")
			if polynomialSourceDebug && disabled {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			has, e := m.HasZeroRoots(mustUPolyView(t, m, p))
			serviceMust(t, e)
			if !has {
				t.Fatal("missing actual zero root")
			}
			serviceMust(t, m.RemoveZeroRoots(mustUPolyView(t, m, p), out))
			upolyGCDExpect(t, m, out, []int64{6, -3})
		})
	}
}
