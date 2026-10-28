package nlsat

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func projectionCoefficientFixture(t *testing.T, factor bool) (*Evaluator, *searchState, *explanationAssumptions, *projectionWorklist) {
	t.Helper()
	e, r, n := explanationAssumptionFixture(t)
	todo, err := newProjectionWorklist(r)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, todo.close()) })
	assignmentMust(t, n.bindProjection(todo, factor))
	return e, r, n, todo
}
func TestProjectionFactorsCanonicalDistinctAndDisabledMode(t *testing.T) {
	for _, factor := range []bool{false, true} {
		e, _, n, todo := projectionCoefficientFixture(t, factor)
		// (x0-1)^2 * (x0+2) = x0^3-3*x0+2.
		p := evaluatorPolynomial(t, e, evaluatorTerm{1, 3, 0}, evaluatorTerm{-3, 1, 0}, evaluatorTerm{2, 0, 0})
		ref := vanishingRef(t, e, p)
		assignmentMust(t, n.collectProjectionFactors(ref))
		count, err := todo.set.Size()
		assignmentMust(t, err)
		if factor {
			if count != 2 {
				t.Fatal("distinct factors/multiplicity", count)
			}
		} else {
			if count != 1 {
				t.Fatal("disabled factoring changed worklist", count)
			}
		}
		for _, q := range projectionValues(t, todo.set) {
			degree, err := e.core.Degree(q, 0)
			assignmentMust(t, err)
			want := uint32(3)
			if factor {
				want = 1
			}
			if degree != want {
				t.Fatal("wrong factor degree", degree, want)
			}
		}
		assignmentMust(t, n.collectProjectionFactors(ref))
		again, err := todo.set.Size()
		assignmentMust(t, err)
		if again != count {
			t.Fatal("canonical duplicate reinsertion", again, count)
		}
	}
}
func TestProjectionFactorsVanishingAndConstantShortcuts(t *testing.T) {
	e, _, n, todo := projectionCoefficientFixture(t, true)
	evaluatorSetFraction(t, e, 0, 0, 1)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 3}, evaluatorTerm{1, 0, 1}, evaluatorTerm{1, 0, 0})
	ref := vanishingRef(t, e, p)
	assignmentMust(t, n.collectProjectionFactors(ref))
	values := projectionValues(t, todo.set)
	if len(values) != 1 || len(n.output.result) != 1 {
		t.Fatal("reduction/zero assumptions", values, n.output.result)
	}
	expected := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1}, evaluatorTerm{1, 0, 0})
	equal, err := e.core.Equal(values[0], expected)
	assignmentMust(t, err)
	if !equal {
		t.Fatal("unreduced vanishing polynomial inserted")
	}
	assignmentMust(t, ref.Assign(evaluatorPolynomial(t, e, evaluatorTerm{7, 0, 0})))
	assignmentMust(t, n.collectProjectionFactors(ref))
	if len(projectionValues(t, todo.set)) != 1 {
		t.Fatal("constant inserted into todo")
	}
	assignmentMust(t, ref.Assign(evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 2})))
	assignmentMust(t, n.collectProjectionFactors(ref))
	value, err := ref.Get()
	assignmentMust(t, err)
	zero, err := e.core.IsZero(value)
	assignmentMust(t, err)
	if !zero || len(projectionValues(t, todo.set)) != 1 {
		t.Fatal("zero reduct did not short circuit")
	}
}
func TestProjectionLeadingCoefficientsFactorAndSkipConstants(t *testing.T) {
	for _, factor := range []bool{false, true} {
		e, _, n, todo := projectionCoefficientFixture(t, factor)
		evaluatorSetFraction(t, e, 0, 3, 1)
		ps := projectionOutput(t, e)
		// (y^2-y-2)*x^3+x+1, whose lc factors as (y-2)(y+1).
		p := evaluatorPolynomial(t, e, evaluatorTerm{1, 2, 3}, evaluatorTerm{-1, 1, 3}, evaluatorTerm{-2, 0, 3}, evaluatorTerm{1, 0, 1}, evaluatorTerm{1, 0, 0})
		constantLC := evaluatorPolynomial(t, e, evaluatorTerm{5, 0, 2}, evaluatorTerm{1, 1, 0})
		for _, p := range []polynomial.Polynomial{p, constantLC, p} {
			assignmentMust(t, ps.Push(p))
		}
		assignmentMust(t, n.collectLeadingCoefficients(ps, 1))
		values := projectionValues(t, todo.set)
		want := 1
		if factor {
			want = 2
		}
		if len(values) != want || len(n.output.result) != 0 {
			t.Fatal("constant lc skip or factorization", values, n.output.result)
		}
		for _, p := range values {
			degree, err := e.core.Degree(p, 0)
			assignmentMust(t, err)
			want := uint32(2)
			if factor {
				want = 1
			}
			if degree != want {
				t.Fatal("lc polynomial contents", degree, want)
			}
		}
		if len(projectionValues(t, ps)) != 3 {
			t.Fatal("borrowed input vector changed")
		}
	}
}

