package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialBerlekampGetRuntimeAssertionActions(t *testing.T) {
	for _, constant := range []bool{false, true} {
		for _, action := range []string{"T", "C", "disabled"} {
			name := "mutable/" + action
			if constant {
				name = "const/" + action
			}
			t.Run(name, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				three := unumeral.NewInt(3)
				serviceMust(t, m.SetZP(&three))
				f := upolySourceNew(t, m, []int32{1, 0, 1})
				q := upolyBerlekampMatrix{m: m}
				t.Cleanup(q.close)
				serviceMust(t, q.init(m, mustUPolyView(t, m, f)))
				// A caller index crosses into the next real constructed row. Continue
				// reaches that same physical numeral; no matrix ownership/count is changed.
				want, e := m.read(m.stateView(&q.values), 2)
				serviceMust(t, e)
				choice := action
				if action == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				var got *unumeral.Int
				if constant {
					got, e = q.getConst(0, 2)
				} else {
					got, e = q.get(0, 2)
				}
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) || got != nil {
						t.Fatal("source Throw did not precede physical read", got, e)
					}
				} else {
					serviceMust(t, e)
					if got != want {
						t.Fatal("Continue/disabled/release changed reached physical field")
					}
				}
				if q.displaying || q.traceBorrow.executing || q.traceBorrow.bound != 0 {
					t.Fatal("assertion left borrowed owners bound")
				}
			})
		}
	}
}
func TestUPolynomialBerlekampGetClosedRuntimeSwitchPrecedesRead(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	m, _, _ := upolySourceOwner(t)
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	f := upolySourceNew(t, m, []int32{1, 0, 1})
	q := upolyBerlekampMatrix{m: m}
	t.Cleanup(q.close)
	serviceMust(t, q.init(m, mustUPolyView(t, m, f)))
	polynomialAssertionActionsFixture(t, "")
	debug := diagnostic.BorrowProcessDebugContext()
	serviceMust(t, debug.Close())
	for _, constant := range []bool{false, true} {
		var value *unumeral.Int
		var e error
		if constant {
			value, e = q.getConst(0, 0)
		} else {
			value, e = q.get(0, 0)
		}
		if value != nil || !errors.Is(e, arena.ErrClosed) {
			t.Fatal("valid predicate bypassed closed switch owner", value, e)
		}
	}
}
