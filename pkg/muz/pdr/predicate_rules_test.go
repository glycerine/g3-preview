package pdr_test

// Source: src/muz/pdr/pdr_context.cpp predicate-transformer rule grounding,
// initial-state construction, transition tagging, and variable substitution.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/muz/base"
	"github.com/glycerine/g3/pkg/muz/pdr"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestPredicateTransformerNormalizesRulesIntoExecutableSystem(t *testing.T) {
	command, manager, owner := pdrManagerFixture(t, 2)
	boolean, err := manager.MkBoolSort()
	pdrMust(t, err)
	predicate := pdrFunction(t, manager, "rule_state", []ast.ASTHandle{boolean}, boolean)
	ruleManager, err := base.NewRuleManager(manager)
	pdrMust(t, err)
	pdrMust(t, ruleManager.RegisterPredicate(predicate))
	variable, err := manager.MkVar(0, boolean)
	pdrMust(t, err)
	head, err := manager.MkApp(predicate, []ast.ASTHandle{variable})
	pdrMust(t, err)
	truth, err := manager.MkTrue()
	pdrMust(t, err)
	initialConstraint, err := manager.MkEq(variable, truth)
	pdrMust(t, err)
	initialRule, err := ruleManager.NewRule(head, []ast.ASTHandle{initialConstraint}, nil, "initial")
	pdrMust(t, err)
	recursiveTail, err := manager.MkApp(predicate, []ast.ASTHandle{variable})
	pdrMust(t, err)
	recursiveRule, err := ruleManager.NewRule(head, []ast.ASTHandle{recursiveTail}, nil, "step")
	pdrMust(t, err)

	transformer, err := pdr.NewPredicateTransformer(owner, predicate, pdr.ReachableConstraintCache)
	pdrMust(t, err)
	pdrMust(t, transformer.InitializeRules(
		map[ast.ASTHandle]*pdr.PredicateTransformer{predicate: transformer},
		[]*base.Rule{initialRule, recursiveRule},
	))
	pdrMust(t, initialRule.Close())
	pdrMust(t, recursiveRule.Close())

	initial, err := transformer.InitialState()
	pdrMust(t, err)
	transition, err := transformer.Transition()
	pdrMust(t, err)
	for name, formula := range map[string]ast.ASTHandle{"initial": initial, "transition": transition} {
		ground, x := manager.IsGround(formula)
		pdrMust(t, x)
		if !ground {
			t.Fatal(name, "formula retained free variables")
		}
	}
	signature, err := transformer.Signature()
	pdrMust(t, err)
	newDeclaration, err := owner.OldToNew(signature[0], 0)
	pdrMust(t, err)
	newState, err := manager.MkApp(newDeclaration, nil)
	pdrMust(t, err)
	oldState, err := manager.MkApp(signature[0], nil)
	pdrMust(t, err)
	notNew, err := manager.MkNot(newState)
	pdrMust(t, err)
	notOld, err := manager.MkNot(oldState)
	pdrMust(t, err)

	result, err := transformer.CheckReachable(0, notNew, false, false)
	pdrMust(t, err)
	if result.Status != lbool.False {
		t.Fatal("initial rule did not force the new-state argument", result.Status)
	}
	newFromInitial, err := manager.MkAnd(newState, notOld)
	pdrMust(t, err)
	result, err = transformer.CheckReachable(1, newFromInitial, false, false)
	pdrMust(t, err)
	if result.Status != lbool.True {
		t.Fatal("initial rule was not available in transition relation", result.Status)
	}
	notNew, err = manager.MkNot(newState)
	pdrMust(t, err)
	badStep, err := manager.MkAnd(notNew, oldState)
	pdrMust(t, err)
	result, err = transformer.CheckReachable(1, badStep, true, false)
	pdrMust(t, err)
	if result.Status != lbool.False {
		t.Fatal("recursive rule did not preserve its argument", result.Status)
	}

	rules, err := transformer.Rules()
	pdrMust(t, err)
	if len(rules) != 2 {
		t.Fatal("retained normalized rules", len(rules))
	}
	for _, rule := range rules {
		pdrMust(t, rule.Close())
	}

	pdrMust(t, transformer.Close())
	pdrMust(t, ruleManager.Close())
	pdrMust(t, owner.Close())
	pdrMust(t, command.Close())
}

