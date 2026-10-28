package base_test

// Source: src/muz/base/dl_rule.cpp rule_manager::mk ordering, negation,
// validation, and ownership branches.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/muz/base"
)

func ruleMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRuleManagerClassifiesAndOwnsNativeTailLayout(t *testing.T) {
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
	arithmetic, err := manager.ArithmeticFamilyID()
	ruleMust(t, err)
	integer, err := manager.MkFamilySort(arithmetic, 1, nil)
	ruleMust(t, err)
	x, err := manager.MkVar(0, integer)
	ruleMust(t, err)
	declarations := make([]ast.ASTHandle, 3)
	applications := make([]ast.ASTHandle, 3)
	for i, name := range []string{"rule-P", "rule-Q", "rule-R"} {
		symbol, e := manager.Symbols.Intern(name)
		ruleMust(t, e)
		declarations[i], e = manager.MkFuncDecl(symbol, []ast.ASTHandle{integer}, boolean, ast.ASTFuncDeclInfo{})
		ruleMust(t, e)
		applications[i], e = manager.MkApp(declarations[i], []ast.ASTHandle{x})
		ruleMust(t, e)
		ruleMust(t, rules.RegisterPredicate(declarations[i]))
	}
	zero, err := manager.ArithmeticIntegerNumeralText("0")
	ruleMust(t, err)
	ten, err := manager.ArithmeticIntegerNumeralText("10")
	ruleMust(t, err)
	lower, err := manager.MkArithmeticGE(x, zero)
	ruleMust(t, err)
	upper, err := manager.MkArithmeticLE(x, ten)
	ruleMust(t, err)

	rule, err := rules.NewRule(applications[0], []ast.ASTHandle{applications[1], lower, applications[2], upper}, []bool{true, false, false, true}, "native-order")
	ruleMust(t, err)
	t.Cleanup(func() { ruleMust(t, rule.Close()) })
	if size, err := rule.TailSize(); err != nil || size != 4 {
		t.Fatal("tail size", size, err)
	}
	if size, err := rule.PositiveTailSize(); err != nil || size != 1 {
		t.Fatal("positive predicate count", size, err)
	}
	if size, err := rule.UninterpretedTailSize(); err != nil || size != 2 {
		t.Fatal("predicate count", size, err)
	}
	want := []ast.ASTHandle{applications[2], applications[1]}
	for i := range want {
		got, negated, e := rule.Tail(i)
		ruleMust(t, e)
		if got != want[i] || negated != (i == 1) {
			t.Fatal("predicate tail order", i, got, negated, want[i])
		}
	}
	negatedUpper, err := manager.MkNot(upper)
	ruleMust(t, err)
	for i, expected := range []ast.ASTHandle{negatedUpper, lower} {
		got, negated, e := rule.Tail(i + 2)
		ruleMust(t, e)
		if got != expected || negated {
			t.Fatal("interpreted tail order", i, got, negated, expected)
		}
	}
	if found, err := rule.IsInTail(declarations[1], false); err != nil || !found {
		t.Fatal("negative predicate lookup", found, err)
	}
	if found, err := rule.IsInTail(declarations[1], true); err != nil || found {
		t.Fatal("positive-only predicate lookup", found, err)
	}
	if negation, err := rule.HasNegation(); err != nil || !negation {
		t.Fatal("negation flag", negation, err)
	}
	if name, err := rule.Name(); err != nil || name != "native-order" {
		t.Fatal("rule name", name, err)
	}
	if err := rule.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := rule.Head(); !errors.Is(err, base.ErrRuleOwner) {
		t.Fatal("closed rule accepted", err)
	}
}

func TestRuleManagerRejectsUnregisteredAndCompoundHeads(t *testing.T) {
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
	symbol, err := manager.Symbols.Intern("rule-invalid")
	ruleMust(t, err)
	declaration, err := manager.MkFuncDecl(symbol, []ast.ASTHandle{boolean}, boolean, ast.ASTFuncDeclInfo{})
	ruleMust(t, err)
	truth, err := manager.MkTrue()
	ruleMust(t, err)
	head, err := manager.MkApp(declaration, []ast.ASTHandle{truth})
	ruleMust(t, err)
	if _, err = rules.NewRule(head, nil, nil, ""); err == nil {
		t.Fatal("unregistered predicate head accepted")
	}
	ruleMust(t, rules.RegisterPredicate(declaration))
	notTruth, err := manager.MkNot(truth)
	ruleMust(t, err)
	compound, err := manager.MkApp(declaration, []ast.ASTHandle{notTruth})
	ruleMust(t, err)
	if _, err = rules.NewRule(compound, nil, nil, ""); err == nil {
		t.Fatal("compound predicate argument accepted in head")
	}
}
