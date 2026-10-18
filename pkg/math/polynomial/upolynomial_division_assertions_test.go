package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialScalarDivisionAssertionCoefficientFrontiers(t *testing.T) {
	for _, action := range []string{"T", "CT", "CC", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{3, 5})
			two := unumeral.NewInt(2)
			choice := action
			if action == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			e := m.DivScalar(p, &two)
			want := []int64{1, 2}
			if polynomialSourceDebug && (action == "T" || action == "CT") {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("source scalar Throw", e)
				}
				want = []int64{3, 5}
				if action == "CT" {
					want = []int64{1, 5}
				}
			} else {
				serviceMust(t, e)
			}
			for i, expected := range want {
				v, e := m.read(mustUPolyView(t, m, p), uint32(i))
				serviceMust(t, e)
				actual, e := m.config.Integer.Int64(v)
				serviceMust(t, e)
				if actual != expected {
					t.Fatal("source per-coefficient publication", action, i, actual, expected)
				}
			}
			if m.factorTrace != nil {
				t.Fatal("scalar assertion retained manager borrow")
			}
		})
	}
}
func TestUPolynomialScalarDivisionZeroThrowAndClosedRuntimeSwitch(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{2})
	zero, two := unumeral.NewInt(0), unumeral.NewInt(2)
	polynomialAssertionActionsFixture(t, "T")
	e := m.DivScalar(p, &zero)
	if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
		t.Fatal("source zero divisor Throw", e)
	}
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	e = m.DivScalar(p, &two)
	if !errors.Is(e, arena.ErrClosed) {
		t.Fatal("valid divisor bypassed closed assertion switch", e)
	}
	v, e := m.read(mustUPolyView(t, m, p), 0)
	serviceMust(t, e)
	n, e := m.config.Integer.Int64(v)
	serviceMust(t, e)
	if n != 2 {
		t.Fatal("assertion failure changed coefficient", n)
	}
}
