// Retained Boolean interning assertions from src/test/ast.cpp::tst1.
package ast

import (
	"testing"
)

func TestASTSourceInterningAssertions(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	astTestPin(t, m, b)
	a, e := m.MkConst(astTestSymbol(t, m, "a"), b)
	astTestMust(t, e)
	astTestPin(t, m, a)
	c, e := m.MkConst(astTestSymbol(t, m, "c"), b)
	astTestMust(t, e)
	astTestPin(t, m, c)
	i1, e := m.MkFamilyApp(m.BasicFamilyID(), ASTOpAnd, nil, []ASTHandle{a, c})
	astTestMust(t, e)
	astTestPin(t, m, i1)
	i2, e := m.MkFamilyApp(m.BasicFamilyID(), ASTOpAnd, nil, []ASTHandle{a, c})
	astTestMust(t, e)
	astTestPin(t, m, i2)
	i3, e := m.MkFamilyApp(m.BasicFamilyID(), ASTOpOr, nil, []ASTHandle{a, c})
	astTestMust(t, e)
	astTestPin(t, m, i3)
	if i1 != i2 {
		t.Fatal("source tst1: repeated AND must intern")
	}
	if i1 == i3 {
		t.Fatal("source tst1: AND and OR differ")
	}
}
