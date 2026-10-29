package transforms_test

// Source: src/muz/transforms/dl_mk_array_eq_rewrite.cpp representative
// preference, equality restoration, and select-over-store simplification.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/muz/base"
	"github.com/glycerine/g3/pkg/muz/transforms"
)

func TestArrayEqRewriteSelectsNonVariableRepresentativeAndExpandsSelectStore(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	coneMust(t, err)
	t.Cleanup(func() { coneMust(t, context.Close()) })
	manager, err := context.Manager()
	coneMust(t, err)
	globals, err := context.GlobalParameters()
	coneMust(t, err)
	rules, err := base.NewRuleManager(manager)
	coneMust(t, err)
	t.Cleanup(func() { coneMust(t, rules.Close()) })
	set, err := base.NewRuleSet(rules)
	coneMust(t, err)
	t.Cleanup(func() { coneMust(t, set.Close()) })

	boolean, err := manager.MkBoolSort()
	coneMust(t, err)
	arraySort, err := manager.MkArraySort(ast.SliceArguments([]ast.ASTHandle{boolean}), boolean)
	coneMust(t, err)
	left, err := manager.MkVar(0, arraySort)
	coneMust(t, err)
	right, err := manager.MkVar(1, arraySort)
	coneMust(t, err)
	truth, err := manager.MkTrue()
	coneMust(t, err)
	falsity, err := manager.MkFalse()
	coneMust(t, err)
	arrayFamily, err := manager.ArrayFamilyID()
	coneMust(t, err)
	stored, err := manager.MkFamilyApp(arrayFamily, ast.ASTArrayOpStore, nil, []ast.ASTHandle{right, truth, falsity})
	coneMust(t, err)
	selected, err := manager.MkFamilyApp(arrayFamily, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{left, truth})
	coneMust(t, err)
	arrayEquality, err := manager.MkEq(left, stored)
	coneMust(t, err)
	selectedEquality, err := manager.MkEq(selected, falsity)
	coneMust(t, err)

	name, err := manager.Symbols.Intern("array-eq-goal")
	coneMust(t, err)
	declaration, err := manager.MkFuncDecl(name, nil, boolean, ast.ASTFuncDeclInfo{})
	coneMust(t, err)
	coneMust(t, rules.RegisterPredicate(declaration))
	head, err := manager.MkApp(declaration, nil)
	coneMust(t, err)
	rule, err := rules.NewRule(head, []ast.ASTHandle{arrayEquality, selectedEquality}, nil, "array-eq")
	coneMust(t, err)
	coneMust(t, set.Add(rule))
	coneMust(t, rule.Close())
	valid, err := set.Finalize()
	coneMust(t, err)
	if !valid {
		t.Fatal("source rule set is not stratified")
	}

	plugin, err := transforms.NewArrayEqRewritePlugin(manager, rules, globals.Context, globals, 34998)
	coneMust(t, err)
	transformer := base.NewRuleTransformer()
	coneMust(t, transformer.Register(plugin))
	t.Cleanup(func() { coneMust(t, transformer.Close()) })
	modified, err := transformer.Apply(set)
	coneMust(t, err)
	if !modified {
		t.Fatal("array rewrite did not publish its source-style rebuilt rule set")
	}
	rewritten, err := set.Rule(0)
	coneMust(t, err)
	defer func() { coneMust(t, rewritten.Close()) }()
	size, err := rewritten.TailSize()
	coneMust(t, err)
	if size != 3 {
		t.Fatalf("rewritten tail size %d, want restored class equalities plus condition", size)
	}
	trueCount := 0
	for i := 0; i < size; i++ {
		tail, _, getErr := rewritten.Tail(i)
		coneMust(t, getErr)
		if manager.IsTrue(tail) {
			trueCount++
		}
	}
	if trueCount != 2 {
		t.Fatalf("rewritten true conditions %d, want select-store and reflexive equality simplified", trueCount)
	}
}
