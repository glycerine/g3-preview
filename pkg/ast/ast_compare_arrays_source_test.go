// Source: src/util/util.h:307-315; src/ast/ast.cpp:451-463.
package ast

import "testing"

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
