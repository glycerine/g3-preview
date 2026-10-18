package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"testing"
)

func TestUPolynomialMulAliasAssertionActions(t *testing.T) {
	for _, second := range []bool{false, true} {
		for _, action := range []string{"T", "C", "disabled"} {
			name := "p1/" + action
			if second {
				name = "p2/" + action
			}
			t.Run(name, func(t *testing.T) {
				m, _, budget := upolySourceOwner(t)
				input := upolySourceNew(t, m, []int32{7})
				empty := upolySourceNew(t, m, nil)
				out := upolySourceNew(t, m, []int32{9})
				iv, e := m.wholeView(input)
				serviceMust(t, e)
				serviceMust(t, m.setView(&m.basic, iv))
				a := m.stateView(&m.basic)
				b, e := m.wholeView(empty)
				serviceMust(t, e)
				if second {
					a, b = b, a
				}
				polynomialAssertionActionsFixture(t, action)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				d, e := m.descriptor(out)
				serviceMust(t, e)
				before, used := m.basic, budget.Used()
				body := m.mul(a, b, &d.state)
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(body, ErrUPolynomialAssertion) || !linearThrownAssertion(body) {
						t.Fatal("alias did not reach source Throw", body)
					}
					if m.basic != before || budget.Used() != used {
						t.Fatal("Throw changed multiplication scratch")
					}
					upolyGCDExpect(t, m, out, []int64{9})
				} else {
					serviceMust(t, body)
					upolyGCDExpect(t, m, out, nil)
				}
			})
		}
	}
}

func TestUPolynomialMulDisabledAssertionsSkipOperands(t *testing.T) {
	m, _, budget := upolySourceOwner(t)
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	used := budget.Used()
	budget.FailAfter(0)
	serviceMust(t, m.mulAliasAssertions(UPolynomialView{}, UPolynomialView{}, nil))
	budget.FailAfter(-1)
	if budget.Used() != used {
		t.Fatal("erased alias operands acquired storage")
	}
}

func TestUPolynomialMulContinuesFirstAliasBeforeSecondThrow(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	m, _, budget := upolySourceOwner(t)
	input := upolySourceNew(t, m, []int32{7})
	out := upolySourceNew(t, m, []int32{9})
	v, e := m.wholeView(input)
	serviceMust(t, e)
	serviceMust(t, m.setView(&m.basic, v))
	v = m.stateView(&m.basic)
	d, e := m.descriptor(out)
	serviceMust(t, e)
	polynomialAssertionActionsFixture(t, "CT")
	before, used := m.basic, budget.Used()
	body := m.mul(v, v, &d.state)
	if !errors.Is(body, ErrUPolynomialAssertion) || !linearThrownAssertion(body) {
		t.Fatal("second assertion did not Throw after first Continue", body)
	}
	if m.basic != before || budget.Used() != used {
		t.Fatal("second Throw passed source scratch reset")
	}
	value, e := m.read(v, 0)
	serviceMust(t, e)
	got, e := m.config.Integer.Int64(value)
	serviceMust(t, e)
	if got != 7 {
		t.Fatal("second Throw changed input before body", got)
	}
	upolyGCDExpect(t, m, out, []int64{9})
}
