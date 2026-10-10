package ast

import (
	"errors"
	"testing"
)

func TestASTDependencyDecConsumedActualRootOwnership(t *testing.T) {
	m := astTestManager(t)
	h := astDependencyExpr(t, m, 2301)
	d, e := m.DependencyLeaf(h)
	astTestMust(t, e)
	astTestMust(t, m.DependencyInc(d))
	astTestMust(t, m.DependencyInc(d))
	m.BudgetField.FailAfter(0)
	consumed, e := m.DependencyDecConsumed(d)
	astTestMust(t, e)
	if !consumed {
		t.Fatal("live reference not consumed")
	}
	if rc, e := m.DependencyRefCount(d); e != nil || rc != 1 {
		t.Fatal("shared root decrement", rc, e)
	}
	consumed, e = m.DependencyDecConsumed(d)
	astTestMust(t, e)
	if !consumed || m.dependencies.live != 0 {
		t.Fatal("last root decrement", consumed, m.dependencies.live)
	}
	consumed, e = m.DependencyDecConsumed(d)
	if consumed || !errors.Is(e, ErrASTHandle) {
		t.Fatal("freed root must not be consumed again", consumed, e)
	}
	consumed, e = m.DependencyDecConsumed(ASTDependency{})
	astTestMust(t, e)
	if !consumed {
		t.Fatal("empty ownership must retire")
	}
	if rc, e := m.GetRefCount(h); e != nil || rc != 1 {
		t.Fatal("actual leaf callback", rc, e)
	}
	m.BudgetField.FailAfter(-1)

	other := astTestManager(t)
	d, e = m.DependencyLeaf(h)
	astTestMust(t, e)
	astTestMust(t, m.DependencyInc(d))
	consumed, e = other.DependencyDecConsumed(d)
	if consumed || !errors.Is(e, ErrASTHandle) {
		t.Fatal("foreign guard must preserve owner", consumed, e)
	}
	if rc, e := m.DependencyRefCount(d); e != nil || rc != 1 {
		t.Fatal("foreign guard decremented owner", rc, e)
	}
	astTestMust(t, m.DependencyDec(d))
}
