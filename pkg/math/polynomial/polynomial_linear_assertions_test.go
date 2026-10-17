package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
	"testing"
)

func TestLinearSourceConstructorAssertionActionsAndErasure(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, budget := modularManager(t, 0)
			choices := action
			if action == "disabled" {
				choices = ""
			}
			polynomialAssertionActionsFixture(t, choices)
			if action == "disabled" && polynomialSourceDebug {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			before := budget.Used()
			budget.FailAfter(0)
			var local ModularLinearSolver
			e := local.constructDefault(m)
			budget.FailAfter(-1)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) || !linearThrownAssertion(e) || local.self != nil {
					t.Fatal("source constructor Throw", e)
				}
			} else {
				serviceMust(t, e)
				if local.self != &local || local.header != 0 {
					t.Fatal("continued/erased constructor did not complete")
				}
				serviceMust(t, local.Close())
			}
			if before != budget.Used() {
				t.Fatal("source assertion acquired numeric storage")
			}
		})
	}
}

func TestLinearSourceAddAssertionBeforeReachedOperands(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			q, _ := rationalManager(t, false)
			s, e := NewRationalLinearSolver(q)
			serviceMust(t, e)
			defer func() { serviceMust(t, s.Close()) }()
			serviceMust(t, s.Resize(1))
			choices := action
			if action == "disabled" {
				choices = ""
			}
			polynomialAssertionActionsFixture(t, choices)
			if action == "disabled" && polynomialSourceDebug {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			e = s.Add(1, nil, nil)
			if polynomialSourceDebug && action == "T" {
				if !linearThrownAssertion(e) {
					t.Fatal("source Add guard did not Throw before operand access", e)
				}
			} else if e == nil || linearThrownAssertion(e) {
				t.Fatal("continued/erased guard did not reach independent bounds protection", e)
			}
		})
	}
}

func linearThrownAssertion(e error) bool {
	var exception z3exception.DefaultException
	return errors.As(e, &exception) && exception.Message() == "assertion violation" && !exception.HasErrorCode()
}
