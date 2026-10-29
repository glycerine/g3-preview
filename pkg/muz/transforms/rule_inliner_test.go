package transforms_test

// Source: src/muz/transforms/dl_mk_rule_inliner.cpp planning, rule resolution,
// protected outputs, and recursive-predicate exclusion.

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/muz/base"
	"github.com/glycerine/g3/pkg/muz/transforms"
)

func TestRuleInlinerEliminatesAcyclicChainIntoProtectedOutput(t *testing.T) {
	f := newConeFixture(t, "P", "Q", "Goal")
	f.add(t, f.applications["P"], "fact-P")
	f.edge(t, "P", "Q")
	f.edge(t, "Q", "Goal")
	coneMust(t, f.set.SetOutputPredicate(f.declarations["Goal"]))
	valid, err := f.set.Finalize()
	coneMust(t, err)
	if !valid {
		t.Fatal("inliner chain rejected")
	}
	transformer := base.NewRuleTransformer()
	t.Cleanup(func() { coneMust(t, transformer.Close()) })
	coneMust(t, transformer.Register(transforms.NewRuleInlinerPlugin(f.manager, f.rules, []ast.ASTHandle{f.declarations["Goal"]}, 0)))
	modified, err := transformer.Apply(f.set)
	coneMust(t, err)
	if !modified {
		t.Fatal("acyclic chain was not inlined")
	}
	if count, err := f.set.Len(); err != nil || count != 1 {
		t.Fatal("inlined rule count", count, err)
	}
	rule, err := f.set.Rule(0)
	coneMust(t, err)
	defer func() { coneMust(t, rule.Close()) }()
	head, err := rule.HeadDeclaration()
	coneMust(t, err)
	tails, err := rule.TailSize()
	coneMust(t, err)
	if head != f.declarations["Goal"] || tails != 0 {
		t.Fatal("inlined output rule", head, tails)
	}
}

func TestRuleInlinerUsesDisjointVariableBanks(t *testing.T) {
	f := newConeFixture(t)
	boolean, err := f.manager.MkBoolSort()
	coneMust(t, err)
	zero, err := f.manager.ArithmeticNumeralRat(big.NewRat(0, 1), true)
	coneMust(t, err)
	integer, err := f.manager.GetSort(zero)
	coneMust(t, err)
	declare := func(name string) ast.ASTHandle {
		symbol, e := f.manager.Symbols.Intern("inline-" + name)
		coneMust(t, e)
		declaration, e := f.manager.MkFuncDecl(symbol, []ast.ASTHandle{integer}, boolean, ast.ASTFuncDeclInfo{})
		coneMust(t, e)
		coneMust(t, f.rules.RegisterPredicate(declaration))
		return declaration
	}
	p := declare("P")
	q := declare("Q")
	pZero, err := f.manager.MkApp(p, []ast.ASTHandle{zero})
	coneMust(t, err)
	fact, err := f.rules.NewRule(pZero, nil, nil, "fact")
	coneMust(t, err)
	coneMust(t, f.set.Add(fact))
	coneMust(t, fact.Close())
	variable, err := f.manager.MkVar(0, integer)
	coneMust(t, err)
	pVariable, err := f.manager.MkApp(p, []ast.ASTHandle{variable})
	coneMust(t, err)
	qVariable, err := f.manager.MkApp(q, []ast.ASTHandle{variable})
	coneMust(t, err)
	edge, err := f.rules.NewRule(qVariable, []ast.ASTHandle{pVariable}, nil, "edge")
	coneMust(t, err)
	coneMust(t, f.set.Add(edge))
	coneMust(t, edge.Close())
	coneMust(t, f.set.SetOutputPredicate(q))
	valid, err := f.set.Finalize()
	coneMust(t, err)
	if !valid {
		t.Fatal("variable inliner fixture rejected")
	}
	transformer := base.NewRuleTransformer()
	t.Cleanup(func() { coneMust(t, transformer.Close()) })
	coneMust(t, transformer.Register(transforms.NewRuleInlinerPlugin(f.manager, f.rules, []ast.ASTHandle{q}, 0)))
	modified, err := transformer.Apply(f.set)
	coneMust(t, err)
	if !modified {
		t.Fatal("variable rule was not inlined")
	}
	rule, err := f.set.Rule(0)
	coneMust(t, err)
	defer func() { coneMust(t, rule.Close()) }()
	head, err := rule.Head()
	coneMust(t, err)
	argument, err := f.manager.GetArg(head, 0)
	coneMust(t, err)
	if headDeclaration, err := f.manager.GetDecl(head); err != nil || headDeclaration != q || argument != zero {
		t.Fatal("resolved variable head", headDeclaration, argument, err)
	}
}

func TestRuleInlinerPreservesRecursivePredicate(t *testing.T) {
	f := newConeFixture(t, "P", "Goal")
	f.add(t, f.applications["P"], "fact-P")
	f.edge(t, "P", "P")
	f.edge(t, "P", "Goal")
	coneMust(t, f.set.SetOutputPredicate(f.declarations["Goal"]))
	valid, err := f.set.Finalize()
	coneMust(t, err)
	if !valid {
		t.Fatal("recursive inliner fixture rejected")
	}
	transformer := base.NewRuleTransformer()
	t.Cleanup(func() { coneMust(t, transformer.Close()) })
	coneMust(t, transformer.Register(transforms.NewRuleInlinerPlugin(f.manager, f.rules, []ast.ASTHandle{f.declarations["Goal"]}, 0)))
	modified, err := transformer.Apply(f.set)
	coneMust(t, err)
	if modified {
		t.Fatal("recursive predicate was inlined")
	}
}
