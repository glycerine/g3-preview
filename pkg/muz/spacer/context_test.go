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
	spacerMust(t, globals.SourceSetName("fixedpoint.pdr.validate_result", "true"))
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
	reachableBase, err := chainContext.Reachable(basePredicate)
	spacerMust(t, err)
	reachableBaseHandle, err := manager.RefGet(reachableBase)
	spacerMust(t, err)
	if !manager.IsTrue(reachableBaseHandle) {
		t.Fatal("Spacer base reach summary", reachableBaseHandle)
	}
	spacerMust(t, manager.DelRef(&reachableBase))
	traceRules, err := chainContext.RulesAlongTrace()
	spacerMust(t, err)
	if len(traceRules) != 2 {
		t.Fatal("Spacer preorder trace rule count", len(traceRules))
	}
	firstName, err := traceRules[0].Name()
	spacerMust(t, err)
	secondName, err := traceRules[1].Name()
	spacerMust(t, err)
	if firstName != "chain" || secondName != "base" {
		t.Fatal("Spacer preorder trace rules", firstName, secondName)
	}
	for index := len(traceRules) - 1; index >= 0; index-- {
		spacerMust(t, traceRules[index].Close())
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
	certificate, err := recursiveContext.Answer(recursivePredicate)
	spacerMust(t, err)
	certificateHandle, err := manager.RefGet(certificate)
	spacerMust(t, err)
	if !manager.IsTrue(certificateHandle) {
		t.Fatal("single-query Spacer certificate is not the empty conjunction")
	}
	spacerMust(t, manager.DelRef(&certificate))
	inductiveModel, err := recursiveContext.Model(recursivePredicate)
	spacerMust(t, err)
	if declarations, err := inductiveModel.Core().NumDeclarations(); err != nil || declarations != 0 {
		t.Fatal("single-query Spacer model", declarations, err)
	}
	spacerMust(t, inductiveModel.Close())
	falsity, err := manager.MkFalse()
	spacerMust(t, err)
	spacerMust(t, recursiveContext.AddCover(-1, recursivePredicate, falsity))
	cover, err := recursiveContext.CoverDelta(-1, recursivePredicate)
	spacerMust(t, err)
	coverHandle, err := manager.RefGet(cover)
	spacerMust(t, err)
	if !manager.IsFalse(coverHandle) {
		t.Fatal("Spacer invariant cover", coverHandle)
	}
	spacerMust(t, manager.DelRef(&cover))
	if err = recursiveContext.AddCover(-2, recursivePredicate, falsity); err == nil {
		t.Fatal("Spacer accepted invalid cover level")
	}
	spacerMust(t, recursiveContext.Close())
	spacerMust(t, recursiveSet.Close())
	spacerMust(t, recursiveRule.Close())
	spacerMust(t, recursiveManager.Close())
	spacerMust(t, command.Close())
}

func TestSpacerContextInitializesOutputWithoutRetainedRules(t *testing.T) {
	command, err := cmd_context.NewStandaloneContext()
	spacerMust(t, err)
	t.Cleanup(func() { spacerMust(t, command.Close()) })
	manager, err := command.Manager()
	spacerMust(t, err)
	globals, err := command.GlobalParameters()
	spacerMust(t, err)
	rules, err := base.NewRuleManager(manager)
	spacerMust(t, err)
	t.Cleanup(func() { spacerMust(t, rules.Close()) })
	set, err := base.NewRuleSet(rules)
	spacerMust(t, err)
	t.Cleanup(func() { spacerMust(t, set.Close()) })
	boolean, err := manager.MkBoolSort()
	spacerMust(t, err)
	name, err := manager.Symbols.Intern("spacer-ruleless-output")
	spacerMust(t, err)
	declaration, err := manager.MkFuncDecl(name, nil, boolean, ast.ASTFuncDeclInfo{})
	spacerMust(t, err)
	spacerMust(t, rules.RegisterPredicate(declaration))
	spacerMust(t, set.SetOutputPredicate(declaration))
	valid, err := set.Finalize()
	spacerMust(t, err)
	if !valid {
		t.Fatal("ruleless output set is unstratified")
	}
	context, err := spacer.NewContext(manager, nil, 1, smtcore.SMTSolverConfig{Params: globals.Context, Globals: globals})
	spacerMust(t, err)
	t.Cleanup(func() { spacerMust(t, context.Close()) })
	spacerMust(t, context.InitializePredicates(set, nil))
	spacerMust(t, context.SetQuery(declaration))
	if status, solveErr := context.Solve(0, 1); solveErr != nil || status != lbool.False {
		t.Fatal("ruleless output query", status, solveErr)
	}
}
