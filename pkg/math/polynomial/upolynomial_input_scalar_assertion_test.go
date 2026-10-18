package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialRationalInputAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "CT", "CC", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, q, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{99})
			values := [3]unumeral.Rational{unumeral.NewRational(), unumeral.NewRational(), unumeral.NewRational()}
			serviceMust(t, q.SetInt64(&values[0], 7))
			serviceMust(t, q.SetFraction(&values[1], 3, 2))
			serviceMust(t, q.SetFraction(&values[2], 5, 3))
			t.Cleanup(func() {
				for i := range values {
					serviceMust(t, q.Del(&values[i]))
				}
			})
			choice := action
			if action == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			e := m.SetRationals(p, values[:])
			throws := polynomialSourceDebug && (action == "T" || action == "CT")
			if throws {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("integrality Throw", e)
				}
			} else {
				serviceMust(t, e)
			}
			want := []int64{7, 3, 5}
			if polynomialSourceDebug && action == "T" {
				want = []int64{7, 0, 0}
			} else if polynomialSourceDebug && action == "CT" {
				want = []int64{7, 3, 0}
			}
			// reserve publishes all three entries before any integrality assertion;
			// source Continue copies numerators without dividing or normalizing rationals.
			upolyGCDExpect(t, m, p, want)
		})
	}
}
func TestUPolynomialRawScalarZeroAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{3, 4})
			v := mustUPolyView(t, m, p)
			choice := action
			if action == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			zero := unumeral.NewInt(0)
			e := m.MulScalarView(v, &zero)
			want := []int64{0, 0}
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("raw multiplier Throw", e)
				}
				want = []int64{3, 4}
			} else {
				serviceMust(t, e)
			}
			upolyGCDExpect(t, m, p, want)
			// The vector overload has its own zero early return before this raw assertion.
			serviceMust(t, m.MulScalar(p, &zero))
			upolyGCDExpect(t, m, p, nil)
		})
	}
}
