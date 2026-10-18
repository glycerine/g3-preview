package polynomial

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialConvertFinalAssertionActions(t *testing.T) {
	for _, line := range []int32{2864, 2892, 2893, 2912, 2913} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(fmt.Sprint(line)+action, func(t *testing.T) {
				m, q, budget := upolySourceOwner(t)
				bqm := upolyIsolationBinaryOwner(t, q, budget)
				p := upolySourceNew(t, m, []int32{-1, 1})
				v := mustUPolyView(t, m, p)
				c, d := unumeral.NewBinaryIntPower(2, 0), unumeral.NewBinaryIntPower(1, 0)
				if line == 2913 {
					d = unumeral.NewBinaryIntPower(3, 0)
				}
				t.Cleanup(func() { serviceMust(t, bqm.Del(&c)); serviceMust(t, bqm.Del(&d)) })
				choice := action
				if action == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				// Independent predicate arguments; real evaluations at2,3 have same sign.
				var e error
				switch line {
				case 2864:
					e = m.convertUpperSignAssertion(1, -1)
				case 2913:
					e = m.convertFinalSignAssertion(v, &c, &d)
				default:
					e = m.convertBinaryOrderAssertion(line, bqm, &c, &d)
				}
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("conversion final Throw", e)
					}
				} else {
					serviceMust(t, e)
				}
				if upolyIsolationBinaryRat(t, m, bqm, &c).RatString() != "2" {
					t.Fatal("predicate changed endpoint")
				}
			})
		}
	}
}
func TestUPolynomialConvertFinalPredicateErasure(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	serviceMust(t, m.convertFinalSignAssertion(UPolynomialView{}, nil, nil))
	for _, line := range []int32{2892, 2893, 2912} {
		serviceMust(t, m.convertBinaryOrderAssertion(line, nil, nil, nil))
	}
}

func TestUPolynomialConvertFinalAliasedSourceActions(t *testing.T) {
	for _, action := range []string{"T", "CT", "CC", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, q, budget := upolySourceOwner(t)
			bqm := upolyIsolationBinaryOwner(t, q, budget)
			p := upolySourceNew(t, m, []int32{-1, 1})
			a, b := unumeral.NewRational(), unumeral.NewRational()
			serviceMust(t, q.SetInt64(&b, 2))
			out := unumeral.NewBinaryInt(37)
			t.Cleanup(func() { serviceMust(t, q.Del(&a)); serviceMust(t, q.Del(&b)); serviceMust(t, bqm.Del(&out)) })
			choice := action
			if action == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			ok, e := m.ConvertRationalToBinaryInterval(mustUPolyView(t, m, p), q, &a, &b, bqm, &out, &out)
			if polynomialSourceDebug && (action == "T" || action == "CT") {
				if ok || !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("actual final alias actions", ok, e)
				}
			} else {
				serviceMust(t, e)
				if !ok {
					t.Fatal("literal alias Continue return")
				}
			}
			// Source swaps publish the second endpoint before final assertions. A true
			// Continue result on this aliased output is not claimed as a valid interval.
			if upolyIsolationBinaryRat(t, m, bqm, &out).RatString() != "2" {
				t.Fatal("final action lost preceding output publication")
			}
		})
	}
}
