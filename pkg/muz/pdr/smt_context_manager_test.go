package pdr_test

// Source: src/muz/pdr/pdr_smt_context_manager.cpp guarded context reuse and
// delayed-scope behavior, exercised through the retained Go SMT backend.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/muz/pdr"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func pdrMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func pdrContextFixture(t *testing.T, maximum uint32) (*cmd_context.Context, *ast.ASTManager, *pdr.SMTContextManager) {
	t.Helper()
	command, err := cmd_context.NewStandaloneContext()
	pdrMust(t, err)
	manager, err := command.Manager()
	pdrMust(t, err)
	globals, err := command.GlobalParameters()
	pdrMust(t, err)
	contexts, err := pdr.NewSMTContextManager(manager, nil, maximum, smtcore.SMTSolverConfig{Params: globals.Context, Globals: globals})
	pdrMust(t, err)
	return command, manager, contexts
}

func pdrBool(t *testing.T, manager *ast.ASTManager, name string) ast.ASTHandle {
	t.Helper()
	symbol, err := manager.Symbols.Intern(name)
	pdrMust(t, err)
	boolean, err := manager.MkBoolSort()
	pdrMust(t, err)
	value, err := manager.MkConst(symbol, boolean)
	pdrMust(t, err)
	return value
}

func TestSMTContextManagerReusesGuardedSolverWithoutCrossTalk(t *testing.T) {
	command, manager, contexts := pdrContextFixture(t, 1)
	first, err := contexts.Fresh()
	pdrMust(t, err)
	second, err := contexts.Fresh()
	pdrMust(t, err)
	p := pdrBool(t, manager, "pdr-context-p")
	notP, err := manager.MkNot(p)
	pdrMust(t, err)
	pdrMust(t, first.Assert(p))
	pdrMust(t, second.Assert(notP))

	for _, check := range []struct {
		context     *pdr.SMTContext
		assumptions []ast.ASTHandle
		want        lbool.LBool
	}{
		{first, nil, lbool.True},
		{first, []ast.ASTHandle{notP}, lbool.False},
		{second, nil, lbool.True},
		{second, []ast.ASTHandle{p}, lbool.False},
	} {
		got, x := check.context.Check(check.assumptions)
		pdrMust(t, x)
		if got != check.want {
			t.Fatalf("guarded check: got %v want %v", got, check.want)
		}
	}

	for _, context := range []*pdr.SMTContext{first, second} {
		predicate, x := context.Predicate()
		pdrMust(t, x)
		declaration, x := manager.GetDecl(predicate)
		pdrMust(t, x)
		auxiliary, x := context.IsAuxPredicate(declaration)
		pdrMust(t, x)
		if !auxiliary {
			t.Fatal("guard predicate was not registered as auxiliary")
		}
	}

	pdrMust(t, second.Close())
	pdrMust(t, first.Close())
	pdrMust(t, contexts.Close())
	pdrMust(t, command.Close())
}

func TestSMTContextDelayScopeRollsBackLocalAssertions(t *testing.T) {
	command, manager, contexts := pdrContextFixture(t, 0)
	context, err := contexts.Fresh()
	pdrMust(t, err)
	p := pdrBool(t, manager, "pdr-delay-p")
	notP, err := manager.MkNot(p)
	pdrMust(t, err)
	pdrMust(t, context.Assert(p))
	scope, err := context.BeginDelay()
	pdrMust(t, err)
	pdrMust(t, context.Assert(notP))
	status, err := context.Check(nil)
	pdrMust(t, err)
	if status != lbool.False {
		t.Fatal("delayed contradiction", status)
	}
	pdrMust(t, scope.Close())
	status, err = context.Check(nil)
	pdrMust(t, err)
	if status != lbool.True {
		t.Fatal("delayed assertion survived scope", status)
	}
	model, err := context.Model()
	pdrMust(t, err)
	if model == nil {
		t.Fatal("SMT context did not expose its model")
	}
	pdrMust(t, model.DecRef())

	predicate, err := context.Predicate()
	pdrMust(t, err)
	truth, err := manager.MkTrue()
	pdrMust(t, err)
	if predicate != truth {
		t.Fatal("unpooled context did not use the true guard")
	}
	pdrMust(t, context.Close())
	pdrMust(t, contexts.Close())
	pdrMust(t, command.Close())
}
