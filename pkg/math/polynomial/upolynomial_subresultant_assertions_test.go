package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// Qualified predicate adapters exercise the exact guards used by the live
// algorithm. They do not claim these failed invariants arise from valid PRS.
func TestUPolynomialSubresultantAssertionActions(t *testing.T) {
	for _, predicate := range []string{"degree", "exponent", "divisibility"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(predicate+"/"+action, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				a, b := upolySourceNew(t, m, []int32{1}), upolySourceNew(t, m, []int32{1, 1})
				ao, e := m.descriptor(a)
				serviceMust(t, e)
				bo, e := m.descriptor(b)
				serviceMust(t, e)
				h, aux := unumeral.NewInt(2), unumeral.NewInt(3)
				choice := action
				if choice == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				switch predicate {
				case "degree":
					e = m.subresultantDegreeAssertion(&ao.state, &bo.state)
				case "exponent":
					e = m.subresultantExponentAssertion(1)
				default:
					e = m.subresultantDivisibilityAssertion(&h, &aux)
				}
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("source debugger Throw", e)
					}
				} else {
					serviceMust(t, e)
				}
				if h != unumeral.NewInt(2) || aux != unumeral.NewInt(3) || m.factorTrace != nil {
					t.Fatal("predicate action changed values or retained borrow")
				}
				upolyGCDExpect(t, m, a, []int64{1})
				upolyGCDExpect(t, m, b, []int64{1, 1})
			})
		}
	}
}
func TestUPolynomialSubresultantAssertionErasureAndClosedGate(t *testing.T) {
	for _, closed := range []bool{false, true} {
		t.Run(map[bool]string{false: "erased", true: "closed"}[closed], func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			polynomialAssertionActionsFixture(t, "")
			if polynomialSourceDebug {
				if closed {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
				} else {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
			}
			// No operand read is authorized before the runtime gate, even for nil
			// query-only adapters. No source scratch or construction counts are set.
			for _, e := range []error{m.subresultantDegreeAssertion(nil, nil), m.subresultantExponentAssertion(0), m.subresultantDivisibilityAssertion(nil, nil)} {
				if polynomialSourceDebug && closed {
					if !errors.Is(e, arena.ErrClosed) {
						t.Fatal("closed gate", e)
					}
				} else {
					serviceMust(t, e)
				}
			}
		})
	}
}
func TestUPolynomialSubresultantDegreeGapAssertionProfiles(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "enabled", true: "disabled"}[disabled], func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			// A=(x+1)(2*x^3+3), B=(x+1)(3*x+2): initial degree gap 2.
			a, b := upolySourceNew(t, m, []int32{3, 3, 0, 2, 2}), upolySourceNew(t, m, []int32{2, 5, 3})
			out := upolySourceNew(t, m, nil)
			polynomialAssertionActionsFixture(t, "")
			if polynomialSourceDebug && disabled {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			serviceMust(t, m.SubresultantGCD(a, b, out))
			upolyGCDExpect(t, m, out, []int64{1, 1})
		})
	}
}
