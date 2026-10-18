package polynomial

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialRefinableSignInvariantActions(t *testing.T) {
	for _, line := range []int32{2605, 2619, 2624, 2643, 2652, 2661, 2727, 2734} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(fmt.Sprint(line)+"/"+action, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				choice := action
				if action == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				// Qualified direct invariant adapter; no fabricated evaluator result or
				// rewritten algorithm scratch. Several predicates follow from branch guards.
				e := m.refinableSignAssertion(line, 2, 1)
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("sign invariant Throw", e)
					}
				} else {
					serviceMust(t, e)
				}
			})
		}
	}
}
func TestUPolynomialRefinableSameSignSourceActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, q, budget := upolySourceOwner(t)
			bqm := upolyIsolationBinaryOwner(t, q, budget)
			p := upolySourceNew(t, m, []int32{-1, 0, 1})
			v := mustUPolyView(t, m, p)
			a, b := unumeral.NewBinaryIntPower(2, 0), unumeral.NewBinaryIntPower(3, 0)
			t.Cleanup(func() { serviceMust(t, bqm.Del(&a)); serviceMust(t, bqm.Del(&b)) })
			choice := action
			if action == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			ok, e := m.IsolatingToRefinable(v, bqm, &a, &b)
			if polynomialSourceDebug && action == "T" {
				if ok || !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("actual same-sign Throw", ok, e)
				}
			} else {
				serviceMust(t, e)
				if !ok {
					t.Fatal("literal same-sign Continue return")
				}
			}
			// This interval violates the isolating precondition. Continue's true result
			// is source behavior, not a claim that [2,3] brackets a root of x^2-1.
			if upolyIsolationBinaryRat(t, m, bqm, &a).RatString() != "2" || upolyIsolationBinaryRat(t, m, bqm, &b).RatString() != "3" {
				t.Fatal("entry sign action changed endpoints")
			}
		})
	}
}
func TestUPolynomialRefinableSignRuntimeOracles(t *testing.T) {
	for _, f := range []struct {
		name   string
		coeff  []int32
		a, b   int32
		ok     bool
		wa, wb string
	}{
		{"bracket", []int32{-1, 0, 1}, 0, 2, true, "0", "2"},
		{"left-endpoint", []int32{0, -1, 1}, 0, 3, true, "3/4", "3/2"},
		{"right-endpoint", []int32{0, -1, 1}, -2, 1, true, "-1/2", "1/4"},
		{"left-exact", []int32{2, -3, 1}, 1, 3, false, "2", "3"},
		{"right-exact", []int32{0, -1, 1}, -1, 1, false, "0", "1"},
		{"both-endpoints", []int32{-6, 11, -6, 1}, 1, 3, false, "2", "3"},
		{"both-endpoint-updates", []int32{0, 87, -59, 10}, 0, 3, true, "45/16", "93/32"},
	} {
		for _, disabled := range []bool{false, true} {
			t.Run(f.name+fmt.Sprint(disabled), func(t *testing.T) {
				m, q, budget := upolySourceOwner(t)
				bqm := upolyIsolationBinaryOwner(t, q, budget)
				p := upolySourceNew(t, m, f.coeff)
				v := mustUPolyView(t, m, p)
				a, b := unumeral.NewBinaryIntPower(f.a, 0), unumeral.NewBinaryIntPower(f.b, 0)
				t.Cleanup(func() { serviceMust(t, bqm.Del(&a)); serviceMust(t, bqm.Del(&b)) })
				polynomialAssertionActionsFixture(t, "")
				if polynomialSourceDebug && disabled {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				ok, e := m.IsolatingToRefinable(v, bqm, &a, &b)
				serviceMust(t, e)
				if ok != f.ok || upolyIsolationBinaryRat(t, m, bqm, &a).RatString() != f.wa || upolyIsolationBinaryRat(t, m, bqm, &b).RatString() != f.wb {
					t.Fatal("independent refinable oracle", ok, upolyIsolationBinaryRat(t, m, bqm, &a), upolyIsolationBinaryRat(t, m, bqm, &b), f)
				}
			})
		}
	}
}

func TestUPolynomialRefinableSignClosedGate(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	m, _, _ := upolySourceOwner(t)
	polynomialAssertionActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	if e := m.refinableSignAssertion(2605, 1, -1); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed runtime evaluated valid signs", e)
	}
}
