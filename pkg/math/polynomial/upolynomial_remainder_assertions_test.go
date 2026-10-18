package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"strings"
	"testing"
)

func TestUPolynomialRemainderAliasAssertionActions(t *testing.T) {
	for _, signed := range []bool{false, true} {
		for _, second := range []bool{false, true} {
			for _, action := range []string{"T", "C", "disabled"} {
				name := "rem/p1/" + action
				if signed {
					name = "srem/p1/" + action
				}
				if second {
					name = strings.Replace(name, "p1", "p2", 1)
				}
				t.Run(name, func(t *testing.T) {
					m, _, _ := upolySourceOwner(t)
					a := upolySourceNew(t, m, []int32{3})
					b := upolySourceNew(t, m, []int32{1})
					out := a
					if second {
						out = b
					}
					choices := strings.Repeat(action, 2)
					if action == "disabled" {
						choices = ""
					}
					polynomialAssertionActionsFixture(t, choices)
					if polynomialSourceDebug && action == "disabled" {
						serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
					}
					d := uint32(77)
					var e error
					if signed {
						e = m.SignedRem(a, b, out)
					} else {
						e = m.Rem(a, b, out, &d)
					}
					if polynomialSourceDebug && action == "T" {
						if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
							t.Fatal("alias assertion did not Throw", e)
						}
						if d != 77 {
							t.Fatal("alias Throw passed source d assignment", d)
						}
						want := int64(3)
						if second {
							want = 1
						}
						upolyGCDExpect(t, m, out, []int64{want})
					} else {
						serviceMust(t, e)
						if !signed && d != 0 {
							t.Fatal("Continue skipped source d assignment", d)
						}
						upolyGCDExpect(t, m, out, nil)
					}
				})
			}
		}
	}
}

func TestUPolynomialRemainderSizeAssertionActions(t *testing.T) {
	for _, signed := range []bool{false, true} {
		for _, action := range []string{"T", "C", "disabled"} {
			name := "rem/" + action
			if signed {
				name = "srem/" + action
			}
			t.Run(name, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				a := upolySourceNew(t, m, []int32{3})
				b := upolySourceNew(t, m, nil)
				out := upolySourceNew(t, m, []int32{5})
				choices := action
				if action == "disabled" {
					choices = ""
				}
				polynomialAssertionActionsFixture(t, choices)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				d := uint32(77)
				var e error
				if signed {
					e = m.SignedRem(a, b, out)
				} else {
					e = m.Rem(a, b, out, &d)
				}
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("size assertion did not Throw", e)
					}
					if !signed && d != 0 {
						t.Fatal("size assertion preceded d assignment", d)
					}
					upolyGCDExpect(t, m, out, []int64{5})
				} else {
					serviceMust(t, e)
					want := int64(3)
					if signed {
						want = -3
					}
					upolyGCDExpect(t, m, out, []int64{want})
				}
			})
		}
	}
}

func TestUPolynomialRemainderLeadingAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			a := upolySourceNew(t, m, []int32{1, 3})
			b := upolySourceNew(t, m, []int32{1, 2})
			out := upolySourceNew(t, m, nil)
			bv, e := m.wholeView(b)
			serviceMust(t, e)
			leading, e := m.read(bv, 1)
			serviceMust(t, e)
			// Keep the actual pointer/size shape while qualifying the zero-LC guard.
			serviceMust(t, m.modular.SetInt64(leading, 0))
			m.config.Checkpoint = func() bool { return false }
			m.config.Cooperation = func() error { return nil }
			choices := action
			if action == "disabled" {
				choices = ""
			}
			polynomialAssertionActionsFixture(t, choices)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			d := uint32(77)
			e = m.Rem(a, b, out, &d)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("leading assertion did not Throw", e)
				}
			} else if !errors.Is(e, ErrUPolynomialCanceled) {
				t.Fatal("Continue did not reach actual checkpoint", e)
			}
			if d != 0 {
				t.Fatal("leading assertion lost source d assignment", d)
			}
			upolyGCDExpect(t, m, out, []int64{1, 3})
		})
	}
}

func TestUPolynomialRemainderDisabledAssertionsSkipOperands(t *testing.T) {
	m, _, budget := upolySourceOwner(t)
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	used := budget.Used()
	budget.FailAfter(0)
	for _, line := range []int32{517, 568} {
		serviceMust(t, m.remainderAliasAssertions(UPolynomialView{}, UPolynomialView{}, nil, line))
	}
	serviceMust(t, m.remainderLeadingAssertion(nil))
	budget.FailAfter(-1)
	if used != budget.Used() {
		t.Fatal("disabled remainder assertions acquired storage")
	}
}
