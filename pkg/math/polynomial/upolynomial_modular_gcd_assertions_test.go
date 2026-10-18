package polynomial

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestUPolynomialModularGCDEntryAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			a := upolySourceNew(t, m, []int32{1})
			b := upolySourceNew(t, m, []int32{1})
			out := upolySourceNew(t, m, []int32{2})
			// A concrete retained raw pointer/count adapter deliberately reaches
			// a guarded second read only after both entry assertions continue.
			av, bv := mustUPolyView(t, m, a).WithCount(2), mustUPolyView(t, m, b)
			o, e := m.descriptor(out)
			serviceMust(t, e)
			serviceMust(t, m.SetZPUint64(7))
			choice := action
			if choice == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			e = m.modularGCD(av, bv, &o.state)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) || !reflect.DeepEqual(upolyDivisionValues(t, m, out), []int64{2}) || m.mgcd[0].size != 0 || m.mgcd[1].size != 0 {
					t.Fatal("modular-mode Throw crossed first scalar/input-copy frontier", e)
				}
			} else {
				if !errors.Is(e, arena.ErrBounds) || linearThrownAssertion(e) || !reflect.DeepEqual(upolyDivisionValues(t, m, out), []int64{2}) {
					t.Fatal("continued entry did not reach actual guarded raw read", e)
				}
			}
			prime, e := m.modular.BorrowedModulus()
			serviceMust(t, e)
			n, e := m.config.Integer.Int64(prime)
			serviceMust(t, e)
			if n != 7 {
				t.Fatal("literal image scope did not restore actual entering modulus", n)
			}
			if m.factorTrace != nil {
				t.Fatal("modular GCD debugger retained borrow")
			}
		})
	}
}

func TestUPolynomialModularGCDEmptyThenModeAssertionOrder(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions compiled out")
	}
	for _, choices := range []string{"T", "CT"} {
		t.Run(choices, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			a := upolySourceNew(t, m, nil)
			b := upolySourceNew(t, m, []int32{1})
			out := upolySourceNew(t, m, []int32{2})
			av, bv := mustUPolyView(t, m, a), mustUPolyView(t, m, b)
			o, e := m.descriptor(out)
			serviceMust(t, e)
			serviceMust(t, m.SetZPUint64(7))
			polynomialAssertionActionsFixture(t, choices)
			e = m.modularGCD(av, bv, &o.state)
			if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) || !reflect.DeepEqual(upolyDivisionValues(t, m, out), []int64{2}) || m.mgcd[0].size != 0 || m.mgcd[1].size != 0 {
				t.Fatal("entry assertions did not stop before scalar declarations/copy", choices, e)
			}
		})
	}
}

func TestUPolynomialModularGCDClosedAssertionSwitchBeforeCopies(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions compiled out")
	}
	m, _, _ := upolySourceOwner(t)
	a := upolySourceNew(t, m, []int32{1})
	b := upolySourceNew(t, m, []int32{1})
	out := upolySourceNew(t, m, []int32{2})
	av, bv := mustUPolyView(t, m, a), mustUPolyView(t, m, b)
	o, e := m.descriptor(out)
	serviceMust(t, e)
	polynomialAssertionActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	e = m.modularGCD(av, bv, &o.state)
	if !errors.Is(e, arena.ErrClosed) || o.state.size != 1 || m.values(&o.state)[0].Val != 2 || m.mgcd[0].size != 0 || m.mgcd[1].size != 0 {
		t.Fatal("closed assertion gate crossed initial modular GCD frontier", e)
	}
}