func TestProjectionCoefficientAndPSCAllocationStops(t *testing.T) {
	for _, psc := range []bool{false, true} {
		t.Run(map[bool]string{false: "leading", true: "psc"}[psc], func(t *testing.T) {
			success, failures := false, 0
			for stop := int64(0); stop < 256 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					e, r, n, todo := projectionCoefficientFixture(t, true)
					evaluatorSetFraction(t, e, 0, 2, 1)
					p := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 2}, evaluatorTerm{1, 1, 1}, evaluatorTerm{1, 0, 1}, evaluatorTerm{2, 0, 0})
					q := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 1}, evaluatorTerm{3, 0, 0})
					ps := projectionOutput(t, e)
					assignmentMust(t, ps.Push(p))
					pRef, qRef := vanishingRef(t, e, p), vanishingRef(t, e, q)
					call := func() error {
						if psc {
							return n.collectProjectionPSC(pRef, qRef, 1)
						}
						return n.collectLeadingCoefficients(ps, 1)
					}
					e.budget.FailAfter(stop)
					err := call()
					e.budget.FailAfter(-1)
					if err == nil {
						success = true
						return
					}
					failures++
					var oom umemory.Failure
					if !errors.Is(err, arena.ErrInjectedFailure) && !(errors.As(err, &oom) && oom.Kind == umemory.FailureMemout && oom.ErrorCode() == 101) {
						t.Fatal("unexpected projection failure", stop, err)
					}
					if err := r.newLevel(); !errors.Is(err, ErrSearchPending) {
						t.Fatal("pending projection allowed search", err)
					}
					// These real borrowed owners acknowledge their own accepted operations.
					assignmentMust(t, e.am.CleanupRetained())
					assignmentMust(t, r.store.atoms.ineq.polynomials.CleanupPSC())
					assignmentMust(t, r.store.atoms.ineq.polynomials.CleanupFactor())
					assignmentMust(t, r.store.atoms.ineq.polynomials.CleanupUnique())
					assignmentMust(t, r.store.atoms.ineq.pm.CleanupFlipSignIfLeadingMonomialNegative())
					assignmentMust(t, r.store.atoms.cleanup())
					assignmentMust(t, n.cleanup())
					assignmentMust(t, todo.cleanup())
					if n.leading.active || n.projectionFactors.active || n.psc.active || n.resultant.active || n.vanishing.active {
						t.Fatal("owning projection locals remain", stop)
					}
					// Reset the real worklist and interrupted cache before source retry.
					assignmentMust(t, todo.reset())
					assignmentMust(t, r.store.atoms.ineq.polynomials.Reset())
					assignmentMust(t, call())
					count, err := todo.set.Size()
					assignmentMust(t, err)
					if count == 0 {
						t.Fatal("retry did not supply projection polynomials", stop)
					}
				})
			}
			if !success || failures < 2 {
				t.Fatal("projection allocation horizon incomplete", success, failures)
			}
			t.Log("consecutive failures", failures)
		})
	}
}
func TestProjectionPartialBindingRemainsPending(t *testing.T) {
	e, r, n := explanationAssumptionFixture(t)
	todo, err := newProjectionWorklist(r)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, todo.close()) })
	e.budget.FailAfter(0)
	err = n.bindProjection(todo, true)
	e.budget.FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) || n.projectionTodo != todo || n.pscTmp != nil {
		t.Fatal("actual persistent PSC-vector construction did not stop", err)
	}
	assignmentMust(t, n.cleanup())
	if err := r.newLevel(); !errors.Is(err, ErrSearchPending) {
		t.Fatal("cleanup made incomplete projection binding usable", err)
	}
	assignmentMust(t, r.close())
	if !n.closed || r.projection != nil {
		t.Fatal("parent did not close partial binding")
	}
}

func TestProjectionLeadingConstantDoesNotConstructCoefficient(t *testing.T) {
	e, _, n, todo := projectionCoefficientFixture(t, true)
	p := evaluatorPolynomial(t, e, evaluatorTerm{5, 0, 2}, evaluatorTerm{1, 1, 0})
	ps := projectionOutput(t, e)
	assignmentMust(t, ps.Push(p))
	calls := 0
	h, err := polynomial.NewPolynomialDeletionHandler(e.budget, func(_ *polynomial.PolynomialDeletionHandler, _ *polynomial.MultivariatePolynomialCore, _ polynomial.Polynomial) error {
		calls++
		return nil
	})
	assignmentMust(t, err)
	assignmentMust(t, e.core.AddDeletionHandler(h))
	t.Cleanup(func() { assignmentMust(t, e.core.RemoveDeletionHandler(h)); assignmentMust(t, h.CloseChecked()) })
	assignmentMust(t, n.collectLeadingCoefficients(ps, 1))
	if calls != 0 || len(projectionValues(t, todo.set)) != 0 {
		t.Fatal("nonzero-constant shortcut constructed a polynomial coefficient", calls)
	}
}
func TestProjectionWorklistExplanationBindingGuard(t *testing.T) {
	e, _, n, todo := projectionCoefficientFixture(t, false)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0})
	before := e.budget.Used()
	err := todo.insertFromExplanation(n, p)
	if !errors.Is(err, ErrSearchOwner) || e.budget.Used() != before || len(projectionValues(t, todo.set)) != 0 {
		t.Fatal("idle caller bypassed actual explanation binding", err)
	}
	ref := vanishingRef(t, e, p)
	assignmentMust(t, n.collectProjectionFactors(ref))
	if len(projectionValues(t, todo.set)) != 1 {
		t.Fatal("accepted explanation failed canonical insertion")
	}
}
