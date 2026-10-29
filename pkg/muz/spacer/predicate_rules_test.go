package spacer_test

// Source: src/muz/spacer/spacer_context.cpp predicate rule initialization.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/muz/base"
	"github.com/glycerine/g3/pkg/muz/spacer"
)

func TestSpacerPredicateRulesBuildTaggedInitialAndTransitionSystem(t *testing.T) {
	command, manager, owner := lemmaOwnerFixture(t)
	boolean, err := manager.MkBoolSort()
	spacerMust(t, err)
	pName, err := manager.Symbols.Intern("spacer-rule-p")
	spacerMust(t, err)
	pDecl, err := manager.MkFuncDecl(pName, []ast.ASTHandle{boolean}, boolean, ast.ASTFuncDeclInfo{})
	spacerMust(t, err)
	transformer, err := spacer.NewRulePredicateTransformer(owner, pDecl, spacer.PropSolverConfig{})
	spacerMust(t, err)
	ruleManager, err := base.NewRuleManager(manager)
	spacerMust(t, err)
	spacerMust(t, ruleManager.RegisterPredicate(pDecl))
	truth, err := manager.MkTrue()
	spacerMust(t, err)
	initialHead, err := manager.MkApp(pDecl, []ast.ASTHandle{truth})
	spacerMust(t, err)
	condition := spacerBool(t, manager, "spacer-rule-condition")
	one, err := manager.ArithmeticIntegerNumeralText("1")
	spacerMust(t, err)
	two, err := manager.ArithmeticIntegerNumeralText("2")
	spacerMust(t, err)
	choice, err := manager.MkIte(condition, one, two)
	spacerMust(t, err)
	constraint, err := manager.MkEq(choice, one)
	spacerMust(t, err)
	initialRule, err := ruleManager.NewRule(initialHead, []ast.ASTHandle{constraint}, []bool{false}, "initial")
	spacerMust(t, err)
	variable, err := manager.MkVar(0, boolean)
	spacerMust(t, err)
	recursiveHead, err := manager.MkApp(pDecl, []ast.ASTHandle{variable})
	spacerMust(t, err)
	recursiveTail, err := manager.MkApp(pDecl, []ast.ASTHandle{variable})
	spacerMust(t, err)
	recursiveRule, err := ruleManager.NewRule(recursiveHead, []ast.ASTHandle{recursiveTail}, nil, "recursive")
	spacerMust(t, err)
	spacerMust(t, transformer.InitializeRules(map[ast.ASTHandle]*spacer.PredicateTransformer{pDecl: transformer}, []*base.Rule{initialRule, recursiveRule}))
	rules, err := transformer.Rules()
	spacerMust(t, err)
	if len(rules) != 2 {
		t.Fatal("normalized Spacer rules", len(rules))
	}
	for _, rule := range rules {
		spacerMust(t, rule.Close())
	}
	initial, err := transformer.InitialState()
	spacerMust(t, err)
	transition, err := transformer.Transition()
	spacerMust(t, err)
	if manager.IsTrue(initial) || manager.IsFalse(transition) {
		t.Fatal("tagged Spacer system", initial, transition)
	}
	features, err := ast.NewStaticFeatures(manager)
	spacerMust(t, err)
	spacerMust(t, features.Collect(transition))
	if features.NumITETerms != 0 {
		t.Fatal("Spacer rule normalization retained term ITE", features.NumITETerms)
	}
	spacerMust(t, features.Close())
	if all, err := transformer.AllInitial(); err != nil || all {
		t.Fatal("mixed initial/recursive classification", all, err)
	}
	signature, err := transformer.Signature()
	spacerMust(t, err)
	current, err := owner.OldToNew(signature[0], 0)
	spacerMust(t, err)
	state, err := manager.MkApp(current, nil)
	spacerMust(t, err)
	if added, err := transformer.AddLemma(state, 0); err != nil || !added {
		t.Fatal("self-predecessor lemma propagation", added, err)
	}
	spacerMust(t, recursiveRule.Close())
	spacerMust(t, initialRule.Close())
	spacerMust(t, transformer.Close())
	spacerMust(t, ruleManager.Close())
	spacerMust(t, owner.Close())
	spacerMust(t, command.Close())
}
