package nlsat

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func vanishingRef(t *testing.T, e *Evaluator, p polynomial.Polynomial) *polynomial.PolynomialRef {
	t.Helper()
	r, err := polynomial.NewPolynomialRef(e.core)
	assignmentMust(t, err)
	assignmentMust(t, r.Assign(p))
	t.Cleanup(func() { assignmentMust(t, r.Close()) })
	return r
}
func TestExplanationVanishingLeadingCoefficientsAndAssumptions(t *testing.T) {
	e, r, n := explanationAssumptionFixture(t)
	evaluatorSetFraction(t, e, 0, 1, 1)
	// (x0-1)*x1^3 + (x0^2-1)*x1^2 + 2*x1 + 7.
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 3}, evaluatorTerm{-1, 0, 3}, evaluatorTerm{1, 2, 2}, evaluatorTerm{-1, 0, 2}, evaluatorTerm{2, 0, 1}, evaluatorTerm{7, 0, 0})
	ref := vanishingRef(t, e, p)
	assignmentMust(t, n.collectVanishing(ref))
	result, err := ref.Get()
	assignmentMust(t, err)
	degree, err := e.core.Degree(result, 1)
	assignmentMust(t, err)
	if degree != 1 || len(n.output.result) != 1 || n.vanishing.active {
		t.Fatal("vanishing reductions or canonical assumptions", degree, n.output.result)
	}
	expected := evaluatorPolynomial(t, e, evaluatorTerm{2, 0, 1}, evaluatorTerm{7, 0, 0})
	equal, err := e.core.Equal(result, expected)
	assignmentMust(t, err)
	if !equal {
		t.Fatal("source reduct contents differ")
	}
	for _, l := range n.output.result {
		a := r.store.atoms.bools.atoms[l.Variable()]
		value, err := e.Eval(a, l.Sign())
		assignmentMust(t, err)
		if value {
			t.Fatal("vanishing assumption true in model")
		}
	}
}
func TestExplanationVanishingNonzeroAndConstantShortcuts(t *testing.T) {
	for _, value := range []int64{0, 2} {
		e, _, n := explanationAssumptionFixture(t)
		evaluatorSetFraction(t, e, 0, value, 1)
		p := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 4}, evaluatorTerm{3, 0, 0})
		ref := vanishingRef(t, e, p)
		assignmentMust(t, n.collectVanishing(ref))
		result, err := ref.Get()
		assignmentMust(t, err)
		if value == 2 {
			if result != p || len(n.output.result) != 0 {
				t.Fatal("nonzero symbolic leading coefficient changed input")
			}
		} else {
			constant, err := e.core.IsConstant(result)
			assignmentMust(t, err)
			if !constant || len(n.output.result) != 1 {
				t.Fatal("vanishing polynomial did not reduce to constant")
			}
		}
	}
}
func TestExplanationVanishingAllZeroAndVariableDescent(t *testing.T) {
	e, _, n := explanationAssumptionFixture(t)
	evaluatorSetFraction(t, e, 0, 0, 1)
	evaluatorSetFraction(t, e, 1, 0, 1)
	// x0*x1^2 + x0: eliminating x1 exposes x0 as the next maximal variable.
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 2}, evaluatorTerm{1, 1, 0})
	ref := vanishingRef(t, e, p)
	assignmentMust(t, n.collectVanishing(ref))
	result, err := ref.Get()
	assignmentMust(t, err)
	// Once x1 vanishes the new leading coefficient in x0 is the constant one.
	expected := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0})
	equal, err := e.core.Equal(result, expected)
	assignmentMust(t, err)
	if !equal || len(n.output.result) != 1 {
		t.Fatal("source maximal-variable descent", result)
	}
	// A polynomial with only positive powers of x1 drops directly to zero.
	assignmentMust(t, ref.Assign(evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 2})))
	assignmentMust(t, n.collectVanishing(ref))
	result, err = ref.Get()
	assignmentMust(t, err)
	zero, err := e.core.IsZero(result)
	assignmentMust(t, err)
	if !zero {
		t.Fatal("zero reduct not published")
	}
}

