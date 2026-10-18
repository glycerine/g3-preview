package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func gcdAliasCall(m *UPolynomialManager, route string, a, b UPolynomialView, out *upolyVectorState) error {
	switch route {
	case "gcd":
		return m.gcd(a, b, out)
	case "euclid":
		return m.euclidGCD(a, b, out)
	default:
		return m.subresultantGCD(a, b, out)
	}
}

func TestUPolynomialGCDAliasAssertionSourceActions(t *testing.T) {
	for _, route := range []string{"gcd", "euclid", "subresultant"} {
		for _, input := range []string{"first", "second"} {
			for _, action := range []string{"T", "C", "disabled"} {
				t.Run(route+"/"+input+"/"+action, func(t *testing.T) {
					m, _, _ := upolySourceOwner(t)
					nonempty := upolySourceNew(t, m, []int32{-1, -2})
					empty := upolySourceNew(t, m, nil)
					a, b := mustUPolyView(t, m, nonempty), mustUPolyView(t, m, empty)
					if input == "second" {
						a, b = b, a
					}
					o, e := m.descriptor(nonempty)
					serviceMust(t, e)
					choice := action
					if choice == "disabled" {
						choice = ""
					}
					polynomialAssertionActionsFixture(t, choice)
					if polynomialSourceDebug && action == "disabled" {
						serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
					}
					e = gcdAliasCall(m, route, a, b, &o.state)
					if polynomialSourceDebug && action == "T" {
						if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
							t.Fatal("GCD alias debugger Throw", e)
						}
						upolyGCDExpect(t, m, nonempty, []int64{-1, -2})
					} else {
						serviceMust(t, e)
						// Literal zero-input return copies the same base then flips its sign.
						upolyGCDExpect(t, m, nonempty, []int64{1, 2})
					}
					if m.factorTrace != nil || m.gcd1.size != 0 || m.gcd2.size != 0 {
						t.Fatal("alias action crossed empty-return/kernel-copy frontier")
					}
				})
			}
		}
	}
}

func TestUPolynomialGCDSeparateAliasDebuggerActions(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions compiled out")
	}
	for _, route := range []string{"gcd", "euclid", "subresultant"} {
		t.Run(route, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{-1, -2})
			v := mustUPolyView(t, m, p)
			o, e := m.descriptor(p)
			serviceMust(t, e)
			polynomialAssertionActionsFixture(t, "CT")
			e = gcdAliasCall(m, route, v, v, &o.state)
			if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
				t.Fatal("second independent alias assertion did not Throw", e)
			}
			upolyGCDExpect(t, m, p, []int64{-1, -2})
			if m.gcd1.size != 0 || m.gcd2.size != 0 {
				t.Fatal("second assertion crossed first kernel copy")
			}
		})
	}
}

func TestUPolynomialGCDClosedAliasAssertionSwitchBeforeCopy(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions compiled out")
	}
	for _, route := range []string{"gcd", "euclid", "subresultant"} {
		t.Run(route, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			a, b := upolySourceNew(t, m, nil), upolySourceNew(t, m, []int32{-1, -2})
			out := upolySourceNew(t, m, []int32{7})
			av, bv := mustUPolyView(t, m, a), mustUPolyView(t, m, b)
			o, e := m.descriptor(out)
			serviceMust(t, e)
			polynomialAssertionActionsFixture(t, "")
			serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
			e = gcdAliasCall(m, route, av, bv, &o.state)
			if !errors.Is(e, arena.ErrClosed) || o.state.size != 1 || m.values(&o.state)[0].Val != 7 {
				t.Fatal("closed alias gate crossed empty-input copy", e)
			}
		})
	}
}

func TestUPolynomialGCDAliasNonemptyContinuation(t *testing.T) {
	for _, route := range []string{"gcd", "euclid", "subresultant"} {
		for _, input := range []string{"first", "second"} {
			for _, action := range []string{"C", "disabled"} {
				t.Run(route+"/"+input+"/"+action, func(t *testing.T) {
					m, _, _ := upolySourceOwner(t)
					// A=-(x+1)(x+2), B=-(x+1)(x+3); primitive positive GCD=x+1.
					a, b := upolySourceNew(t, m, []int32{-2, -3, -1}), upolySourceNew(t, m, []int32{-3, -4, -1})
					av, bv := mustUPolyView(t, m, a), mustUPolyView(t, m, b)
					out := a
					if input == "second" {
						out = b
					}
					o, e := m.descriptor(out)
					serviceMust(t, e)
					choice := action
					if choice == "disabled" {
						choice = ""
					}
					polynomialAssertionActionsFixture(t, choice)
					if polynomialSourceDebug && action == "disabled" {
						serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
					}
					serviceMust(t, gcdAliasCall(m, route, av, bv, &o.state))
					upolyGCDExpect(t, m, out, []int64{1, 1})
				})
			}
		}
	}
}
