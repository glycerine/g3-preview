package ast

// Source: src/ast/ast.h:1984,2008-2018,2031-2060; src/ast/ast_util.cpp:173-180.
// Source: src/test/sorting_network.cpp:150-181.

import (
	"testing"
)

func TestASTSortingBooleanFactoryRealNodesAndNotUnwrap(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	x, e := m.MkFreshConst("x", sort)
	astTestMust(t, e)
	astTestPin(t, m, x)
	f, e := m.MkFalse()
	astTestMust(t, e)
	tr, e := m.MkTrue()
	astTestMust(t, e)
	if f == tr {
		t.Fatal("pinned constants")
	}
	neg, e := m.MkNot(x)
	astTestMust(t, e)
	astTestPin(t, m, neg)
	yes, inner, e := m.IsNot(neg)
	astTestMust(t, e)
	if !yes || inner != x {
		t.Fatal("actual unary app", yes, inner)
	}
	yes, _, e = m.IsNot(x)
	astTestMust(t, e)
	if yes {
		t.Fatal("fresh application is not NOT")
	}
	either, e := m.MkOr(x, neg)
	astTestMust(t, e)
	astTestPin(t, m, either)
	both, e := m.MkAnd(x, neg)
	astTestMust(t, e)
	astTestPin(t, m, both)
	if either == both {
		t.Fatal("actual op declarations")
	}
	z, e := m.MkOrArray(nil)
	astTestMust(t, e)
	if z != f {
		t.Fatal("zero clause")
	}
	one, e := m.MkOrArray([]ASTHandle{x})
	astTestMust(t, e)
	if one != x {
		t.Fatal("singleton clause pointer")
	}
}