func TestExplanationVanishingAllocationStopsAndRetry(t *testing.T) {
	success, failures := false, 0
	for stop := int64(0); stop < 256 && !success; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			e, r, n := explanationAssumptionFixture(t)
			evaluatorSetFraction(t, e, 0, 1, 1)
			p := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 3}, evaluatorTerm{-1, 0, 3}, evaluatorTerm{2, 0, 1}, evaluatorTerm{7, 0, 0})
			ref := vanishingRef(t, e, p)
			e.budget.FailAfter(stop)
			err := n.collectVanishing(ref)
			e.budget.FailAfter(-1)
			if err == nil {
				success = true
				return
			}
			failures++
			var oom umemory.Failure
			if !errors.Is(err, arena.ErrInjectedFailure) && !(errors.As(err, &oom) && oom.Kind == umemory.FailureMemout && oom.ErrorCode() == 101) {
				t.Fatal("unexpected source stop", stop, err)
			}
			if err := r.newLevel(); !errors.Is(err, ErrSearchPending) {
				t.Fatal("failed normalization allowed parent search", err)
			}
			assignmentMust(t, e.am.CleanupRetained())
			assignmentMust(t, r.store.atoms.ineq.polynomials.CleanupFactor())
			assignmentMust(t, r.store.atoms.ineq.polynomials.CleanupUnique())
			assignmentMust(t, r.store.atoms.ineq.pm.CleanupFlipSignIfLeadingMonomialNegative())
			assignmentMust(t, r.store.atoms.cleanup())
			assignmentMust(t, n.cleanup())
			if n.vanishing.active || n.localFactor != nil || n.resultActive {
				t.Fatal("owning lexical cleanup incomplete", stop)
			}
			assignmentMust(t, r.store.atoms.ineq.polynomials.Reset())
			assignmentMust(t, ref.Assign(p))
			assignmentMust(t, n.collectVanishing(ref))
			result, err := ref.Get()
			assignmentMust(t, err)
			degree, err := e.core.Degree(result, 1)
			assignmentMust(t, err)
			if degree != 1 {
				t.Fatal("retry did not reduce leading degree", stop, degree)
			}
		})
	}
	if !success || failures < 3 {
		t.Fatal("source allocation horizon incomplete", success, failures)
	}
	t.Log("consecutive failures", failures)
}

func TestExplanationVanishingLexicalOrderAndCleanupCallbackGuard(t *testing.T) {
	e, r, n := explanationAssumptionFixture(t)
	evaluatorSetFraction(t, e, 0, 2, 1)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 2}, evaluatorTerm{1, 0, 2}, evaluatorTerm{1, 1, 1}, evaluatorTerm{2, 0, 1}, evaluatorTerm{3, 0, 0})
	ref := vanishingRef(t, e, p)
	stop := errors.New("stop younger reduct destructor")
	var order []polynomial.Polynomial
	var degrees []uint32
	var cleanupErr, parentErr error
	h, err := polynomial.NewPolynomialDeletionHandler(e.budget, func(_ *polynomial.PolynomialDeletionHandler, _ *polynomial.MultivariatePolynomialCore, p polynomial.Polynomial) error {
		order = append(order, p)
		degree, err := e.core.Degree(p, 1)
		if err != nil {
			return err
		}
		degrees = append(degrees, degree)
		if len(order) == 1 {
			return stop
		}
		cleanupErr = n.cleanup()
		parentErr = r.newLevel()
		return nil
	})
	assignmentMust(t, err)
	assignmentMust(t, e.core.AddDeletionHandler(h))
	t.Cleanup(func() { assignmentMust(t, e.core.RemoveDeletionHandler(h)); assignmentMust(t, h.CloseChecked()) })
	err = n.collectVanishing(ref)
	if !errors.Is(err, stop) || len(order) != 1 || !n.vanishing.active || n.vanishing.cursor != 0 {
		t.Fatal("source lexical stop not preserved", err, order, n.vanishing.cursor)
	}
	assignmentMust(t, n.cleanup())
	if len(order) != 2 || degrees[0] != 1 || degrees[1] != 0 || order[0] == order[1] || n.vanishing.active || !errors.Is(cleanupErr, ErrSearchPending) || !errors.Is(parentErr, ErrSearchPending) {
		t.Fatal("owning cleanup replayed destructor or allowed callback reentry", order, cleanupErr, parentErr)
	}
}

func TestExplanationVanishingConstantSourceGuard(t *testing.T) {
	atomDebugFixture(t)
	e, _, n := explanationAssumptionFixture(t)
	p := evaluatorPolynomial(t, e, evaluatorTerm{3, 0, 0})
	ref := vanishingRef(t, e, p)
	err := n.collectVanishing(ref)
	if diagnostic.DebugBuild {
		if !errors.Is(err, diagnostic.ErrDebugAssertion) {
			t.Fatal("source constant assertion not taken", err)
		}
	} else {
		assignmentMust(t, err)
	}
	assignmentMust(t, n.cleanup())
	result, err := ref.Get()
	assignmentMust(t, err)
	if result != p || len(n.output.result) != 0 || n.vanishing.active {
		t.Fatal("source guard changed caller ownership")
	}
}
