package smtcore

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func smtEnableCoreValidation(t *testing.T, s *SMTSolver) {
	t.Helper()
	p, err := s.config.Params.NewValue()
	astTestMust(t, err)
	defer func() { astTestMust(t, s.config.Params.Del(&p)) }()
	name := "core.validate"
	astTestMust(t, s.config.Params.SetBoolName(&p, &name, true))
	astTestMust(t, s.UpdateParams(&p))
}

func TestSMTCoreValidationAfterAssumptionCleanup(t *testing.T) {
	m, s := smtSolverFixture(t)
	smtEnableCoreValidation(t, s)
	a := smtRelevancyBool(t, m, "validated-core-a")
	notA, err := m.MkNot(a)
	astTestMust(t, err)
	astTestPin(t, m, notA)
	astTestMust(t, s.Assert(a))
	for i := 0; i < 2; i++ {
		status, err := s.CheckSatCore([]ast.ASTHandle{notA})
		astTestMust(t, err)
		if status != lbool.False || len(s.kernel.UnsatCore()) != 1 || s.kernel.UnsatCore()[0] != notA {
			t.Fatal("validated assumption conflict lost its core", status)
		}
		if s.kernel.internalizer.scopeLevel != 0 || len(s.kernel.assumptions) != 0 {
			t.Fatal("validation retained temporary assumptions")
		}
		status, err = s.CheckSatCore(nil)
		astTestMust(t, err)
		if status != lbool.True {
			t.Fatal("fresh validation contaminated the original context", status)
		}
	}
}

func TestSMTCoreValidationRejectsInvalidCoreAndHonorsDisabledFlag(t *testing.T) {
	_, s := smtSolverFixture(t)
	// Empty assertions and an empty extracted core are satisfiable. This
	// deliberately malformed core must fail only when validation is enabled.
	astTestMust(t, s.kernel.validateUnsatCore())
	smtEnableCoreValidation(t, s)
	err := s.kernel.validateUnsatCore()
	if err == nil || err.Error() != "Core could not be validated" {
		t.Fatal("invalid core accepted", err)
	}
}

func TestSMTCoreValidationRejectsUnknown(t *testing.T) {
	m, s := smtSolverFixture(t)
	smtEnableCoreValidation(t, s)
	m.Limit.Cancel()
	err := s.kernel.validateUnsatCore()
	m.Limit.ResetCancel()
	if err == nil || err.Error() != "Core could not be validated" {
		t.Fatal("unknown validation result accepted", err)
	}
}

func TestSMTCoreValidationAllocationFrontiers(t *testing.T) {
	succeeded := false
	for stop := int64(0); stop < 512 && !succeeded; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			m, s := smtSolverFixture(t)
			smtEnableCoreValidation(t, s)
			a := smtRelevancyBool(t, m, "validated-core-budget-a")
			notA, err := m.MkNot(a)
			astTestMust(t, err)
			astTestPin(t, m, notA)
			astTestMust(t, s.Assert(a))
			status, err := s.CheckSatCore([]ast.ASTHandle{notA})
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("fixture is not unsatisfiable")
			}
			m.Budget().FailAfter(stop)
			err = s.kernel.validateUnsatCore()
			m.Budget().FailAfter(-1)
			if err == nil {
				succeeded = true
			} else if !errors.Is(err, arena.ErrInjectedFailure) {
				t.Fatal("unexpected validation failure", err)
			}
		})
		if t.Failed() {
			t.Fatalf("core validation frontier %d", stop)
		}
	}
	if !succeeded {
		t.Fatal("no successful validation allocation frontier")
	}
}
