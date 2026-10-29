package transforms_test

// Source: src/muz/transforms/dl_mk_interp_tail_simplifier.cpp interpreted-tail
// conjunction rewriting, infeasible-rule removal, and equivalence propagation.

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	"github.com/glycerine/g3/pkg/muz/base"
	"github.com/glycerine/g3/pkg/muz/transforms"
)

type interpTailFixture struct {
	manager *ast.ASTManager
	rules   *base.RuleManager
	set     *base.RuleSet
	plugin  *transforms.InterpTailSimplifierPlugin
}

func newInterpTailFixture(t *testing.T) *interpTailFixture {
	t.Helper()
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
	plugin, err := transforms.NewInterpTailSimplifierPlugin(manager, rules, globals.Context, globals, 0)
	coneMust(t, err)
	t.Cleanup(func() { coneMust(t, plugin.Close()) })
	return &interpTailFixture{manager: manager, rules: rules, set: set, plugin: plugin}
}

func (f *interpTailFixture) predicate(t *testing.T, name string, domain []ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	boolean, err := f.manager.MkBoolSort()
	coneMust(t, err)
	symbol, err := f.manager.Symbols.Intern("tail-" + name)
	coneMust(t, err)
	declaration, err := f.manager.MkFuncDecl(symbol, domain, boolean, ast.ASTFuncDeclInfo{})
	coneMust(t, err)
	coneMust(t, f.rules.RegisterPredicate(declaration))
	return declaration
}

func TestInterpTailSimplifierPropagatesEqualityAndRemovesTruth(t *testing.T) {
	f := newInterpTailFixture(t)
	one, err := f.manager.ArithmeticNumeralRat(big.NewRat(1, 1), true)
	coneMust(t, err)
	integer, err := f.manager.GetSort(one)
	coneMust(t, err)
	variable, err := f.manager.MkVar(0, integer)
	coneMust(t, err)
	predicate := f.predicate(t, "P", []ast.ASTHandle{integer})
	head, err := f.manager.MkApp(predicate, []ast.ASTHandle{variable})
	coneMust(t, err)
	equality, err := f.manager.MkEq(variable, one)
	coneMust(t, err)
	truth, err := f.manager.MkTrue()
	coneMust(t, err)
	rule, err := f.rules.NewRule(head, []ast.ASTHandle{equality, truth}, nil, "propagate")
	coneMust(t, err)
	coneMust(t, f.set.Add(rule))
	coneMust(t, rule.Close())
	coneMust(t, f.set.SetOutputPredicate(predicate))
	valid, err := f.set.Finalize()
	coneMust(t, err)
	if !valid {
		t.Fatal("tail simplifier fixture rejected")
	}
	transformer := base.NewRuleTransformer()
	t.Cleanup(func() { coneMust(t, transformer.Close()) })
	coneMust(t, transformer.Register(f.plugin))
	modified, err := transformer.Apply(f.set)
	coneMust(t, err)
	if !modified {
		t.Fatal("interpreted tail was not simplified")
	}
	result, err := f.set.Rule(0)
	coneMust(t, err)
	defer func() { coneMust(t, result.Close()) }()
	resultHead, err := result.Head()
	coneMust(t, err)
	argument, err := f.manager.GetArg(resultHead, 0)
	coneMust(t, err)
	if argument != one {
		t.Fatal("head equality was not propagated", argument, one)
	}
	if tails, getErr := result.TailSize(); getErr != nil || tails != 0 {
		t.Fatal("simplified equality was not removed", tails, getErr)
	}
}

func TestInterpTailSimplifierDropsFalseRule(t *testing.T) {
	f := newInterpTailFixture(t)
	predicate := f.predicate(t, "P", nil)
	head, err := f.manager.MkApp(predicate, nil)
	coneMust(t, err)
	falsehood, err := f.manager.MkFalse()
	coneMust(t, err)
	rule, err := f.rules.NewRule(head, []ast.ASTHandle{falsehood}, nil, "false")
	coneMust(t, err)
	coneMust(t, f.set.Add(rule))
	coneMust(t, rule.Close())
	coneMust(t, f.set.SetOutputPredicate(predicate))
	valid, err := f.set.Finalize()
	coneMust(t, err)
	if !valid {
		t.Fatal("false-tail fixture rejected")
	}
	transformer := base.NewRuleTransformer()
	t.Cleanup(func() { coneMust(t, transformer.Close()) })
	coneMust(t, transformer.Register(f.plugin))
	modified, err := transformer.Apply(f.set)
	coneMust(t, err)
	if !modified {
		t.Fatal("false rule was not removed")
	}
	if count, getErr := f.set.Len(); getErr != nil || count != 0 {
		t.Fatal("false rule remains", count, getErr)
	}
}
