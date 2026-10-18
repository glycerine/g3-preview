package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialTransformInvariantAssertionActions(t *testing.T) {
	for _, route := range []string{"reciprocal", "binary"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(route+"/"+action, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				choice := action
				if choice == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				// Qualified direct invariant adapters. Neither failed counter is reached
				// by valid transform inputs; no numerical scratch is fabricated to do so.
				var e error
				if route == "reciprocal" {
					e = m.reciprocalIndexAssertion(0)
				} else {
					e = m.binaryCompositionExponentAssertion(1)
				}
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("source debugger Throw", e)
					}
				} else {
					serviceMust(t, e)
				}
				if m.factorTrace != nil {
					t.Fatal("transform assertion retained borrow")
				}
			})
		}
	}
}
func TestUPolynomialTransformAssertionProfiles(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "enabled", true: "disabled"}[disabled], func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{1, 2, 3})
			polynomialAssertionActionsFixture(t, "")
			if polynomialSourceDebug && disabled {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			v := mustUPolyView(t, m, p)
			serviceMust(t, m.ReciprocalViewInPlace(v))
			upolyGCDExpect(t, m, p, []int64{3, 2, 1})
			// 4*p(x/2) for p=3+2*x+x^2 gives 12+4*x+x^2.
			b := unumeral.NewBinaryIntPower(1, 1)
			serviceMust(t, m.ComposeBinaryXViewInPlace(v, &b))
			upolyGCDExpect(t, m, p, []int64{12, 4, 1})
		})
	}
}
func TestUPolynomialReciprocalClosedGateBeforeSwap(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions compiled out")
	}
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{1, 2, 3})
	one := upolySourceNew(t, m, []int32{7})
	v, sv := mustUPolyView(t, m, p), mustUPolyView(t, m, one)
	o, e := m.descriptor(p)
	serviceMust(t, e)
	polynomialAssertionActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	e = m.ReciprocalViewInPlace(v)
	if !errors.Is(e, arena.ErrClosed) || o.state.size != 3 {
		t.Fatal("closed reciprocal gate", e)
	}
	for i, n := range []int32{1, 2, 3} {
		if m.values(&o.state)[i].Val != n {
			t.Fatal("closed assertion gate crossed first swap")
		}
	}
	// Literal constant early returns precede both assertions and operand reads.
	serviceMust(t, m.ReciprocalViewInPlace(sv))
	serviceMust(t, m.ComposeBinaryXViewInPlace(sv, nil))
	if e = m.binaryCompositionExponentAssertion(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed final exponent gate", e)
	}
}
