package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialPrimitiveContentAssertionActions(t *testing.T) {
	for _, nonempty := range []bool{false, true} {
		for _, action := range []string{"T", "CT", "CC", "disabled"} {
			t.Run(map[bool]string{false: "empty", true: "zero"}[nonempty]+"/"+action, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				p, out := upolySourceNew(t, m, nil), upolySourceNew(t, m, []int32{99})
				if nonempty {
					serviceMust(t, m.Reserve(p, 3))
				}
				cont := unumeral.NewInt(77)
				choice := action
				if action == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				e := m.PrimitiveAndContent(p, out, &cont)
				throws := polynomialSourceDebug && (action == "T" || (!nonempty && action == "CT"))
				if throws {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("primitive content Throw", e)
					}
					want := int32(0)
					if !nonempty && action == "T" {
						want = 77
					}
					if cont != unumeral.NewInt(want) {
						t.Fatal("content publication before failed assertion", cont, want)
					}
					upolyGCDExpect(t, m, out, []int64{99})
				} else {
					serviceMust(t, e)
					if cont != unumeral.NewInt(0) {
						t.Fatal("source zero content", cont)
					}
					// Continue skips division for actual zero coefficients, then trims all.
					upolyGCDExpect(t, m, out, nil)
				}
			})
		}
	}
}
func TestUPolynomialPrimitiveContentNonzeroOracle(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "enabled", true: "disabled"}[disabled], func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p, out := upolySourceNew(t, m, []int32{-6, 0, 12}), upolySourceNew(t, m, nil)
			polynomialAssertionActionsFixture(t, "")
			if polynomialSourceDebug && disabled {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			cont := unumeral.NewInt(77)
			serviceMust(t, m.PrimitiveAndContent(p, out, &cont))
			if cont != unumeral.NewInt(6) {
				t.Fatal(cont)
			}
			upolyGCDExpect(t, m, out, []int64{-1, 0, 2})
		})
	}
}
func TestUPolynomialPrimitiveContentClosedGate(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	m, _, _ := upolySourceOwner(t)
	p, out := upolySourceNew(t, m, []int32{2, 4}), upolySourceNew(t, m, []int32{99})
	d, e := m.descriptor(out)
	serviceMust(t, e)
	cont := unumeral.NewInt(77)
	polynomialAssertionActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	e = m.PrimitiveAndContent(p, out, &cont)
	if !errors.Is(e, arena.ErrClosed) || cont != unumeral.NewInt(77) || d.state.size != 1 || m.values(&d.state)[0] != unumeral.NewInt(99) {
		t.Fatal("closed gate crossed content/output mutation", e)
	}
}
