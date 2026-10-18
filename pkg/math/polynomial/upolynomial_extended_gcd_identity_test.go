package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestUPolynomialExtendedGCDIdentitySourceActions(t *testing.T) {
	for _, caller := range []string{"public", "hensel"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(caller+"/"+action, func(t *testing.T) {
				var m *UPolynomialManager
				var c *upolyHenselCallerScope
				if caller == "public" {
					m, _, _ = upolySourceOwner(t)
				} else {
					_, s := preparationPhysicalFixture(t)
					parent, e := s.newSquareFreeScope()
					serviceMust(t, e)
					serviceMust(t, s.beginHenselCaller(parent))
					c = &parent.hensel
					c.related, e = NewRelatedUPolynomialManager(s.manager)
					serviceMust(t, e)
					m = c.related
					// Concrete direct retained-caller adapter; the parent has no invented
					// source declarations. Outputs below have independent registered owners.
					t.Cleanup(func() { c.executing = false; serviceMust(t, c.own(false)); serviceMust(t, s.CleanupRetained()) })
				}
				serviceMust(t, m.SetZPUint64(7))
				a, b := upolySourceNew(t, m, []int32{2, 2}), upolySourceNew(t, m, []int32{2, 1})
				u, v, d := upolySourceNew(t, m, nil), upolySourceNew(t, m, nil), upolySourceNew(t, m, nil)
				choice := action
				if choice == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				var e error
				if c == nil {
					e = m.ExtendedGCD(a, b, u, v, d)
				} else {
					av, bv := mustUPolyView(t, m, a), mustUPolyView(t, m, b)
					us, x := m.descriptor(u)
					serviceMust(t, x)
					vs, x := m.descriptor(v)
					serviceMust(t, x)
					ds, x := m.descriptor(d)
					serviceMust(t, x)
					e = m.extendedGCDWithCaller(av, bv, &us.state, &vs.state, &ds.state, c)
				}
				wantU, wantV, wantD := []int64{-1}, []int64{2}, []int64{1}
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("actual identity Throw", e)
					}
					wantU, wantV, wantD = []int64{1}, []int64{-2}, []int64{-1}
					if c != nil && c.extendedScopedConstructed != 0 {
						t.Fatal("Throw reached final scaling NUM declarations")
					}
				} else {
					serviceMust(t, e)
				}
				upolyGCDExpect(t, m, u, wantU)
				upolyGCDExpect(t, m, v, wantV)
				upolyGCDExpect(t, m, d, wantD)
				// This preserves the source nonmonic quirk. Continue publishes A*U+B*V=2
				// although D=1; do not repair source numerals to assert a false identity.
				if m.factorTrace != nil {
					t.Fatal("identity action retained owner borrow")
				}
				if c != nil && polynomialSourceDebug && (c.extendedDebugConstructed != 2 || c.extendedDebugSource != 0) {
					t.Fatal("actual DEBUG vectors did not finish their source suffix")
				}
			})
		}
	}
}
func TestUPolynomialExtendedGCDIdentityGateErasure(t *testing.T) {
	for _, closed := range []bool{false, true} {
		t.Run(map[bool]string{false: "disabled", true: "closed"}[closed], func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			polynomialAssertionActionsFixture(t, "")
			if polynomialSourceDebug {
				if closed {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
				} else {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
			}
			e := m.extendedGCDIdentityAssertion(nil, nil)
			if polynomialSourceDebug && closed {
				if !errors.Is(e, arena.ErrClosed) {
					t.Fatal(e)
				}
			} else {
				serviceMust(t, e)
			}
		})
	}
}
