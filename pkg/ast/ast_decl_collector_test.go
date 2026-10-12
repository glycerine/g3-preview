package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTDeclCollectorVisitsRealGraphAndOrdersSorts(t *testing.T) {
	m := astTestManager(t)
	a, e := m.MkUninterpretedSort(astTestSymbol(t, m, "A"), nil)
	astTestMust(t, e)
	astTestPin(t, m, a)
	pa, e := m.NewParameterAST(a)
	astTestMust(t, e)
	b, e := m.MkUninterpretedSort(astTestSymbol(t, m, "B"), []ASTParameter{pa})
	astTestMust(t, e)
	astTestPin(t, m, b)
	astTestMust(t, m.DelParameter(&pa))
	boolSort, e := m.MkBoolSort()
	astTestMust(t, e)
	f, e := m.MkFuncDecl(astTestSymbol(t, m, "f"), []ASTHandle{a}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, f)
	p, e := m.MkFuncDecl(astTestSymbol(t, m, "p"), []ASTHandle{b}, boolSort, ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, p)
	x, e := m.MkVar(0, a)
	astTestMust(t, e)
	astTestPin(t, m, x)
	fx, e := m.MkApp(f, []ASTHandle{x})
	astTestMust(t, e)
	astTestPin(t, m, fx)
	root, e := m.MkApp(p, []ASTHandle{fx})
	astTestMust(t, e)
	astTestPin(t, m, root)

	c := newASTDeclCollector(m, true)
	astTestMust(t, c.visit(root))
	if !equalSortHandles(c.sorts, []ASTHandle{b, a}) || !equalSortHandles(c.decls, []ASTHandle{f}) || !equalSortHandles(c.predicates, []ASTHandle{p}) {
		t.Fatalf("source LIFO discovery: sorts=%v decls=%v preds=%v", c.sorts, c.decls, c.predicates)
	}
	astTestMust(t, c.orderDeps())
	if !equalSortHandles(c.sorts, []ASTHandle{a, b}) {
		t.Fatalf("dependency order = %v, want A before B", c.sorts)
	}
	astTestMust(t, c.visit(root))
	if len(c.sorts) != 2 || len(c.decls) != 1 || len(c.predicates) != 1 {
		t.Fatal("visited graph must not be collected twice")
	}

	together := newASTDeclCollector(m, false)
	astTestMust(t, together.visit(root))
	if !equalSortHandles(together.decls, []ASTHandle{p, f}) || len(together.predicates) != 0 {
		t.Fatalf("predicate splitting disabled: decls=%v preds=%v", together.decls, together.predicates)
	}
	q, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{b}, Names: []usymbol.Symbol{astTestSymbol(t, m, "v")}, Body: root})
	astTestMust(t, e)
	astTestPin(t, m, q)
	quantified := newASTDeclCollector(m, true)
	astTestMust(t, quantified.visit(q))
	astTestMust(t, quantified.orderDeps())
	if !equalSortHandles(quantified.sorts, []ASTHandle{a, b}) || !equalSortHandles(quantified.predicates, []ASTHandle{p}) {
		t.Fatalf("quantifier body and declaration sorts: sorts=%v preds=%v", quantified.sorts, quantified.predicates)
	}
}
