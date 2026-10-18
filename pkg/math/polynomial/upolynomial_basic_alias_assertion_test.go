package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

// A public Neg swaps the original p backing into the real private basic buffer.
// The previously issued view remains bound to that actual backing.
func basicAliasFixture(t *testing.T) (*UPolynomialManager, UPolynomialView, UPolynomialView, UPolynomial) {
	m, _, _ := upolySourceOwner(t)
	p, q, out := upolySourceNew(t, m, []int32{1, 2}), upolySourceNew(t, m, []int32{3, 4}), upolySourceNew(t, m, []int32{99})
	captured := mustUPolyView(t, m, p)
	serviceMust(t, m.Neg(q, p))
	if !m.alias(captured, &m.basic) {
		t.Fatal("public swap did not retain actual basic alias")
	}
	return m, captured, mustUPolyView(t, m, q), out
}
func TestUPolynomialBasicAliasAssertionActions(t *testing.T) {
	for _, route := range []string{"neg", "add-first", "add-second", "sub-first", "sub-second", "add-both", "sub-both"} {
		for _, action := range []string{"T", "C", "CT", "CC", "disabled"} {
			t.Run(route+"/"+action, func(t *testing.T) {
				m, alias, other, out := basicAliasFixture(t)
				both := route == "add-both" || route == "sub-both"
				choice := action
				if both && action == "C" {
					choice = "CC"
				}
				if action == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				var e error
				if route == "neg" {
					e = m.NegView(alias, out)
				} else {
					a, b := alias, other
					if route == "add-second" || route == "sub-second" {
						a, b = b, a
					}
					if route == "add-both" || route == "sub-both" {
						b = alias
					}
					e = m.addSubViews(a, b, out, route[:3] == "sub")
				}

				throws := polynomialSourceDebug && (action == "T" || both && action == "CT")
				if throws {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("basic alias Throw", e)
					}
					upolyGCDExpect(t, m, out, []int64{99})
					if m.basic.size != 2 || m.values(&m.basic)[0] != unumeral.NewInt(1) || m.values(&m.basic)[1] != unumeral.NewInt(2) {
						t.Fatal("Throw crossed basic mutation")
					}
					return
				}
				serviceMust(t, e)
				want := []int64{4, 6}
				switch route {
				case "neg":
					want = []int64{-1, -2}
				case "sub-first":
					want = []int64{-2, -2}
				case "sub-second":
					want = []int64{2, 2}
				case "add-both":
					want = []int64{2, 4}
				case "sub-both":
					want = nil
				}
				upolyGCDExpect(t, m, out, want)
			})
		}
	}
}

func TestUPolynomialBasicAliasClosedGate(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	for _, route := range []string{"neg", "add", "sub"} {
		t.Run(route, func(t *testing.T) {
			m, _, other, out := basicAliasFixture(t)
			d, e := m.descriptor(out)
			serviceMust(t, e)
			polynomialAssertionActionsFixture(t, "")
			serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
			if route == "neg" {
				e = m.NegView(other, out)
			} else {
				e = m.addSubViews(other, other, out, route == "sub")
			}
			if !errors.Is(e, arena.ErrClosed) || d.state.size != 1 || m.values(&d.state)[0] != unumeral.NewInt(99) || m.basic.size != 2 || m.values(&m.basic)[0] != unumeral.NewInt(1) || m.values(&m.basic)[1] != unumeral.NewInt(2) {
				t.Fatal("closed gate crossed basic reservation/mutation", e)
			}
		})
	}
}
