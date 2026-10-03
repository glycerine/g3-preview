// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

// parray.h Get/Set logical assertions are debug-only. Pop's physical pointer
// remains in backing; independent references keep its pointed-to node live.
func TestASTExprArrayRetainedRootGetAndSet(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkVar(801, b)
	astTestMust(t, e)
	v, e := m.MkVar(802, b)
	astTestMust(t, e)
	astTestMust(t, m.IncRef(h))
	astTestMust(t, m.IncRef(h))
	astTestMust(t, m.IncRef(v))
	var a ASTExprArray
	astTestMust(t, m.ExprArrayPush(&a, h))
	astTestMust(t, m.ExprArrayPop(&a))
	got, e := m.ExprArrayGet(&a, 0)
	if astSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatalf("source debug Get assertion: %v", e)
		}
		if e = m.ExprArraySet(&a, 0, v); !errors.Is(e, ErrASTDomain) {
			t.Fatalf("source debug Set assertion: %v", e)
		}
		astTestMust(t, m.ExprArrayDel(&a))
		return
	}
	astTestMust(t, e)
	if got != h {
		t.Fatal("lost initialized retained pointer", got, h)
	}
	astTestMust(t, m.ExprArraySet(&a, 0, v))
	got, e = m.ExprArrayGet(&a, 0)
	astTestMust(t, e)
	if got != v {
		t.Fatal("retained Set not published")
	}
	rc, e := m.GetRefCount(h)
	astTestMust(t, e)
	if rc != 1 {
		t.Fatal("retained Set must DEC old pointer", rc)
	}
	rc, e = m.GetRefCount(v)
	astTestMust(t, e)
	if rc != 2 {
		t.Fatal("retained Set must INC incoming pointer", rc)
	}
	astTestMust(t, m.ExprArrayDel(&a))
	rc, e = m.GetRefCount(v)
	astTestMust(t, e)
	if rc != 2 {
		t.Fatal("empty destructor must not DEC retained tail", rc)
	}
	astTestMust(t, m.DecRef(h))
	astTestMust(t, m.DecRef(v))
	astTestMust(t, m.DecRef(v))
}

func TestASTExprArrayNonrootSetArbitraryIndex(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkVar(811, b)
	astTestMust(t, e)
	v, e := m.MkVar(812, b)
	astTestMust(t, e)
	var current, saved, out ASTExprArray
	astTestMust(t, m.ExprArrayPush(&current, h))
	astTestMust(t, m.ExprArrayCopy(&current, &saved))
	astTestMust(t, m.ExprArraySet(&current, 0, v)) // saved becomes inverse SET
	old := saved.cell
	e = m.ExprArraySetFrom(&saved, ^uint32(0), v, &out)
	if astSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatalf("source debug SetFrom assertion: %v", e)
		}
		if out.cell != (arena.SmallBytesRef{}) {
			t.Fatal("assertion must precede output copy")
		}
	} else {
		astTestMust(t, e)
		if out.cell == old {
			t.Fatal("SET cell not allocated")
		}
		got, e := m.ExprArrayGet(&out, ^uint32(0))
		astTestMust(t, e)
		if got != v {
			t.Fatal("nonroot matching SET should return without root backing read")
		}
		n, e := m.ExprArraySize(&out)
		astTestMust(t, e)
		if n != 1 {
			t.Fatal("SET changed logical size", n)
		}
	}
	astTestMust(t, m.ExprArrayDel(&out))
	astTestMust(t, m.ExprArrayDel(&saved))
	astTestMust(t, m.ExprArrayDel(&current))
}

func TestASTExprArrayUninitializedAndFreedRetainedSlots(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkVar(821, b)
	astTestMust(t, e)
	var a ASTExprArray
	astTestMust(t, m.ExprArrayPush(&a, h)) // backing capacity2, only index0 initialized
	_, e = m.ExprArrayGet(&a, 1)
	want := arena.ErrBounds
	if astSourceDebug {
		want = ErrASTDomain
	}
	if !errors.Is(e, want) {
		t.Fatalf("never initialized physical slot: %v", e)
	}
	astTestMust(t, m.ExprArrayPop(&a)) // source pointer now dangling, but still stored
	got, e := m.ExprArrayGet(&a, 0)
	if astSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatalf("source debug logical assertion: %v", e)
		}
	} else {
		astTestMust(t, e)
		if got != h {
			t.Fatal("raw retained token changed")
		}
		if _, e = m.GetKind(got); !errors.Is(e, ErrASTHandle) {
			t.Fatal("dereference must reject freed AST", e)
		}
	}
	astTestMust(t, m.ExprArrayDel(&a))
}
