// Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestASTRefVectorSourceRetainedNullShrink(t *testing.T) {
	m := astTestManager(t)
	v, e := m.NewRefVector()
	astTestMust(t, e)
	astTestMust(t, m.RefVectorResize(v, 3))
	astTestMust(t, m.RefVectorShrink(v, 1))
	used := m.budget.Used()
	m.budget.FailAfter(0)
	e = m.RefVectorShrink(v, 2)
	if astSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal("source shrink assertion", e)
		}
	} else {
		astTestMust(t, e)
		h, e := m.RefVectorGet(v, 1)
		astTestMust(t, e)
		if !h.IsNull() {
			t.Fatal("retained null", h)
		}
	}
	n, e := m.RefVectorSize(v)
	astTestMust(t, e)
	expect := uint32(2)
	if astSourceDebug {
		expect = 1
	}
	if n != expect || m.budget.Used() != used {
		t.Fatal(n, expect, m.budget.Used(), used)
	}
	m.budget.FailAfter(-1)
	astTestMust(t, m.DelRefVector(&v))
}

func TestASTRefVectorSourceRetainedPointerAndBorrow(t *testing.T) {
	m := astTestManager(t)
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkVar(990, s)
	astTestMust(t, e)
	astTestPin(t, m, h)
	v, e := m.NewRefVectorFrom([]ASTHandle{h, h, h})
	astTestMust(t, e)
	view, e := m.RefVectorElement(v, 2)
	astTestMust(t, e)
	it, e := m.RefVectorBegin(v)
	astTestMust(t, e)
	astTestMust(t, m.RefVectorReset(v))
	got, e := m.RefElementGet(view)
	astTestMust(t, e) // element_ref::get has no repeated vector-size assertion.
	if got != h {
		t.Fatal("retained physical element", got)
	}
	for j := 0; j < 3; j++ {
		got, ok, e := m.RefVectorNext(&it)
		astTestMust(t, e)
		if !ok || got != h {
			t.Fatal("retained source pointer iterator", got, ok)
		}
	}
	got, e = m.RefVectorGet(v, 2)
	if astSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal("operator[] assertion", e)
		}
	} else if e != nil || got != h {
		t.Fatal("release retained pointer", got, e)
	}
	// Incoming INC occurs before operator[]'s debug assertion and old DEC.
	e = m.RefVectorSet(v, 2, h)
	rc, re := m.GetRefCount(h)
	astTestMust(t, re)
	if astSourceDebug {
		if !errors.Is(e, ErrASTDomain) || rc != 2 {
			t.Fatal("increment assertion prefix", e, rc)
		}
		astTestMust(t, m.DecRef(h))
	} else if e != nil || rc != 1 {
		t.Fatal("retained self set inc/dec", e, rc)
	}
	astTestMust(t, m.DelRefVector(&v))
}

func TestASTRefVectorNeverInitializedPhysicalSlot(t *testing.T) {
	m := astTestManager(t)
	v, e := m.NewRefVector()
	astTestMust(t, e)
	astTestMust(t, m.RefVectorResize(v, 4)) // requested capacity5; slot4 unconstructed.
	astTestMust(t, m.RefVectorShrink(v, 1))
	_, e = m.RefVectorGet(v, 4)
	if !errors.Is(e, ErrASTDomain) {
		t.Fatal("never-constructed pointer is not a source null", e)
	}
	e = m.RefVectorShrink(v, 5)
	if !errors.Is(e, ErrASTDomain) {
		t.Fatal("never-initialized size publication", e)
	}
	if !astSourceDebug {
		_, e = m.RefVectorGet(v, 5)
		if !errors.Is(e, arena.ErrBounds) {
			t.Fatal("physical boundary", e)
		}
	}
	astTestMust(t, m.DelRefVector(&v))
}

func TestASTRefVectorInitializedGrowthFailurePrefix(t *testing.T) {
	m := astTestManager(t)
	v, e := m.NewRefVector()
	astTestMust(t, e)
	astTestMust(t, m.RefVectorResize(v, 4))
	astTestMust(t, m.RefVectorShrink(v, 1))
	old, e := m.vectorState(v)
	astTestMust(t, e)
	used := m.budget.Used()
	m.budget.FailAfter(1)
	e = m.RefVectorResize(v, 9)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	s, e := m.vectorState(v)
	astTestMust(t, e)
	if s.size != 1 || s.capacity != 8 || s.initialized != 4 {
		t.Fatal("published growth retains initialization only", s)
	}
	if m.vectorAllocator.Stats().RequestedBytes != uint64(s.capacity)*8+uint64(astRefVectorHeaderBytes) {
		t.Fatal("initialized header must be real charged payload", m.vectorAllocator.Stats())
	}
	// DirectOnly payload overlap charges the new physical header too; the
	// already-created descriptor page has ample space and does not grow.
	if m.budget.Used() != used+uint64(s.capacity-old.capacity)*8 {
		t.Fatal("unaccounted physical growth", used, m.budget.Used())
	}
	if m.vectorAllocator.IsLive(old.payload) {
		t.Fatal("old backing remains live")
	}
	r, e := m.vectorPhysicalValue(s, 3)
	astTestMust(t, e)
	if r != (astNodeRef{}) {
		t.Fatal("initialized tail", r)
	}
	_, e = m.vectorPhysicalValue(s, 4)
	if !errors.Is(e, ErrASTDomain) {
		t.Fatal("growth must not initialize spare capacity", e)
	}
	m.budget.FailAfter(-1)
	astTestMust(t, m.DelRefVector(&v))
	if m.vectorAllocator.Stats().Live != 0 || m.vectorAllocator.Stats().RequestedBytes != 0 {
		t.Fatal("immediate backing cleanup", m.vectorAllocator.Stats())
	}
}

func TestASTRefVectorRetainedFreedHandleBoundary(t *testing.T) {
	m := astTestManager(t)
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkVar(991, s)
	astTestMust(t, e)
	v, e := m.NewRefVectorFrom([]ASTHandle{h})
	astTestMust(t, e)
	view, e := m.RefVectorElement(v, 0)
	astTestMust(t, e)
	astTestMust(t, m.RefVectorReset(v)) // the only owning reference is released.
	got, e := m.RefElementGet(view)
	astTestMust(t, e) // source raw pointer copy owns nothing and does not dereference.
	if got != h {
		t.Fatal("retained pointer bits", got, h)
	}
	if _, e = m.GetRefCount(got); !errors.Is(e, ErrASTHandle) {
		t.Fatal("freed AST token cannot be dereferenced", e)
	}
	astTestMust(t, m.DelRefVector(&v))
}
