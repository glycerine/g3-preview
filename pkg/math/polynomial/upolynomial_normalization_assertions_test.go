package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialDefaultNormalizationIdentityActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{3, 4})
			v := mustUPolyView(t, m, p)
			divisor := unumeral.NewInt(2)
			choice := action
			if action == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			// Qualified direct coefficient-helper adapter with independently supplied
			// divisor2: no valid whole-polynomial GCD is claimed, and no scratch is
			// changed after computation or fake enclosing numerical frame constructed.
			e := m.normalizeCoefficient(v, 0, &divisor)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("default normalization Throw", e)
				}
			} else {
				serviceMust(t, e)
			}
			// Actual truncating division precedes DEBUG product and the assertion.
			upolyGCDExpect(t, m, p, []int64{1, 4})
		})
	}
}
func TestUPolynomialDefaultNormalizationRuntimeOracle(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "enabled", true: "disabled"}[disabled], func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{-6, 0, 12})
			polynomialAssertionActionsFixture(t, "")
			if polynomialSourceDebug && disabled {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			serviceMust(t, m.NormalizeView(mustUPolyView(t, m, p)))
			upolyGCDExpect(t, m, p, []int64{-1, 0, 2})
		})
	}
}
func TestUPolynomialNormalizationIdentityClosedAndErased(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	serviceMust(t, m.normalizationCoefficientAssert(nil, nil))
	if !polynomialSourceDebug {
		return
	}
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	one := unumeral.NewInt(1)
	if e := m.normalizationCoefficientAssert(&one, &one); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed identity gate evaluated valid numerals", e)
	}
}
