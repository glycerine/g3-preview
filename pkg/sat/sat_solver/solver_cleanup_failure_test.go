package sat_solver_test

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	incsat "github.com/glycerine/g3/pkg/sat/sat_solver"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestIncrementalSolverCleanupReceiptPreservesObservedCause(t *testing.T) {
	f, s := newSolverFixture(t)
	if !ast.ASTSourceDebug {
		return
	}
	incMust(t, s.Assert(solverForm(t, f, `(= (bvadd x y) #x3)`)))
	status, err := s.CheckSat(nil)
	incMust(t, err)
	if status != lbool.True {
		t.Fatal(status)
	}
	f.manager.Budget().FailAfter(0)
	err = s.Close()
	f.manager.Budget().FailAfter(-1)
	var failure *incsat.CleanupFailure
	if !errors.As(err, &failure) || !errors.Is(err, arena.ErrInjectedFailure) {
		t.Fatal("missing observed destructor receipt", err)
	}
	message := err.Error()
	if failure.RuntimeOwner() != f.manager.NumeralService().RuntimeOwner() {
		t.Fatal("receipt lost the actual runtime owner")
	}
	if again := s.Close(); !errors.Is(again, arena.ErrInjectedFailure) {
		t.Fatal("ordinary close suppressed the observed source guard", again)
	}
	incMust(t, failure.Cleanup())
	incMust(t, s.Close())
	if err.Error() != message {
		t.Fatal("cleanup discarded source failure history")
	}
}