func TestPredicateTransformerCouplesPredecessorFramesByRuleTag(t *testing.T) {
	command, manager, owner := pdrManagerFixture(t, 3)
	boolean, err := manager.MkBoolSort()
	pdrMust(t, err)
	pDeclaration := pdrFunction(t, manager, "predecessor", []ast.ASTHandle{boolean}, boolean)
	qDeclaration := pdrFunction(t, manager, "consumer", []ast.ASTHandle{boolean}, boolean)
	ruleManager, err := base.NewRuleManager(manager)
	pdrMust(t, err)
	pdrMust(t, ruleManager.RegisterPredicate(pDeclaration))
	pdrMust(t, ruleManager.RegisterPredicate(qDeclaration))
	variable, err := manager.MkVar(0, boolean)
	pdrMust(t, err)
	truth, err := manager.MkTrue()
	pdrMust(t, err)
	pHead, err := manager.MkApp(pDeclaration, []ast.ASTHandle{variable})
	pdrMust(t, err)
	pConstraint, err := manager.MkEq(variable, truth)
	pdrMust(t, err)
	pRule, err := ruleManager.NewRule(pHead, []ast.ASTHandle{pConstraint}, nil, "p-initial")
	pdrMust(t, err)
	qHead, err := manager.MkApp(qDeclaration, []ast.ASTHandle{variable})
	pdrMust(t, err)
	pTail, err := manager.MkApp(pDeclaration, []ast.ASTHandle{variable})
	pdrMust(t, err)
	qRule, err := ruleManager.NewRule(qHead, []ast.ASTHandle{pTail}, nil, "q-step")
	pdrMust(t, err)

	pTransformer, err := pdr.NewPredicateTransformer(owner, pDeclaration, pdr.ReachableNoCache)
	pdrMust(t, err)
	qTransformer, err := pdr.NewPredicateTransformer(owner, qDeclaration, pdr.ReachableNoCache)
	pdrMust(t, err)
	transformers := map[ast.ASTHandle]*pdr.PredicateTransformer{
		pDeclaration: pTransformer,
		qDeclaration: qTransformer,
	}
	pdrMust(t, pTransformer.InitializeRules(transformers, []*base.Rule{pRule}))
	pdrMust(t, qTransformer.InitializeRules(transformers, []*base.Rule{qRule}))
	pdrMust(t, pRule.Close())
	pdrMust(t, qRule.Close())

	pSignature, err := pTransformer.Signature()
	pdrMust(t, err)
	pNewDeclaration, err := owner.OldToNew(pSignature[0], 0)
	pdrMust(t, err)
	pNew, err := manager.MkApp(pNewDeclaration, nil)
	pdrMust(t, err)
	qSignature, err := qTransformer.Signature()
	pdrMust(t, err)
	qNewDeclaration, err := owner.OldToNew(qSignature[0], 0)
	pdrMust(t, err)
	qNew, err := manager.MkApp(qNewDeclaration, nil)
	pdrMust(t, err)
	notQ, err := manager.MkNot(qNew)
	pdrMust(t, err)

	before, err := qTransformer.CheckReachable(1, notQ, false, false)
	pdrMust(t, err)
	if before.Status != lbool.True {
		t.Fatal("predecessor property unexpectedly active before insertion", before.Status)
	}
	changed, err := pTransformer.AddProperty(pNew, 0)
	pdrMust(t, err)
	if !changed {
		t.Fatal("predecessor property was ignored")
	}
	notQ, err = manager.MkNot(qNew)
	pdrMust(t, err)
	after, err := qTransformer.CheckReachable(1, notQ, false, false)
	pdrMust(t, err)
	if after.Status != lbool.False {
		t.Fatal("tagged predecessor frame was not coupled to consumer", after.Status)
	}
	premises, err := qTransformer.Premises(0)
	pdrMust(t, err)
	if len(premises) < 3 {
		t.Fatal("predecessor premises were omitted", len(premises))
	}
	propagation, err := qTransformer.PropagationFormula(1)
	pdrMust(t, err)
	if ground, x := manager.IsGround(propagation); x != nil || !ground {
		t.Fatal("propagation formula is not ground", ground, x)
	}

	pdrMust(t, qTransformer.Close())
	pdrMust(t, pTransformer.Close())
	pdrMust(t, ruleManager.Close())
	pdrMust(t, owner.Close())
	pdrMust(t, command.Close())
}
