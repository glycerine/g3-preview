package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"testing"
)

func TestUPolynomialDivRemCoreAliasAssertionSourceActions(t *testing.T) {
	for _, route := range []string{"p1-q", "p2-q", "p1-r", "p2-r"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(route+"/"+action, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				a, b := upolySourceNew(t, m, []int32{6}), upolySourceNew(t, m, []int32{2})
				quotient, remainder := upolySourceNew(t, m, []int32{71}), upolySourceNew(t, m, []int32{72})
				switch route {
				case "p1-q":
					quotient = a
				case "p2-q":
					quotient = b
				case "p1-r":
					remainder = a
				case "p2-r":
					remainder = b
				}
				av, bv := mustUPolyView(t, m, a), mustUPolyView(t, m, b)
				q, e := m.descriptor(quotient)
				serviceMust(t, e)
				r, e := m.descriptor(remainder)
				serviceMust(t, e)
				beforeQ, beforeR := upolyDivisionValues(t, m, quotient), upolyDivisionValues(t, m, remainder)
				choice := action
				if action == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				d := uint32(77)
				body := m.divRemCore(av, bv, &d, &q.state, &r.state)
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(body, ErrUPolynomialAssertion) || !linearThrownAssertion(body) || d != 77 || !reflect.DeepEqual(upolyDivisionValues(t, m, quotient), beforeQ) || !reflect.DeepEqual(upolyDivisionValues(t, m, remainder), beforeR) {
						t.Fatal("alias Throw crossed d/output frontier", route, body, d)
					}
				} else {
					serviceMust(t, body)
					// Literal integer-mode constant-divisor branch copies A and clears R.
					if d != 0 || !reflect.DeepEqual(upolyDivisionValues(t, m, quotient), []int64{6}) || len(upolyDivisionValues(t, m, remainder)) != 0 {
						t.Fatal("alias continuation changed source constant branch", route, d)
					}
				}
				if m.factorTrace != nil {
					t.Fatal("alias debugger retained numerical borrow")
				}
			})
		}
	}
}
func TestUPolynomialDivRemCoreDivisorAssertionFrontiers(t *testing.T) {
	for _, route := range []string{"empty", "zero-leading"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(route+"/"+action, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				input := []int32{7}
				if route == "zero-leading" {
					input = []int32{6, 1}
				}
				a := upolySourceNew(t, m, input)
				b := upolySourceNew(t, m, nil)
				if route == "zero-leading" {
					// Actual caller storage constructs the trailing zero numeral. SetInts
					// would normalize it away; no numerical scratch or count is rewritten.
					serviceMust(t, m.SetInts(b, []unumeral.Int{unumeral.NewInt(1)}))
					serviceMust(t, m.Reserve(b, 2))
				}
				quotient, remainder := upolySourceNew(t, m, []int32{71}), upolySourceNew(t, m, []int32{72})
				av, bv := mustUPolyView(t, m, a), mustUPolyView(t, m, b)
				q, e := m.descriptor(quotient)
				serviceMust(t, e)
				r, e := m.descriptor(remainder)
				serviceMust(t, e)
				choice := action
				if action == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				d := uint32(77)
				body := m.divRemCore(av, bv, &d, &q.state, &r.state)
				wantQ, wantR, wantD := []int64{}, []int64{7}, uint32(0)
				if route == "zero-leading" {
					wantQ, wantR, wantD = []int64{1}, []int64{-1}, 1
				}
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(body, ErrUPolynomialAssertion) || !linearThrownAssertion(body) {
						t.Fatal("source divisor Throw", route, body)
					}
					if route == "empty" {
						wantQ, wantR = []int64{71}, []int64{72}
					} else {
						wantQ, wantR = []int64{0}, []int64{6, 1}
					}
					wantD = 0
				} else {
					serviceMust(t, body)
				}
				if d != wantD || !reflect.DeepEqual(upolyDivisionValues(t, m, quotient), wantQ) || !reflect.DeepEqual(upolyDivisionValues(t, m, remainder), wantR) {
					t.Fatal("divisor assertion source publication frontier", route, action, d, upolyDivisionValues(t, m, quotient), upolyDivisionValues(t, m, remainder))
				}
				if m.factorTrace != nil {
					t.Fatal("divisor debugger retained numerical borrow")
				}
			})
		}
	}
}
func TestUPolynomialDivRemCoreClosedAssertionSwitchBeforeD(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions compiled out")
	}
	m, _, _ := upolySourceOwner(t)
	a, b := upolySourceNew(t, m, []int32{6}), upolySourceNew(t, m, []int32{2})
	quotient, remainder := upolySourceNew(t, m, []int32{71}), upolySourceNew(t, m, []int32{72})
	av, bv := mustUPolyView(t, m, a), mustUPolyView(t, m, b)
	q, e := m.descriptor(quotient)
	serviceMust(t, e)
	r, e := m.descriptor(remainder)
	serviceMust(t, e)
	polynomialAssertionActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	d := uint32(77)
	body := m.divRemCore(av, bv, &d, &q.state, &r.state)
	if !errors.Is(body, arena.ErrClosed) || d != 77 || m.values(&q.state)[0].Val != 71 || m.values(&r.state)[0].Val != 72 {
		t.Fatal("closed runtime switch crossed initial kernel frontier", body, d)
	}
}
