package ast

import (
	"errors"
	"testing"
)

func TestASTRefVectorElementBorrowRetainsAndInvalidatesOnGrowth(t *testing.T) {
	m := astTestManager(t)
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	a, e := m.MkVar(500, s)
	astTestMust(t, e)
	b, e := m.MkVar(501, s)
	astTestMust(t, e)
	astTestPin(t, m, a)
	astTestPin(t, m, b)
	v, e := m.NewRefVectorFrom([]ASTHandle{a, b})
	astTestMust(t, e)
	view, e := m.RefVectorElement(v, 0)
	astTestMust(t, e)
	astTestMust(t, m.RefElementSet(view, b))
	got, e := m.RefElementGet(view)
	astTestMust(t, e)
	if got != b {
		t.Fatal("element assignment")
	}
	rc, e := m.GetRefCount(b)
	astTestMust(t, e)
	if rc != 3 {
		t.Fatal("element inc-before-dec", rc)
	}
	iter, e := m.RefVectorBegin(v)
	astTestMust(t, e)
	// C++ element_ref binds a physical slot, not the vector object. A
	// move transfers that still-live payload without invalidating the borrow.
	moved, e := m.MoveRefVector(v)
	astTestMust(t, e)
	astTestMust(t, m.DelRefVector(&v))
	got, e = m.RefElementGet(view)
	astTestMust(t, e)
	if got != b {
		t.Fatal("move invalidated live physical borrow")
	}
	v = moved
	other, e := m.NewRefVector()
	astTestMust(t, e)
	astTestMust(t, m.SwapRefVectors(v, other))
	got, e = m.RefElementGet(view)
	astTestMust(t, e)
	if got != b {
		t.Fatal("swap invalidated physical element")
	}
	astTestMust(t, m.SwapRefVectors(v, other))
	astTestMust(t, m.DelRefVector(&other))
	for i := 0; i < 2; i++ {
		h, ok, e := m.RefVectorNext(&iter)
		astTestMust(t, e)
		if !ok || h != b {
			t.Fatal("borrowed iteration", h, ok)
		}
	}
	_, ok, e := m.RefVectorNext(&iter)
	astTestMust(t, e)
	if ok {
		t.Fatal("source end")
	}
	astTestMust(t, m.RefVectorPush(v, a)) // capacity2→3 invalidates source pointer borrow
	if _, e = m.RefElementGet(view); !errors.Is(e, ErrASTHandle) {
		t.Fatal("stale physical element borrow", e)
	}
	astTestMust(t, m.DelRefVector(&v))
}
