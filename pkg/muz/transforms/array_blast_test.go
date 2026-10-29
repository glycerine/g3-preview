package transforms_test

// Source: src/muz/transforms/dl_mk_array_blast.cpp store substitution and
// pairwise Ackermann consistency constraints.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/muz/base"
	"github.com/glycerine/g3/pkg/muz/transforms"
)

func TestArrayBlastEliminatesStoreDefinitionAndSelectPairs(t *testing.T) {
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
	goalName, err := manager.Symbols.Intern("array-blast-goal")
	coneMust(t, err)
	goalDecl, err := manager.MkFuncDecl(goalName, nil, boolean, ast.ASTFuncDeclInfo{})
	coneMust(t, err)
	coneMust(t, rules.RegisterPredicate(goalDecl))
	goal, err := manager.MkApp(goalDecl, nil)
	coneMust(t, err)
	a, err := manager.MkVar(0, arraySort)
	coneMust(t, err)
	b, err := manager.MkVar(1, arraySort)
	coneMust(t, err)
	i, err := manager.MkVar(2, boolean)
	coneMust(t, err)
	j, err := manager.MkVar(3, boolean)
	coneMust(t, err)
	v, err := manager.MkVar(4, boolean)
	coneMust(t, err)
	x, err := manager.MkVar(5, boolean)
	coneMust(t, err)
	y, err := manager.MkVar(6, boolean)
	coneMust(t, err)
	family, err := manager.ArrayFamilyID()
	coneMust(t, err)
	store, err := manager.MkFamilyApp(family, ast.ASTArrayOpStore, nil, []ast.ASTHandle{b, i, v})
	coneMust(t, err)
	definition, err := manager.MkEq(a, store)
	coneMust(t, err)
	selectI, err := manager.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{b, i})
	coneMust(t, err)
	selectJ, err := manager.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{b, j})
	coneMust(t, err)
	first, err := manager.MkEq(selectI, x)
	coneMust(t, err)
	second, err := manager.MkEq(selectJ, y)
	coneMust(t, err)
	rule, err := rules.NewRule(goal, []ast.ASTHandle{definition, first, second}, nil, "array-blast")
	coneMust(t, err)
	coneMust(t, set.Add(rule))
	coneMust(t, rule.Close())
	coneMust(t, set.SetOutputPredicate(goalDecl))
	valid, err := set.Finalize()
	coneMust(t, err)
	if !valid {
		t.Fatal("array blast source is unstratified")
	}
	plugin, err := transforms.NewArrayBlastPlugin(manager, rules, globals.Context, globals, 35999)
	coneMust(t, err)
	transformer := base.NewRuleTransformer()
	coneMust(t, transformer.Register(plugin))
	t.Cleanup(func() { coneMust(t, transformer.Close()) })
	modified, err := transformer.Apply(set)
	coneMust(t, err)
	if !modified {
		t.Fatal("array blast did not modify the rule set")
	}
	transformed, err := set.Rule(0)
	coneMust(t, err)
	t.Cleanup(func() { coneMust(t, transformed.Close()) })
	tails, err := transformed.TailSize()
	coneMust(t, err)
	if tails == 0 {
		t.Fatal("Ackermann consistency constraint was lost")
	}
	for index := 0; index < tails; index++ {
		tail, _, getErr := transformed.Tail(index)
		coneMust(t, getErr)
		stack := []ast.ASTHandle{tail}
		for len(stack) != 0 {
			last := len(stack) - 1
			current := stack[last]
			stack = stack[:last]
			kind, kindErr := manager.GetKind(current)
			coneMust(t, kindErr)
			if kind != ast.ASTApp {
				continue
			}
			currentFamily, familyErr := manager.GetFamilyID(current)
			coneMust(t, familyErr)
			if currentFamily == family {
				op, opErr := manager.GetDeclKind(current)
				coneMust(t, opErr)
				if op == ast.ASTArrayOpSelect || op == ast.ASTArrayOpStore {
					t.Fatal("array operation survived blasting", op)
				}
			}
			count, countErr := manager.GetNumArgs(current)
			coneMust(t, countErr)
			for child := uint32(0); child < count; child++ {
				argument, argumentErr := manager.GetArg(current, child)
				coneMust(t, argumentErr)
				stack = append(stack, argument)
			}
		}
	}
}
