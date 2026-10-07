package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
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
	if ASTSourceDebug {
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
	if ASTSourceDebug {
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
	if ASTSourceDebug {
		want = ErrASTDomain
	}
	if !errors.Is(e, want) {
		t.Fatalf("never initialized physical slot: %v", e)
	}
	astTestMust(t, m.ExprArrayPop(&a)) // source pointer now dangling, but still stored
	got, e := m.ExprArrayGet(&a, 0)
	if ASTSourceDebug {
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

// Empty nonroot POP allocates a patch with index0 in release; its unsigned
// logical size wraps. No backing access occurs on this finite source branch.
func TestASTExprArrayEmptyNonrootPopSourcePrefix(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkVar(831, b)
	astTestMust(t, e)
	astTestMust(t, m.IncRef(h))
	var current, saved, out ASTExprArray
	astTestMust(t, m.ExprArrayPush(&current, h))
	astTestMust(t, m.ExprArrayPop(&current))
	astTestMust(t, m.ExprArrayCopy(&current, &saved))
	astTestMust(t, m.ExprArrayPush(&current, h))
	old := saved.cell
	for _, from := range []bool{true, false} {
		var err error
		if from {
			err = m.ExprArrayPopFrom(&saved, &out)
		} else {
			err = m.ExprArrayPop(&saved)
		}
		if ASTSourceDebug {
			if !errors.Is(err, ErrASTDomain) {
				t.Fatalf("source empty assertion: %v", err)
			}
			if saved.cell != old || out.cell != (arena.SmallBytesRef{}) {
				t.Fatal("assertion changed owners")
			}
		} else {
			astTestMust(t, err)
			ref := &saved
			if from {
				ref = &out
			}
			n, e := m.ExprArraySize(ref)
			astTestMust(t, e)
			if n != ^uint32(0) {
				t.Fatalf("source size wrap: %d", n)
			}
		}
	}
	astTestMust(t, m.ExprArrayDel(&out))
	astTestMust(t, m.ExprArrayDel(&saved))
	astTestMust(t, m.ExprArrayDel(&current))
	astTestMust(t, m.DecRef(h))
}

func TestASTExprArrayRetainedSetInvalidOldPreservesIncomingIncrement(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkVar(841, b)
	astTestMust(t, e)
	v, e := m.MkVar(842, b)
	astTestMust(t, e)
	astTestMust(t, m.IncRef(v))
	var a ASTExprArray
	astTestMust(t, m.ExprArrayPush(&a, h))
	astTestMust(t, m.ExprArrayPop(&a))
	e = m.ExprArraySet(&a, 0, v)
	want := ErrASTHandle
	rcWant := uint32(2)
	if ASTSourceDebug {
		want = ErrASTDomain
		rcWant = 1
	}
	if !errors.Is(e, want) {
		t.Fatalf("invalid retained old: %v", e)
	}
	rc, e := m.GetRefCount(v)
	astTestMust(t, e)
	if rc != rcWant {
		t.Fatalf("INC before invalid old access must survive: got%d want%d", rc, rcWant)
	}
	astTestMust(t, m.ExprArrayDel(&a))
	for ; rcWant > 0; rcWant-- {
		astTestMust(t, m.DecRef(v))
	}
}

func TestASTExprArrayUninitializedSetPreservesIncomingIncrement(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkVar(851, b)
	astTestMust(t, e)
	v, e := m.MkVar(852, b)
	astTestMust(t, e)
	astTestMust(t, m.IncRef(v))
	var a ASTExprArray
	astTestMust(t, m.ExprArrayPush(&a, h)) // capacity2, slot1 was never initialized.
	e = m.ExprArraySet(&a, 1, v)
	want := arena.ErrBounds
	rcWant := uint32(2)
	if ASTSourceDebug {
		want = ErrASTDomain
		rcWant = 1
	}
	if !errors.Is(e, want) {
		t.Fatalf("uninitialized old access: %v", e)
	}
	rc, e := m.GetRefCount(v)
	astTestMust(t, e)
	if rc != rcWant {
		t.Fatalf("INC before uninitialized old read must survive: got%d want%d", rc, rcWant)
	}
	astTestMust(t, m.ExprArrayDel(&a))
	for ; rcWant > 0; rcWant-- {
		astTestMust(t, m.DecRef(v))
	}
}
