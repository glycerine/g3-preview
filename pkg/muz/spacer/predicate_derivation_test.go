package spacer_test

// Source: src/muz/spacer/spacer_context.cpp context::create_children.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/muz/base"
	"github.com/glycerine/g3/pkg/muz/spacer"
)

func TestSpacerPredicateCreatesAndAttachesProjectedDerivation(t *testing.T) {
	command, manager, owner := lemmaOwnerFixture(t)
	boolean, err := manager.MkBoolSort()
	spacerMust(t, err)
	name, err := manager.Symbols.Intern("spacer-create-children-p")
	spacerMust(t, err)
	predicate, err := manager.MkFuncDecl(name, nil, boolean, ast.ASTFuncDeclInfo{})
	spacerMust(t, err)
	transformer, err := spacer.NewRulePredicateTransformer(owner, predicate, spacer.PropSolverConfig{})
	spacerMust(t, err)
	ruleManager, err := base.NewRuleManager(manager)
	spacerMust(t, err)
	spacerMust(t, ruleManager.RegisterPredicate(predicate))
	head, err := manager.MkApp(predicate, nil)
	spacerMust(t, err)
	initial, err := ruleManager.NewRule(head, nil, nil, "initial")
	spacerMust(t, err)
	recursive, err := ruleManager.NewRule(head, []ast.ASTHandle{head}, []bool{false}, "recursive")
	spacerMust(t, err)
	spacerMust(t, transformer.InitializeRules(map[ast.ASTHandle]*spacer.PredicateTransformer{predicate: transformer}, []*base.Rule{initial, recursive}))
	spacerMust(t, transformer.InitReachFacts())
	truth, err := manager.MkTrue()
	spacerMust(t, err)
	obligation, err := transformer.NewProofObligation(nil, 2, 1, truth, nil)
	spacerMust(t, err)
	md, err := model.NewModel(manager)
	spacerMust(t, err)
	lastReach, found, err := transformer.LastReachCaseVariable()
	spacerMust(t, err)
	if !found {
		t.Fatal("missing initial Spacer reach-case variable")
	}
	originReach, err := owner.FormulaNewToOld(lastReach, 0, false)
	spacerMust(t, err)
	originReachHandle, err := manager.RefGet(originReach)
	spacerMust(t, err)
	originReachDeclaration, err := manager.GetDecl(originReachHandle)
	spacerMust(t, err)
	falsity, err := manager.MkFalse()
	spacerMust(t, err)
	spacerMust(t, md.Core().RegisterConstant(originReachDeclaration, falsity))
	child, err := transformer.CreateDerivationChild(obligation, recursive, md.Core(), []bool{false}, false)
	spacerMust(t, err)
	if child == nil {
		t.Fatal("Spacer create_children did not produce its first child")
	}
	parent, err := child.Parent()
	spacerMust(t, err)
	if parent != obligation {
		t.Fatal("Spacer create_children parent")
	}
	derivation, found, err := child.Derivation()
	spacerMust(t, err)
	if !found || derivation == nil {
		t.Fatal("Spacer create_children did not attach its derivation")
	}
	derivationParent, err := derivation.Parent()
	spacerMust(t, err)
	if derivationParent != obligation {
		t.Fatal("attached Spacer derivation tracks the wrong parent")
	}
	if level, err := child.Level(); err != nil || level != 1 {
		t.Fatal("Spacer create_children child level", level, err)
	}
	concrete, err := transformer.MakeReachFact(obligation, recursive, md.Core(), true, true)
	spacerMust(t, err)
	justifications, err := concrete.Justifications()
	spacerMust(t, err)
	if len(justifications) != 1 {
		t.Fatal("projected Spacer reach-fact justifications", len(justifications))
	}
	for _, justification := range justifications {
		spacerMust(t, justification.Close())
	}
	if initial, err := concrete.IsInitial(); err != nil || initial {
		t.Fatal("computed Spacer reach fact marked initial", initial, err)
	}
	if added, err := transformer.AddReachFact(concrete); err != nil || added {
		t.Fatal("duplicate projected Spacer reach fact", added, err)
	}
	spacerMust(t, concrete.Close())
	spacerMust(t, manager.DelRef(&originReach))
	spacerMust(t, md.Close())
	spacerMust(t, recursive.Close())
	spacerMust(t, initial.Close())
	spacerMust(t, ruleManager.Close())
	spacerMust(t, transformer.Close())
	spacerMust(t, owner.Close())
	spacerMust(t, command.Close())
}
