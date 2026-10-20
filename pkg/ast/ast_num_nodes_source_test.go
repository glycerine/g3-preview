// Source: src/ast/for_each_ast.cpp:20-33; src/ast/for_each_ast.h:42-134.
package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTGetNumNodesDefaultTraversalAndSharedDeduplication(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	x, e := m.MkVar(0, b)
	astTestMust(t, e)
	f, e := m.MkFuncDecl(astTestSymbol(t, m, "node-count-f"), []ASTHandle{b}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	fx, e := m.MkApp(f, []ASTHandle{x})
	astTestMust(t, e)
	root, e := m.MkAnd(fx, fx)
	astTestMust(t, e)
	n, e := m.GetNumNodes(root)
	astTestMust(t, e)
	// root, basic and decl, fx, f, Bool and x. The repeated fx and shared Bool
	// domain/range are each counted once.
	if n != 6 {
		t.Fatal("source distinct node count", n)
	}
}

func TestASTGetNumNodesQuantifierChildrenAndIgnoredParameters(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	v, e := m.MkVar(0, b)
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "node-count-pattern"), []ASTHandle{b}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	a, e := m.MkApp(d, []ASTHandle{v})
	astTestMust(t, e)
	p, e := m.MkFamilyApp(2, 0, nil, []ASTHandle{a})
	astTestMust(t, e)
	q, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{b}, Names: []usymbol.Symbol{astTestSymbol(t, m, "x")}, Body: a, Patterns: []ASTHandle{p}})
	astTestMust(t, e)
	n, e := m.GetNumNodes(q)
	astTestMust(t, e)
	if n != 7 { // q, p/pattern-decl, a/d, v and Bool through d.
		t.Fatal("quantifier child traversal", n)
	}
	qNoPattern, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{b}, Names: []usymbol.Symbol{astTestSymbol(t, m, "y")}, Body: a, NoPatterns: []ASTHandle{v}})
	astTestMust(t, e)
	n, e = m.GetNumNodes(qNoPattern)
	astTestMust(t, e)
	if n != 5 { // q, a, d, v and Bool.
		t.Fatal("quantifier no-pattern traversal", n)
	}
	param, e := m.NewParameterAST(q)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelParameter(&param)) }()
	s, e := m.MkUninterpretedSort(astTestSymbol(t, m, "parameter-root"), []ASTParameter{param})
	astTestMust(t, e)
	n, e = m.GetNumNodes(s)
	astTestMust(t, e)
	if n != 1 {
		t.Fatal("default traversal unexpectedly visited AST parameter", n)
	}
}

func TestASTGetNumNodesDeepIterativeAndGuards(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	x, e := m.MkVar(0, b)
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "node-count-chain"), []ASTHandle{b}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	root := x
	for i := 0; i < 4096; i++ {
		root, e = m.MkApp(d, []ASTHandle{root})
		astTestMust(t, e)
	}
	n, e := m.GetNumNodes(root)
	astTestMust(t, e)
	if n != 4099 { // 4096 apps, variable, declaration and shared Bool.
		t.Fatal("deep source node count", n)
	}
	if _, e = m.GetNumNodes(ASTHandle{}); e == nil {
		t.Fatal("null root accepted")
	}
}
