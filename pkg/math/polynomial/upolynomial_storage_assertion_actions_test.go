package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestUPolynomialSelfCopyAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{2, 3, 1})
			v := mustUPolyView(t, m, p).WithCount(1)
			choice := action
			if action == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			e := m.SetView(p, v)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("same-base Throw", e)
				}
			} else {
				serviceMust(t, e)
			}
			// Same-base source return precedes resize/copy, even when Continue accepts
			// the mismatched raw count. No caller data is normalized or truncated.
			upolyGCDExpect(t, m, p, []int64{2, 3, 1})
		})
	}
}
func TestUPolynomialSetSizeAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{7})
			serviceMust(t, m.Reserve(p, 3))
			d, e := m.descriptor(p)
			serviceMust(t, e)
			serviceMust(t, m.trim(&d.state))
			// Reserve constructed real zero entries; trimming removed that tail from
			// logical size. The assertion rejects growing size within this backing.
			choice := action
			if action == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			e = m.setSize(&d.state, 3)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("size Throw", e)
				}
			} else {
				serviceMust(t, e)
			}
			// Continue publishes the requested size then trims its actual zero tail.
			upolyGCDExpect(t, m, p, []int64{7})
		})
	}
}

func TestUPolynomialStorageClosedAssertionGate(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	for _, self := range []bool{false, true} {
		t.Run(map[bool]string{false: "size", true: "self"}[self], func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{7, 8})
			v := mustUPolyView(t, m, p)
			d, e := m.descriptor(p)
			serviceMust(t, e)
			polynomialAssertionActionsFixture(t, "")
			serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
			if self {
				e = m.SetView(p, v)
			} else {
				e = m.setSize(&d.state, 0)
			}
			if !errors.Is(e, arena.ErrClosed) {
				t.Fatal("closed assertion gate", e)
			}
			if d.state.size != 2 || m.values(&d.state)[0] != unumeral.NewInt(7) || m.values(&d.state)[1] != unumeral.NewInt(8) {
				t.Fatal("closed gate crossed actual source mutation")
			}
		})
	}
}
