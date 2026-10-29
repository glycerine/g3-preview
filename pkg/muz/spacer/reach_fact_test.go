package spacer_test

// Source: src/muz/spacer/spacer_context.h reach_fact ownership and queries.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/muz/base"
	"github.com/glycerine/g3/pkg/muz/spacer"
)

func TestSpacerReachFactRetainsRuleAuxiliariesAndJustifications(t *testing.T) {
	command, manager, owner := lemmaOwnerFixture(t)
	ruleManager, err := base.NewRuleManager(manager)
	spacerMust(t, err)
	boolean, err := manager.MkBoolSort()
	spacerMust(t, err)
	name, err := manager.Symbols.Intern("spacer-reach-p")
	spacerMust(t, err)
	predicate, err := manager.MkFuncDecl(name, nil, boolean, ast.ASTFuncDeclInfo{})
	spacerMust(t, err)
	spacerMust(t, ruleManager.RegisterPredicate(predicate))
	head, err := manager.MkApp(predicate, nil)
	spacerMust(t, err)
	rule, err := ruleManager.NewRule(head, nil, nil, "reach")
	spacerMust(t, err)
	auxiliary := spacerBool(t, manager, "spacer-reach-aux")
	first, err := spacer.NewReachFact(manager, rule, head, []ast.ASTHandle{auxiliary}, true)
	spacerMust(t, err)
	second, err := spacer.NewReachFact(manager, rule, auxiliary, nil, false)
	spacerMust(t, err)
	spacerMust(t, second.AddJustification(first))
	spacerMust(t, rule.Close())
	if initial, err := first.IsInitial(); err != nil || !initial {
		t.Fatal("initial Spacer reach fact", initial, err)
	}
	auxiliaries, err := first.Auxiliaries()
	spacerMust(t, err)
	if len(auxiliaries) != 1 || auxiliaries[0] != auxiliary {
		t.Fatal("Spacer reach fact auxiliaries", auxiliaries)
	}
	justifications, err := second.Justifications()
	spacerMust(t, err)
	if len(justifications) != 1 {
		t.Fatal("Spacer reach fact justifications", len(justifications))
	}
	fact, err := justifications[0].Fact()
	spacerMust(t, err)
	if fact != head {
		t.Fatal("Spacer reach fact justification", fact, head)
	}
	spacerMust(t, justifications[0].Close())
	spacerMust(t, first.Close())
	spacerMust(t, second.Close())
	spacerMust(t, ruleManager.Close())
	spacerMust(t, owner.Close())
	spacerMust(t, command.Close())
}
