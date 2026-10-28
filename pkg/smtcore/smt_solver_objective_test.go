package smtcore

// Source: src/opt/opt_solver.cpp add_objective, maximize_objective and
// reset_objectives.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTSolverArithmeticObjectiveFacade(t *testing.T) {
	solver, fixture := smtModelUpdateFixture(t, false)
	startSMTModelUpdate(t, solver, fixture)
	x := fixture.variable("solver-objective-x")
	astTestMust(t, solver.Assert(fixture.app(smtArithOpLE, x, fixture.number("5"))))
	index, err := solver.AddObjective(x)
	astTestMust(t, err)
	if index != 0 {
		t.Fatal("first objective index", index)
	}
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("objective setup status", status)
	}
	value, blocker, shared, err := solver.MaximizeObjective(index)
	astTestMust(t, err)
	checkArithmeticObjective(t, value, "5", "0")
	if blocker.IsNull() || shared {
		t.Fatal("bounded unshared objective", blocker, shared)
	}
	updated, err := solver.UpdateModel(shared)
	astTestMust(t, err)
	if !updated || modelUpdateNumber(t, fixture.m, acquireUpdateModel(t, solver), x) != "5" {
		t.Fatal("optimized model was not published")
	}
	current, err := solver.ObjectiveValue(index)
	astTestMust(t, err)
	checkArithmeticObjective(t, current, "5", "0")
	astTestMust(t, solver.ResetObjectives())
	if _, err = solver.ObjectiveValue(index); !errors.Is(err, arena.ErrBounds) {
		t.Fatal("retired objective remained visible", err)
	}
	boolean, err := fixture.m.MkTrue()
	astTestMust(t, err)
	if _, err = solver.AddObjective(boolean); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("Boolean objective accepted", err)
	}
}
