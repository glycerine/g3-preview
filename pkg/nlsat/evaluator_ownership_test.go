package nlsat

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestEvaluatorProjectionAllocationCleanupAndRetry(t *testing.T) {
	for _, root := range []bool{false, true} {
		for _, stop := range []int64{0, 1, 2, 3, 4, 5, 8, 12, 16, 24, 32, 64, 128, -1} {
			t.Run(fmt.Sprintf("root=%v/stop=%d", root, stop), func(t *testing.T) {
				e := evaluatorFixture(t)
				p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 2}, evaluatorTerm{-1, 1, 0})
				evaluatorSetFraction(t, e, 0, 2, 1)
				var a ArithmeticAtom
				if root {
					r := &RootAtom{}
					assignmentMust(t, r.initialize(AtomRootEQ, 1, 2, p))
					r.boolVar = 13
					a = r
				} else {
					i := &InequalityAtom{}
					assignmentMust(t, i.initialize(AtomGT, []polynomial.Polynomial{p}, []bool{false}, 1, make([]InequalityFactor, 1)))
					i.boolVar = 13
					a = i
				}
				e.budget.FailAfter(stop)
				s, err := e.InfeasibleIntervals(a, false)
				e.budget.FailAfter(-1)
				assignmentMust(t, e.am.CleanupRetained())
				assignmentMust(t, e.Cleanup())
				if s != nil {
					assignmentMust(t, e.ism.DecRef(s))
				}
				var physical umemory.Failure
				if err != nil && !errors.Is(err, arena.ErrInjectedFailure) && !(errors.As(err, &physical) && physical.Kind == umemory.FailureMemout && physical.ErrorCode() == 101) {
					t.Fatal("unexpected failure", err)
				}
				if len(e.ism.live) != 0 || len(e.ism.pending) != 0 || len(e.ism.retired) != 0 {
					t.Fatal("projection retained interval owners", len(e.ism.live))
				}
				s, err = e.InfeasibleIntervals(a, false)
				assignmentMust(t, err)
				assignmentMust(t, e.ism.DecRef(s))
			})
		}
	}
}

func TestEvaluatorBorrowedOwnersLiveSetsAndEarlyZero(t *testing.T) {
	e := evaluatorFixture(t)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1})
	evaluatorSetFraction(t, e, 1, 0, 1)
	var a InequalityAtom
	assignmentMust(t, a.initialize(AtomEQ, []polynomial.Polynomial{p, {}}, []bool{false, false}, 1, make([]InequalityFactor, 2)))
	got, err := e.Eval(&a, false)
	assignmentMust(t, err)
	if !got {
		t.Fatal("zero factor must stop before invalid later factor")
	}
	var valid InequalityAtom
	assignmentMust(t, valid.initialize(AtomEQ, []polynomial.Polynomial{p}, []bool{false}, 1, make([]InequalityFactor, 1)))
	valid.boolVar = 7
	s, err := e.InfeasibleIntervals(&valid, false)
	assignmentMust(t, err)
	if err = e.Close(); !errors.Is(err, ErrIntervalLive) {
		t.Fatal("accepted outstanding reference", err)
	}
	assignmentMust(t, e.ism.DecRef(s))
	copy := *e
	if _, err = copy.Eval(&valid, false); !errors.Is(err, polynomial.ErrAlgebraicOwner) {
		t.Fatal("copied evaluator", err)
	}
	assignmentMust(t, e.Push())
	assignmentMust(t, e.Pop(17))
	assignmentMust(t, e.Close())
	assignmentMust(t, e.Close())
	if _, err = e.Eval(&valid, false); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed evaluator", err)
	}
	// Assignment and polynomial manager are borrowed and remain usable.
	_, err = e.assignment.Contains(1)
	assignmentMust(t, err)
	_, err = e.pm.ArithmeticCore()
	assignmentMust(t, err)
}

func TestEvaluatorConstructorReservationOwnership(t *testing.T) {
	e := evaluatorFixture(t)
	for stop := int64(0); stop <= 6; stop++ {
		e.budget.FailAfter(stop)
		candidate, err := NewEvaluator(e.assignment, e.pm, e.ism.allocator)
		e.budget.FailAfter(-1)
		if err != nil && !errors.Is(err, arena.ErrInjectedFailure) {
			t.Fatal("constructor refusal", stop, err)
		}
		if candidate != nil {
			assignmentMust(t, candidate.Close())
		}
	}
}
