// Source: src/util/util.h:307-315; src/ast/ast.cpp:451-490.
package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTSourceCompareArraysDeclarationAndApplicationOrder(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	astTestPin(t, m, b)
	u, e := m.MkUninterpretedSort(astTestSymbol(t, m, "other-sort"), nil)
	astTestMust(t, e)
	astTestPin(t, m, u)
	name := astTestSymbol(t, m, "array-equality")
	forward, e := m.MkFuncDecl(name, []ASTHandle{b, u}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, forward)
	again, e := m.MkFuncDecl(name, []ASTHandle{b, u}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, again)
	reverse, e := m.MkFuncDecl(name, []ASTHandle{u, b}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, reverse)
	if forward != again || forward == reverse {
		t.Fatal("source declaration domain comparison lost order or equality")
	}
	a, e := m.MkConst(astTestSymbol(t, m, "array-a"), b)
	astTestMust(t, e)
	astTestPin(t, m, a)
	c, e := m.MkConst(astTestSymbol(t, m, "array-c"), b)
	astTestMust(t, e)
	astTestPin(t, m, c)
	f, e := m.MkFuncDecl(astTestSymbol(t, m, "array-f"), []ASTHandle{b, b}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, f)
	left, e := m.MkApp(f, []ASTHandle{a, c})
	astTestMust(t, e)
	astTestPin(t, m, left)
	same, e := m.MkApp(f, []ASTHandle{a, c})
	astTestMust(t, e)
	astTestPin(t, m, same)
	right, e := m.MkApp(f, []ASTHandle{c, a})
	astTestMust(t, e)
	astTestPin(t, m, right)
	if left != same || left == right {
		t.Fatal("source application argument comparison lost order or equality")
	}
}

func TestASTSourceCompareArraysQuantifierOrder(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	astTestPin(t, m, b)
	u, e := m.MkUninterpretedSort(astTestSymbol(t, m, "quantifier-other-sort"), nil)
	astTestMust(t, e)
	astTestPin(t, m, u)
	a, e := m.MkConst(astTestSymbol(t, m, "quantifier-a"), b)
	astTestMust(t, e)
	astTestPin(t, m, a)
	c, e := m.MkConst(astTestSymbol(t, m, "quantifier-c"), b)
	astTestMust(t, e)
	astTestPin(t, m, c)
	p, e := m.MkFamilyApp(2, 0, nil, []ASTHandle{a})
	astTestMust(t, e)
	astTestPin(t, m, p)
	q, e := m.MkFamilyApp(2, 0, nil, []ASTHandle{c})
	astTestMust(t, e)
	astTestPin(t, m, q)
	x := astTestSymbol(t, m, "quantifier-x")
	y := astTestSymbol(t, m, "quantifier-y")
	base := ASTQuantifierSpec{
		Forall: true, Sorts: []ASTHandle{b, u}, Names: []usymbol.Symbol{x, y},
		Body: a, Patterns: []ASTHandle{p, q},
	}
	mk := func(spec ASTQuantifierSpec) ASTHandle {
		t.Helper()
		h, e := m.MkQuantifier(spec)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	first := mk(base)
	if first != mk(base) {
		t.Fatal("equal quantifier arrays did not intern together")
	}
	for _, tc := range []struct {
		name string
		spec ASTQuantifierSpec
	}{
		{"decl sorts", ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{u, b}, Names: base.Names, Body: a, Patterns: base.Patterns}},
		{"decl names", ASTQuantifierSpec{Forall: true, Sorts: base.Sorts, Names: []usymbol.Symbol{y, x}, Body: a, Patterns: base.Patterns}},
		{"patterns", ASTQuantifierSpec{Forall: true, Sorts: base.Sorts, Names: base.Names, Body: a, Patterns: []ASTHandle{q, p}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if first == mk(tc.spec) {
				t.Fatal("reordered quantifier array interned with original")
			}
		})
	}
	noPatterns := base
	noPatterns.Patterns = nil
	noPatterns.NoPatterns = []ASTHandle{a, c}
	firstNoPatterns := mk(noPatterns)
	if firstNoPatterns != mk(noPatterns) {
		t.Fatal("equal no-pattern arrays did not intern together")
	}
	noPatterns.NoPatterns = []ASTHandle{c, a}
	if firstNoPatterns == mk(noPatterns) {
		t.Fatal("reordered no-pattern array interned with original")
	}
}
