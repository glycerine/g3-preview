package polynomial

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestUPolynomialRemoveOneHalfRootAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{1, 1}) // p(1/2)=3/2, no half root.
			out := upolySourceNew(t, m, []int32{99})
			choice := action
			if choice == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			e := m.RemoveOneHalfRoot(mustUPolyView(t, m, p), out)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("actual Horner assertion Throw", e)
				}
				upolyGCDExpect(t, m, out, []int64{99})
			} else {
				serviceMust(t, e)
				// Literal integer pseudo-division: 2*(x+1)=(2*x-1)+3.
				// The quotient-only source call discards the nonzero remainder.
				upolyGCDExpect(t, m, out, []int64{1})
			}
			wantInvocation := uint64(0)
			if polynomialSourceDebug && action != "disabled" {
				wantInvocation = 1
			}
			if m.oneHalfRoot.invocation != wantInvocation || m.factorTrace != nil {
				t.Fatal("runtime gate did not erase Horner child or release borrow", m.oneHalfRoot.invocation, wantInvocation)
			}
			upolyGCDExpect(t, m, p, []int64{1, 1})
			serviceMust(t, m.CleanupRemoveOneHalfRoot())
		})
	}
}
func TestUPolynomialRemoveOneHalfRootClosedAssertionBeforeHorner(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions compiled out")
	}
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{-1, 2})
	out := upolySourceNew(t, m, []int32{99})
	v := mustUPolyView(t, m, p)
	o, e := m.descriptor(out)
	serviceMust(t, e)
	polynomialAssertionActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	e = m.RemoveOneHalfRoot(v, out)
	if !errors.Is(e, arena.ErrClosed) || m.oneHalfRoot.invocation != 0 || o.state.size != 1 || m.values(&o.state)[0].Val != 99 {
		t.Fatal("closed runtime gate crossed Horner/output frontier", e)
	}
}

func TestUPolynomialRemoveOneHalfRootDisabledPreservesOlderHorner(t *testing.T) {
	reached := false
	for prefix := int64(0); prefix < 64 && !reached; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, p := algebraicFixture(t)
			z := p.config.Integer
			source := algebraicPolynomial(t, p, 1, 2147483647, 1)
			small := algebraicPolynomial(t, p, -1, 2)
			out, e := p.NewPolynomial()
			serviceMust(t, e)
			sv, e := p.wholeView(source)
			serviceMust(t, e)
			v, e := p.wholeView(small)
			serviceMust(t, e)
			z.Budget().FailAfter(prefix)
			_, body := p.HasOneHalfRoot(sv)
			z.Budget().FailAfter(-1)
			h := &p.oneHalfRoot
			old := p.oneHalfRootReceipt()
			reached = body != nil && h.active && polynomialNegNumericPending(&h.numeric)
			if reached {
				before := *h
				polynomialAssertionActionsFixture(t, "")
				if polynomialSourceDebug {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				serviceMust(t, p.RemoveOneHalfRoot(v, out))
				upolyGCDExpect(t, p, out, []int64{2})
				if p.removeOneHalfRoot.assertion.frame != nil || !reflect.DeepEqual(before, *h) {
					t.Fatal("disabled predicate adopted or changed independent stopped Horner")
				}
				serviceMust(t, p.CleanupRemoveOneHalfRoot())
				if !reflect.DeepEqual(before, *h) {
					t.Fatal("owning removal cleanup changed independent Horner")
				}
			}
			serviceMust(t, p.cleanupOneHalfRootWithReceipt(&old, false))
			serviceMust(t, p.DelPolynomial(&out))
			serviceMust(t, p.CleanupOneHalfRoot())
			algebraicCellSourcePhysical(t, c, []UPolynomial{source, small}, nil, nil)
		})
	}
	if !reached {
		t.Fatal("actual independent stopped Horner prefix not reached")
	}
}
