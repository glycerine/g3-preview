package ast

import (
	"testing"
)

func TestASTExprArrayActualValueCallbacksAndReroot(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	var a, saved ASTExprArray
	h, e := m.MkVar(42, b)
	astTestMust(t, e)
	astTestMust(t, m.ExprArrayPush(&a, h))
	rc, e := m.GetRefCount(h)
	astTestMust(t, e)
	if rc != 1 {
		t.Fatal("root owns actual expression", rc)
	}
	astTestMust(t, m.ExprArrayCopy(&a, &saved))
	replacement, e := m.MkVar(43, b)
	astTestMust(t, e)
	astTestMust(t, m.ExprArraySet(&a, 0, replacement))
	rc, e = m.GetRefCount(h)
	astTestMust(t, e)
	if rc != 1 {
		t.Fatal("inverse SET owns old expression", rc)
	}
	astTestMust(t, m.ExprArrayReroot(&saved))
	got, e := m.ExprArrayGet(&saved, 0)
	astTestMust(t, e)
	if got != h {
		t.Fatal("source reverse transfer")
	}
	got, e = m.ExprArrayGet(&a, 0)
	astTestMust(t, e)
	if got != replacement {
		t.Fatal("source current version")
	}
	astTestMust(t, m.ExprArrayDel(&a))
	astTestMust(t, m.ExprArrayDel(&saved))
	if _, e = m.GetKind(h); e == nil {
		t.Fatal("old expression never released")
	}
	if _, e = m.GetKind(replacement); e == nil {
		t.Fatal("new expression never released")
	}
}
func TestASTExprArrayEscapedCloseReclaimsPhysicalGraph(t *testing.T) {
	s, q, b := astTestOwners(t)
	m, e := NewASTManager(ASTManagerConfig{Symbols: s, Numerals: q, Budget: b})
	astTestMust(t, e)
	var a ASTExprArray
	boolSort, e := m.MkBoolSort()
	astTestMust(t, e)
	v, e := m.MkVar(99, boolSort)
	astTestMust(t, e)
	astTestMust(t, m.ExprArrayPush(&a, v))
	// Source allocator teardown cannot callback into already freed node records.
	own := m.Stats().AllocatedBytes
	used := b.Used()
	astTestMust(t, m.Close())
	if b.Used() != used-own {
		t.Fatal("escaped array physical graph retained", b.Used(), used-own)
	}
	if e = m.ExprArrayDel(&a); e == nil {
		t.Fatal("closed-owner array callback accepted")
	}
}
