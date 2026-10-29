package transforms_test

// Source: src/muz/transforms/dl_mk_loop_counter.cpp and dl_mk_backwards.cpp
// live support transformations used by Karr invariant extraction.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/muz/base"
	"github.com/glycerine/g3/pkg/muz/transforms"
)

func karrSupportFixture(t *testing.T) (*ast.ASTManager, *base.RuleManager, *base.RuleSet, ast.ASTHandle, ast.ASTHandle) {
	t.Helper()
	context, err := cmd_context.NewStandaloneContext()
	coneMust(t, err)
	t.Cleanup(func() { coneMust(t, context.Close()) })
	manager, err := context.Manager()
	coneMust(t, err)
	rules, err := base.NewRuleManager(manager)
	coneMust(t, err)
	t.Cleanup(func() { coneMust(t, rules.Close()) })
	set, err := base.NewRuleSet(rules)
	coneMust(t, err)
	t.Cleanup(func() { coneMust(t, set.Close()) })
	boolean, err := manager.MkBoolSort()
	coneMust(t, err)
	name, err := manager.Symbols.Intern("karr-support-p")
	coneMust(t, err)
	declaration, err := manager.MkFuncDecl(name, nil, boolean, ast.ASTFuncDeclInfo{})
	coneMust(t, err)
	coneMust(t, rules.RegisterPredicate(declaration))
	application, err := manager.MkApp(declaration, nil)
	coneMust(t, err)
	return manager, rules, set, declaration, application
}

func TestLoopCounterAddsRecursiveSuccessorAndRevertsPredicates(t *testing.T) {
	manager, rules, set, declaration, application := karrSupportFixture(t)
	fact, err := rules.NewRule(application, nil, nil, "base")
	coneMust(t, err)
	recursive, err := rules.NewRule(application, []ast.ASTHandle{application}, nil, "step")
	coneMust(t, err)
	coneMust(t, set.Add(fact))
	coneMust(t, set.Add(recursive))
	coneMust(t, fact.Close())
	coneMust(t, recursive.Close())
	coneMust(t, set.SetOutputPredicate(declaration))
	valid, err := set.Finalize()
	coneMust(t, err)
	if !valid {
		t.Fatal("loop-counter source is unstratified")
	}
	plugin := transforms.NewLoopCounterPlugin(manager, rules, 1)
	transformed, err := plugin.Transform(set)
	coneMust(t, err)
	t.Cleanup(func() { coneMust(t, transformed.Close()); coneMust(t, plugin.Close()) })
	valid, err = transformed.Finalize()
	coneMust(t, err)
	if !valid {
		t.Fatal("loop-counter result is unstratified")
	}
	derived, err := transformed.DerivedPredicate(declaration)
	coneMust(t, err)
	if arity, getErr := manager.GetArity(derived); getErr != nil || arity != 1 {
		t.Fatal("counter predicate arity", arity, getErr)
	}
	step, err := transformed.Rule(1)
	coneMust(t, err)
	if tails, getErr := step.TailSize(); getErr != nil || tails != 2 {
		t.Fatal("recursive successor constraint", tails, getErr)
	}
	coneMust(t, step.Close())
	reverted, err := plugin.Revert(transformed)
	coneMust(t, err)
	t.Cleanup(func() { coneMust(t, reverted.Close()) })
	revertedRule, err := reverted.Rule(1)
	coneMust(t, err)
	head, err := revertedRule.HeadDeclaration()
	coneMust(t, err)
	coneMust(t, revertedRule.Close())
	if head != declaration {
		t.Fatal("loop-counter predicate was not reverted", head, declaration)
	}
}

func TestBackwardsBuildsOutputPredecessorAndSyntheticQuery(t *testing.T) {
	manager, rules, set, qDecl, q := karrSupportFixture(t)
	boolean, err := manager.MkBoolSort()
	coneMust(t, err)
	aName, err := manager.Symbols.Intern("karr-support-a")
	coneMust(t, err)
	aDecl, err := manager.MkFuncDecl(aName, nil, boolean, ast.ASTFuncDeclInfo{})
	coneMust(t, err)
	coneMust(t, rules.RegisterPredicate(aDecl))
	a, err := manager.MkApp(aDecl, nil)
	coneMust(t, err)
	fact, err := rules.NewRule(a, nil, nil, "base")
	coneMust(t, err)
	query, err := rules.NewRule(q, []ast.ASTHandle{a}, nil, "query")
	coneMust(t, err)
	coneMust(t, set.Add(fact))
	coneMust(t, set.Add(query))
	coneMust(t, fact.Close())
	coneMust(t, query.Close())
	coneMust(t, set.SetOutputPredicate(qDecl))
	valid, err := set.Finalize()
	coneMust(t, err)
	if !valid {
		t.Fatal("backwards source is unstratified")
	}
	plugin := transforms.NewBackwardsPlugin(manager, rules, 1)
	result, err := plugin.Transform(set)
	coneMust(t, err)
	t.Cleanup(func() { coneMust(t, result.Close()); coneMust(t, plugin.Close()) })
	if count, getErr := result.Len(); getErr != nil || count != 2 {
		t.Fatal("backwards rule count", count, getErr)
	}
	first, err := result.Rule(0)
	coneMust(t, err)
	firstHead, err := first.HeadDeclaration()
	coneMust(t, err)
	coneMust(t, first.Close())
	if firstHead == qDecl {
		t.Fatal("backwards output rule retained its forward head")
	}
	outputs, err := result.OutputPredicates()
	coneMust(t, err)
	if len(outputs) != 1 || outputs[0] == qDecl {
		t.Fatal("missing synthetic backwards query", outputs)
	}
}
