package ast

// Source: src/ast/ast.h:725-731; src/model/model_evaluator.cpp:198-202.

import "testing"

func TestASTAppArgumentViewBorrowAndLockedFactory(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	a, e := m.MkConst(astTestSymbol(t, m, "view-a"), b)
	astTestMust(t, e)
	astTestPin(t, m, a)
	c, e := m.MkConst(astTestSymbol(t, m, "view-b"), b)
	astTestMust(t, e)
	astTestPin(t, m, c)
	app, e := m.MkAnd(a, c)
	astTestMust(t, e)
	pin, e := m.NewRef(app)
	astTestMust(t, e)
	d, e := m.GetDecl(app)
	astTestMust(t, e)
	view, e := m.AppArgumentView(app)
	astTestMust(t, e)
	if view.Count() != 2 || view.Size() != 2 {
		t.Fatal("borrowed argument count", view.Count(), view.Size())
	}
	got, e := view.At(1)
	astTestMust(t, e)
	if got != c {
		t.Fatal("indexed borrow", got)
	}
	rebuilt, e := m.RewriterMakeApp(d, view)
	astTestMust(t, e)
	if rebuilt != app {
		t.Fatal("view inside locked factory", rebuilt)
	}
	if _, e = view.At(2); e == nil {
		t.Fatal("out of bounds borrow accepted")
	}
	astTestMust(t, m.DelRef(&pin))
	if _, e = view.At(0); e == nil {
		t.Fatal("expired view accepted")
	}
}
