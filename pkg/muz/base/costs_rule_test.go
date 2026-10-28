package base_test

// Source: src/muz/base/dl_costs.cpp accounted_object parent-rule ownership and
// processed-cost propagation.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/muz/base"
)

func TestAccountedObjectPropagatesToRetainedRuleParent(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	ruleMust(t, err)
	t.Cleanup(func() { ruleMust(t, context.Close()) })
	manager, err := context.Manager()
	ruleMust(t, err)
	rules, err := base.NewRuleManager(manager)
	ruleMust(t, err)
	t.Cleanup(func() { ruleMust(t, rules.Close()) })
	boolean, err := manager.MkBoolSort()
	ruleMust(t, err)
	name, err := manager.Symbols.Intern("accounted-parent")
	ruleMust(t, err)
	declaration, err := manager.MkFuncDecl(name, nil, boolean, ast.ASTFuncDeclInfo{})
	ruleMust(t, err)
	ruleMust(t, rules.RegisterPredicate(declaration))
	head, err := manager.MkApp(declaration, nil)
	ruleMust(t, err)
	rule, err := rules.NewRule(head, nil, nil, "accounted")
	ruleMust(t, err)

	child := base.NewAccountedObject()
	ruleMust(t, child.SetAccountingParentRule(rule))
	parent, err := child.ParentRule()
	ruleMust(t, err)
	current, err := child.CurrentCosts()
	ruleMust(t, err)
	*current = base.Costs{Milliseconds: 3, Instructions: 2}
	ruleMust(t, rule.Close())
	ruleMust(t, child.ProcessCosts())
	if total, err := child.TotalCosts(); err != nil || total != (base.Costs{Milliseconds: 3, Instructions: 2}) {
		t.Fatal("child total", total, err)
	}
	if total, err := parent.AccountingCosts(); err != nil || total != (base.Costs{Milliseconds: 3, Instructions: 2}) {
		t.Fatal("parent propagation", total, err)
	}
	ruleMust(t, child.Close())
	if _, err := parent.AccountingCosts(); !errors.Is(err, base.ErrRuleOwner) {
		t.Fatal("parent rule did not release after child", err)
	}
}
