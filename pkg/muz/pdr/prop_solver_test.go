package pdr_test

// Source: src/muz/pdr/pdr_prop_solver.cpp level guards, safe proxies,
// subset-core recovery, temporary formulas and model acquisition.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/muz/pdr"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func pdrPropFixture(t *testing.T, maximum uint32) (*pdr.PropSolver, func()) {
	t.Helper()
	command, manager, contexts := pdrContextFixture(t, maximum)
	truth, err := manager.MkTrue()
	pdrMust(t, err)
	solver, err := pdr.NewPropSolver(manager, contexts, truth, "pdr-test")
	pdrMust(t, err)
	return solver, func() {
		pdrMust(t, solver.Close())
		pdrMust(t, contexts.Close())
		pdrMust(t, command.Close())
	}
}

func TestPropSolverLevelFramesCoresAndModels(t *testing.T) {
	command, manager, contexts := pdrContextFixture(t, 1)
	truth, err := manager.MkTrue()
	pdrMust(t, err)
	solver, err := pdr.NewPropSolver(manager, contexts, truth, "frames")
	pdrMust(t, err)
	p := pdrBool(t, manager, "frame-p")
	notP, err := manager.MkNot(p)
	pdrMust(t, err)
	pdrMust(t, solver.AddLevelFormula(p, 0))

	result, err := solver.CheckAtLevel(0, []ast.ASTHandle{notP}, true, false)
	pdrMust(t, err)
	if result.Status != lbool.False || !result.AssumesLevel || len(result.Core) != 1 || result.Core[0] != notP {
		t.Fatal("active level result", result.Status, result.AssumesLevel, result.Core)
	}
	result, err = solver.CheckAtLevel(1, []ast.ASTHandle{notP}, false, true)
	pdrMust(t, err)
	if result.Status != lbool.True || result.Model == nil {
		t.Fatal("inactive level model", result.Status, result.Model)
	}
	pdrMust(t, result.Model.DecRef())
	levels, err := solver.LevelCount()
	pdrMust(t, err)
	if levels != 2 {
		t.Fatal("ensured frame count", levels)
	}

	pdrMust(t, solver.Close())
	pdrMust(t, contexts.Close())
	pdrMust(t, command.Close())
}

func TestPropSolverSafeProxyCoreAndTemporaryFormulaRollback(t *testing.T) {
	command, manager, contexts := pdrContextFixture(t, 0)
	truth, err := manager.MkTrue()
	pdrMust(t, err)
	solver, err := pdr.NewPropSolver(manager, contexts, truth, "safe")
	pdrMust(t, err)
	p := pdrBool(t, manager, "safe-p")
	q := pdrBool(t, manager, "safe-q")
	notP, err := manager.MkNot(p)
	pdrMust(t, err)
	notQ, err := manager.MkNot(q)
	pdrMust(t, err)
	pdrMust(t, solver.AddFormula(notP))
	pdrMust(t, solver.AddFormula(notQ))
	disjunction, err := manager.MkOr(p, q)
	pdrMust(t, err)
	result, err := solver.CheckAssumptions([]ast.ASTHandle{disjunction}, true, false)
	pdrMust(t, err)
	if result.Status != lbool.False || len(result.Core) != 1 || result.Core[0] != disjunction {
		t.Fatal("proxy subset core", result.Status, result.Core)
	}

	r := pdrBool(t, manager, "safe-r")
	notR, err := manager.MkNot(r)
	pdrMust(t, err)
	result, err = solver.CheckAssumptionsAndFormula(nil, r, false, false)
	pdrMust(t, err)
	if result.Status != lbool.True {
		t.Fatal("temporary formula", result.Status)
	}
	result, err = solver.CheckAssumptions([]ast.ASTHandle{notR}, false, false)
	pdrMust(t, err)
	if result.Status != lbool.True {
		t.Fatal("temporary formula survived delayed scope", result.Status)
	}

	pdrMust(t, solver.Close())
	if _, err = solver.LevelCount(); !errors.Is(err, pdr.ErrPropSolverOwner) {
		t.Fatal("closed prop solver", err)
	}
	pdrMust(t, contexts.Close())
	pdrMust(t, command.Close())
}
