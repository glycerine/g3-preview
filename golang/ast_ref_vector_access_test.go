// Licensed under the MIT license.
package g3

import "testing"

func TestASTRefVectorBorrowedInputDefaultEraseAndMove(t *testing.T) {
	m := astTestManager(t)
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	a, e := m.MkVar(400, s)
	astTestMust(t, e)
	b, e := m.MkVar(401, s)
	astTestMust(t, e)
	astTestPin(t, m, a)
	astTestPin(t, m, b)
	v, e := m.NewRefVectorFrom([]ASTHandle{a, b, a})
	astTestMust(t, e)
	got, e := m.RefVectorGetDefault(v, 99, b)
	astTestMust(t, e)
	if got != b {
		t.Fatal("default get")
	}
	astTestMust(t, m.RefVectorEraseValue(v, a))
	got, e = m.RefVectorGet(v, 0)
	astTestMust(t, e)
	if got != b {
		t.Fatal("erase first match")
	}
	astTestMust(t, m.RefVectorAppendValues(v, []ASTHandle{b, {}}))
	rcA, e := m.GetRefCount(a)
	astTestMust(t, e)
	rcB, e := m.GetRefCount(b)
	astTestMust(t, e)
	moved, e := m.MoveRefVector(v)
	astTestMust(t, e)
	empty, e := m.RefVectorEmpty(v)
	astTestMust(t, e)
	if !empty {
		t.Fatal("source moved-from vector not empty")
	}
	afterA, e := m.GetRefCount(a)
	astTestMust(t, e)
	afterB, e := m.GetRefCount(b)
	astTestMust(t, e)
	if afterA != rcA || afterB != rcB {
		t.Fatal("move changed logical references")
	}
	astTestMust(t, m.DelRefVector(&v))
	astTestMust(t, m.DelRefVector(&moved))
}
