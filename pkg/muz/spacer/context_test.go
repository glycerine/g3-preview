package spacer_test

// Source: src/muz/spacer/spacer_context.cpp init_rules and solve_core narrow
// end-to-end Horn reachability paths.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/muz/base"
	"github.com/glycerine/g3/pkg/muz/spacer"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSpacerContextSolvesFactAndRecursiveNoBasePrograms(t *testing.T) {
	command, err := cmd_context.NewStandaloneContext()
	spacerMust(t, err)
	manager, err := command.Manager()
	spacerMust(t, err)
	globals, err := command.GlobalParameters()
	spacerMust(t, err)
	config := smtcore.SMTSolverConfig{Params: globals.Context, Globals: globals}
	boolean, err := manager.MkBoolSort()
	spacerMust(t, err)

	factManager, err := base.NewRuleManager(manager)
	spacerMust(t, err)
	factName, err := manager.Symbols.Intern("spacer-context-fact")
	spacerMust(t, err)
	factPredicate, err := manager.MkFuncDecl(factName, nil, boolean, ast.ASTFuncDeclInfo{})
	spacerMust(t, err)
	spacerMust(t, factManager.RegisterPredicate(factPredicate))
	factHead, err := manager.MkApp(factPredicate, nil)
	spacerMust(t, err)
	factRule, err := factManager.NewRule(factHead, nil, nil, "fact")
	spacerMust(t, err)
	factSet, err := base.NewRuleSet(factManager)
	spacerMust(t, err)
	spacerMust(t, factSet.Add(factRule))
	factContext, err := spacer.NewContext(manager, nil, 1, config)
	spacerMust(t, err)
	spacerMust(t, factContext.Initialize(factSet))
	spacerMust(t, factContext.SetQuery(factPredicate))
	if status, err := factContext.Solve(0, 3); err != nil || status != lbool.True {
		t.Fatal("Spacer fact program", status, err)
	}
	spacerMust(t, factContext.Close())
	spacerMust(t, factSet.Close())
	spacerMust(t, factRule.Close())
	spacerMust(t, factManager.Close())

	chainManager, err := base.NewRuleManager(manager)
	spacerMust(t, err)
	baseName, err := manager.Symbols.Intern("spacer-context-chain-base")
	spacerMust(t, err)
	basePredicate, err := manager.MkFuncDecl(baseName, nil, boolean, ast.ASTFuncDeclInfo{})
	spacerMust(t, err)
	queryName, err := manager.Symbols.Intern("spacer-context-chain-query")
	spacerMust(t, err)
	queryPredicate, err := manager.MkFuncDecl(queryName, nil, boolean, ast.ASTFuncDeclInfo{})
	spacerMust(t, err)
	spacerMust(t, chainManager.RegisterPredicate(basePredicate))
	spacerMust(t, chainManager.RegisterPredicate(queryPredicate))
	baseHead, err := manager.MkApp(basePredicate, nil)
	spacerMust(t, err)
	queryHead, err := manager.MkApp(queryPredicate, nil)
	spacerMust(t, err)
	baseRule, err := chainManager.NewRule(baseHead, nil, nil, "base")
	spacerMust(t, err)
	chainRule, err := chainManager.NewRule(queryHead, []ast.ASTHandle{baseHead}, []bool{false}, "chain")
	spacerMust(t, err)
	chainSet, err := base.NewRuleSet(chainManager)
	spacerMust(t, err)
	spacerMust(t, chainSet.Add(baseRule))
	spacerMust(t, chainSet.Add(chainRule))
	chainContext, err := spacer.NewContext(manager, nil, 1, config)
	spacerMust(t, err)
	spacerMust(t, chainContext.Initialize(chainSet))
	spacerMust(t, chainContext.SetQuery(queryPredicate))
	if status, err := chainContext.Solve(0, 4); err != nil || status != lbool.True {
		t.Fatal("Spacer cross-predicate chain", status, err)
	}
	spacerMust(t, chainContext.Close())
	spacerMust(t, chainSet.Close())
	spacerMust(t, chainRule.Close())
	spacerMust(t, baseRule.Close())
	spacerMust(t, chainManager.Close())

	recursiveManager, err := base.NewRuleManager(manager)
	spacerMust(t, err)
	recursiveName, err := manager.Symbols.Intern("spacer-context-recursive")
	spacerMust(t, err)
	recursivePredicate, err := manager.MkFuncDecl(recursiveName, nil, boolean, ast.ASTFuncDeclInfo{})
	spacerMust(t, err)
	spacerMust(t, recursiveManager.RegisterPredicate(recursivePredicate))
	recursiveHead, err := manager.MkApp(recursivePredicate, nil)
	spacerMust(t, err)
	recursiveRule, err := recursiveManager.NewRule(recursiveHead, []ast.ASTHandle{recursiveHead}, []bool{false}, "recursive")
	spacerMust(t, err)
	recursiveSet, err := base.NewRuleSet(recursiveManager)
	spacerMust(t, err)
	spacerMust(t, recursiveSet.Add(recursiveRule))
	recursiveContext, err := spacer.NewContext(manager, nil, 1, config)
	spacerMust(t, err)
	spacerMust(t, recursiveContext.Initialize(recursiveSet))
	spacerMust(t, recursiveContext.SetQuery(recursivePredicate))
	if status, err := recursiveContext.Solve(0, 4); err != nil || status != lbool.False {
		t.Fatal("Spacer recursive no-base program", status, err)
	}
	spacerMust(t, recursiveContext.Close())
	spacerMust(t, recursiveSet.Close())
	spacerMust(t, recursiveRule.Close())
	spacerMust(t, recursiveManager.Close())
	spacerMust(t, command.Close())
}
