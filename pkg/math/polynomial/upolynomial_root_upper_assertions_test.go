package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialRootUpperEntryAssertionActions(t *testing.T) {
	for _, route := range []string{"empty", "zero-leading"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(route+"/"+action, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				p := upolySourceNew(t, m, []int32{7})
				if route == "zero-leading" {
					serviceMust(t, m.Reserve(p, 2))
				}
				v := mustUPolyView(t, m, p)
				if route == "empty" {
					v = v.WithCount(0)
				}
				out := unumeral.NewInt(99)
				choice := action
				if choice == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				e := m.RootUpperBound(v, &out)
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) || out != unumeral.NewInt(99) || m.rootUpperBound.invocation != 0 {
						t.Fatal("entry Throw crossed source scalar declaration/output frontier", e, out)
					}
				} else if route == "empty" {
					wantInvocation := uint64(1)
					if polynomialSourceDebug && action == "C" {
						wantInvocation = 0
					}
					// Continue reaches the second assertion's read before NUMs;
					// erased predicates instead reach the body read after NUMs.
					if !errors.Is(e, arena.ErrBounds) || out != unumeral.NewInt(99) || m.rootUpperBound.invocation != wantInvocation {
						t.Fatal("empty continuation crossed wrong scalar/read frontier", e, out, m.rootUpperBound.invocation, wantInvocation)
					}
				} else {
					// First bound (7+7)/7+1=3 is published before second bound divides by
					// the genuine zero leading coefficient. This invalid input has no bound.
					if !errors.Is(e, unumeral.ErrDivideByZero) || out != unumeral.NewInt(3) || m.rootUpperBound.invocation == 0 {
						t.Fatal("zero-leading continuation lost first-bound publication", e, out)
					}
				}
				if m.factorTrace != nil {
					t.Fatal("root-upper assertion retained borrow")
				}
			})
		}
	}
}
func TestUPolynomialRootUpperOrderingAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			a, b := unumeral.NewInt(2), unumeral.NewInt(1)
			choice := action
			if choice == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			// Qualified predicate adapter, not a manufactured failed Cauchy result.
			e := m.rootUpperOrderingAssertion(&a, &b)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal(e)
				}
			} else {
				serviceMust(t, e)
			}
			if a != unumeral.NewInt(2) || b != unumeral.NewInt(1) {
				t.Fatal("assertion changed operands")
			}
		})
	}
}
func TestUPolynomialRootUpperClosedEntryAndErasedOrdering(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{1})
	v := mustUPolyView(t, m, p)
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	serviceMust(t, m.rootUpperOrderingAssertion(nil, nil))
	if !polynomialSourceDebug {
		return
	}
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	out := unumeral.NewInt(99)
	e := m.RootUpperBound(v, &out)
	if !errors.Is(e, arena.ErrClosed) || out != unumeral.NewInt(99) || m.rootUpperBound.invocation != 0 {
		t.Fatal("closed entry gate crossed first scalar declaration", e)
	}
	if e = m.rootUpperOrderingAssertion(nil, nil); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ordering operands read before runtime admission", e)
	}
}
