package spacer_test

// Source: src/muz/spacer/spacer_manager.cpp state conversion, background,
// unique numbering, three solver pools, and PropSolver construction.

import (
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/muz/spacer"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSpacerManagerBuildsStateAndPropSolverServices(t *testing.T) {
	command, err := cmd_context.NewStandaloneContext()
	spacerMust(t, err)
	manager, err := command.Manager()
	spacerMust(t, err)
	spacerMust(t, manager.ToggleProofMode(ast.ASTProofEnabled))
	globals, err := command.GlobalParameters()
	spacerMust(t, err)
	owner, err := spacer.NewManager(manager, nil, 2, smtcore.SMTSolverConfig{Params: globals.Context, Globals: globals})
	spacerMust(t, err)
	boolean, err := manager.MkBoolSort()
	spacerMust(t, err)
	name, err := manager.Symbols.Intern("manager-state")
	spacerMust(t, err)
	prefix, err := manager.MkFuncDecl(name, nil, boolean, ast.ASTFuncDeclInfo{})
	spacerMust(t, err)
	current, err := owner.NewPredicate(prefix)
	spacerMust(t, err)
	old0, err := owner.OldPredicate(prefix, 0)
	spacerMust(t, err)
	old1, err := owner.OldPredicate(prefix, 1)
	spacerMust(t, err)
	if converted, err := owner.OldToNew(old0, 0); err != nil || converted != current {
		t.Fatal("old-to-new", converted, current, err)
	}
	if converted, err := owner.OldToOld(old0, 0, 1); err != nil || converted != old1 {
		t.Fatal("old-to-old", converted, old1, err)
	}
	currentAtom, err := manager.MkApp(current, nil)
	spacerMust(t, err)
	old0Atom, err := manager.MkApp(old0, nil)
	spacerMust(t, err)
	old1Atom, err := manager.MkApp(old1, nil)
	spacerMust(t, err)
	mixedOld, err := manager.MkAnd(old0Atom, old1Atom)
	spacerMust(t, err)
	if isNew, err := owner.IsNewFormula(currentAtom); err != nil || !isNew {
		t.Fatal("new formula classification", isNew, err)
	}
	if isOld, err := owner.IsOldFormula(old0Atom); err != nil || !isOld {
		t.Fatal("old formula classification", isOld, err)
	}
	if homogeneous, err := owner.IsHomogeneousFormula(mixedOld); err != nil || homogeneous {
		t.Fatal("mixed old formula classification", homogeneous, err)
	}
	indices, err := owner.CollectStateIndices(mixedOld)
	spacerMust(t, err)
	if len(indices) != 2 || indices[0] != 0 || indices[1] != 1 {
		t.Fatal("manager state indices", indices)
	}
	if different, err := owner.HaveDifferentStateKinds(old0, old1); err != nil || !different {
		t.Fatal("different state kinds", different, err)
	}
	filtered, err := owner.FilterOldAtoms([]ast.ASTHandle{currentAtom, old0Atom, old1Atom, mixedOld}, 0)
	spacerMust(t, err)
	if len(filtered) != 1 || filtered[0] != old0Atom {
		t.Fatal("manager old-atom filter", filtered)
	}
	truth, err := manager.MkTrue()
	spacerMust(t, err)
	equality, err := manager.MkEq(truth, currentAtom)
	spacerMust(t, err)
	state, value, found, err := owner.TryStateAndValueFromAtom(equality)
	spacerMust(t, err)
	if !found || state != currentAtom || value != truth {
		t.Fatal("reversed state equality", state, value, found)
	}
	negated, err := manager.MkNot(currentAtom)
	spacerMust(t, err)
	state, value, found, err = owner.TryStateAndValueFromAtom(negated)
	spacerMust(t, err)
	if !found || state != currentAtom || !manager.IsFalse(value) {
		t.Fatal("negated state atom", state, value, found)
	}
	declaration, found, err := owner.TryStateDeclarationFromAtom(currentAtom)
	spacerMust(t, err)
	if !found || declaration != current {
		t.Fatal("state declaration", declaration, found)
	}
	skolem, err := spacer.MkZKConst(manager, 7, boolean)
	spacerMust(t, err)
	skolemFormula, err := manager.MkAnd(skolem, currentAtom)
	spacerMust(t, err)
	skolemValues, err := spacer.FindZKConstants(manager, skolemFormula)
	spacerMust(t, err)
	if len(skolemValues) != 1 || skolemValues[0] != skolem {
		t.Fatal("Skolem collection", skolemValues)
	}
	if has, err := spacer.HasZKConst(manager, currentAtom); err != nil || has {
		t.Fatal("ordinary state classified as Skolem-bearing", has, err)
	}
	modelCore, err := model.NewModelCore(manager)
	spacerMust(t, err)
	spacerMust(t, modelCore.RegisterConstant(old0, truth))
	modelText, err := owner.ModelString(modelCore)
	spacerMust(t, err)
	if !strings.Contains(modelText, "manager-state_0 -> true") {
		t.Fatal("manager model display", modelText)
	}
	spacerMust(t, modelCore.Close())
	if first, err := owner.Unique(); err != nil || first != 0 {
		t.Fatal("first unique", first, err)
	}
	if second, err := owner.Unique(); err != nil || second != 1 {
		t.Fatal("second unique", second, err)
	}
	prop, err := owner.NewPropSolver("manager-prop", spacer.PropSolverConfig{
		Itp: spacer.ItpSolverConfig{Params: globals.Context, Globals: globals},
	})
	spacerMust(t, err)
	p := spacerBool(t, manager, "manager-prop-p")
	notP, err := manager.MkNot(p)
	spacerMust(t, err)
	spacerMust(t, prop.Assert(notP))
	result, err := prop.CheckAssumptions([]ast.ASTHandle{p}, nil, spacer.PropCheckOptions{NeedCore: true})
	spacerMust(t, err)
	if result.Status != lbool.False || len(result.Core) != 1 {
		t.Fatal("manager PropSolver core", result.Status, len(result.Core))
	}
	spacerMust(t, result.Close(manager))
	spacerMust(t, prop.Close())
	spacerMust(t, owner.Close())
	spacerMust(t, command.Close())
}
