package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestUPolynomialExtendedGCDEntryAliasActions(t *testing.T) {
	for input := 0; input < 2; input++ {
		for output := 0; output < 3; output++ {
			for _, action := range []string{"T", "C", "disabled"} {
				t.Run(fmt.Sprintf("%d-%d/%s", input, output, action), func(t *testing.T) {
					m, _, _ := upolySourceOwner(t)
					serviceMust(t, m.SetZPUint64(7))
					a, b := upolySourceNew(t, m, []int32{1}), upolySourceNew(t, m, []int32{1})
					outs := [3]UPolynomial{upolySourceNew(t, m, []int32{2}), upolySourceNew(t, m, []int32{3}), upolySourceNew(t, m, []int32{-2})}
					outs[output] = a
					if input == 1 {
						outs[output] = b
					}
					av, bv := mustUPolyView(t, m, a), mustUPolyView(t, m, b)
					var states [3]*upolyVectorState
					for i, p := range outs {
						o, e := m.descriptor(p)
						serviceMust(t, e)
						states[i] = &o.state
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
					if polynomialSourceDebug && action == "T" {
						// Public Throw must stop before U reset and any lexical-vector body.
						e = m.ExtendedGCD(a, b, outs[0], outs[1], outs[2])
						if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
							t.Fatal("public alias Throw", e)
						}
					} else {
						// Qualified entry-only continuation: aliasing may change later source
						// inputs. Do not claim a valid Bezout result for that invalid precondition.
						serviceMust(t, m.extendedGCDEntryAssertions(av, bv, states[0], states[1], states[2]))
					}
					want := [3]int64{2, 3, -2}
					want[output] = 1
					for i, p := range outs {
						upolyGCDExpect(t, m, p, []int64{want[i]})
					}
					if m.factorTrace != nil {
						t.Fatal("entry action retained owner borrow")
					}
				})
			}
		}
	}
}
func TestUPolynomialExtendedGCDEntrySeparateAliases(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions compiled out")
	}
	m, _, _ := upolySourceOwner(t)
	serviceMust(t, m.SetZPUint64(7))
	a, b, d := upolySourceNew(t, m, []int32{1}), upolySourceNew(t, m, []int32{1}), upolySourceNew(t, m, []int32{2})
	polynomialAssertionActionsFixture(t, "CT")
	e := m.ExtendedGCD(a, b, a, a, d)
	if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
		t.Fatal("second alias did not Throw after first Continue", e)
	}
	upolyGCDExpect(t, m, a, []int64{1})
	upolyGCDExpect(t, m, d, []int64{2})
}
func TestUPolynomialExtendedGCDFieldAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			a, b := upolySourceNew(t, m, []int32{1}), upolySourceNew(t, m, []int32{1})
			u, v, d := upolySourceNew(t, m, []int32{2}), upolySourceNew(t, m, []int32{3}), upolySourceNew(t, m, []int32{4})
			choice := action
			if choice == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			e := m.ExtendedGCD(a, b, u, v, d)
			wantU, wantV, wantD := []int64{}, []int64{1}, []int64{1}
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("field Throw", e)
				}
				wantU, wantV, wantD = []int64{2}, []int64{3}, []int64{4}
			} else {
				serviceMust(t, e)
			}
			// Constant integer continuation has the independent Bezout identity 0+1=1.
			upolyGCDExpect(t, m, u, wantU)
			upolyGCDExpect(t, m, v, wantV)
			upolyGCDExpect(t, m, d, wantD)
		})
	}
}
func TestUPolynomialExtendedGCDClosedEntryGate(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions compiled out")
	}
	m, _, _ := upolySourceOwner(t)
	a, b := upolySourceNew(t, m, []int32{1}), upolySourceNew(t, m, []int32{1})
	u, v, d := upolySourceNew(t, m, []int32{2}), upolySourceNew(t, m, []int32{3}), upolySourceNew(t, m, []int32{4})
	av, bv := mustUPolyView(t, m, a), mustUPolyView(t, m, b)
	states := [3]*upolyVectorState{}
	for i, p := range []UPolynomial{u, v, d} {
		o, e := m.descriptor(p)
		serviceMust(t, e)
		states[i] = &o.state
	}
	polynomialAssertionActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	e := m.extendedGCD(av, bv, states[0], states[1], states[2])
	if !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed entry gate", e)
	}
	for i, s := range states {
		if s.size != 1 || m.values(s)[0].Val != int32(i+2) {
			t.Fatal("closed gate crossed U/reset or output writes")
		}
	}
}
