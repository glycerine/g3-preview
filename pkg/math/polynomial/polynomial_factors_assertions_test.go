package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialFactorsSourceScalarAssertionActions(t *testing.T) {
	for _, operation := range []string{"push-zero-degree", "set-first-degree"} {
		for _, choice := range []string{"T", "C", "disabled"} {
			t.Run(operation+"/"+choice, func(t *testing.T) {
				m, p, _ := polynomialEvalFixture(t, nil)
				f, e := NewPolynomialFactors(m)
				serviceMust(t, e)
				t.Cleanup(func() { serviceMust(t, f.CloseChecked()) })
				action := choice
				if choice == "disabled" {
					action = "T"
				}
				polynomialAssertionActionsFixture(t, action)
				if polynomialSourceDebug && choice == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				if operation == "set-first-degree" {
					serviceMust(t, f.Push(p, 2))
					e = f.SetDegree(0, 3)
				} else {
					e = f.Push(p, 0)
				}
				thrown := polynomialSourceDebug && choice == "T"
				if thrown {
					if !errors.Is(e, ErrPolynomialAssertion) {
						t.Fatal("source Throw did not stop before factor mutation", e)
					}
				} else {
					serviceMust(t, e)
				}
				if operation == "set-first-degree" {
					want := uint32(3)
					if thrown {
						want = 2
					}
					if f.total != want || len(f.factors) != 1 || f.degrees[0] != want {
						t.Fatal("source first-index action/order", f.total, f.degrees)
					}
				} else {
					want := 1
					if thrown {
						want = 0
					}
					if len(f.factors) != want || f.total != 0 {
						t.Fatal("source zero-degree publication/count", len(f.factors), f.total)
					}
					if want != 0 && (f.degrees[0] != 0 || f.phases[0] != factorReferenced) {
						t.Fatal("Continue/release skipped actual degree publication or INC")
					}
				}
				// Every returning debugger action must leave the class usable.
				serviceMust(t, f.Push(p, 1))
				polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
			})
		}
	}
}

func TestPolynomialFactorsSourceScalarAssertionClosedOwner(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source SASSERT erased")
	}
	m, p, _ := polynomialEvalFixture(t, nil)
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, f.CloseChecked()) })
	serviceMust(t, f.Push(p, 2))
	serviceMust(t, f.Push(p, 3))
	polynomialAssertionActionsFixture(t, "T")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	// The source reads the assertion switch even for true scalar predicates.
	if e := f.SetDegree(1, 4); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("set_degree skipped its source switch read", e)
	}
	if e := f.Push(p, 1); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("push_back skipped its source switch read", e)
	}
	if len(f.factors) != 2 || f.total != 5 || f.degrees[1] != 3 {
		t.Fatal("closed assertion owner changed factors before refusal")
	}
}
