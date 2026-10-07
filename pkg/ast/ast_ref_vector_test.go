// Licensed under the MIT license.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTRefVectorRealOwnershipNullReserveAndSelfSet(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkVar(77, b)
	astTestMust(t, e)
	pin := astTestPin(t, m, h)
	_ = pin
	v, e := m.NewRefVector()
	astTestMust(t, e)
	astTestMust(t, m.RefVectorReserve(v, 2))
	n, e := m.RefVectorSize(v)
	astTestMust(t, e)
	if n != 2 {
		t.Fatal("source reserve grows logical size", n)
	}
	astTestMust(t, m.RefVectorSet(v, 0, h))
	astTestMust(t, m.RefVectorSet(v, 0, h))
	rc, e := m.GetRefCount(h)
	astTestMust(t, e)
	if rc != 2 {
		t.Fatal("INC before DEC self set", rc)
	}
	astTestMust(t, m.RefVectorPush(v, h))
	rc, e = m.GetRefCount(h)
	astTestMust(t, e)
	if rc != 3 {
		t.Fatal("source owned entries", rc)
	}
	astTestMust(t, m.RefVectorErase(v, 0))
	rc, e = m.GetRefCount(h)
	astTestMust(t, e)
	if rc != 2 {
		t.Fatal("erase shifts before DEC", rc)
	}
	astTestMust(t, m.DelRefVector(&v))
	rc, e = m.GetRefCount(h)
	astTestMust(t, e)
	if rc != 1 {
		t.Fatal("only actual entries released", rc)
	}
}
func TestASTRefVectorFailedPushRetainsSourceIncrement(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkVar(78, b)
	astTestMust(t, e)
	astTestPin(t, m, h)
	v, e := m.NewRefVector()
	astTestMust(t, e)
	m.BudgetField.FailAfter(0)
	for attempt := uint32(0); attempt < 2; attempt++ {
		if e = m.RefVectorPush(v, h); !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal("source push reserve", e)
		}
		n, e := m.RefVectorSize(v)
		astTestMust(t, e)
		rc, e := m.GetRefCount(h)
		astTestMust(t, e)
		if n != 0 || rc != attempt+2 {
			t.Fatal("source completed INC prefix", n, rc, attempt)
		}
	}
	astTestMust(t, m.DelRefVector(&v))
	rc, e := m.GetRefCount(h)
	astTestMust(t, e)
	if rc != 3 {
		t.Fatal("empty vector must not consume failed retained prefixes", rc)
	}
	// Explicit caller balancing uses the real source manager API.
	astTestMust(t, m.DecRef(h))
	astTestMust(t, m.DecRef(h))
	m.BudgetField.FailAfter(-1)
}

func TestASTRefVectorCopyAppendResizeAndTransfer(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	a, e := m.MkVar(79, b)
	astTestMust(t, e)
	astTestPin(t, m, a)
	v, e := m.NewRefVector()
	astTestMust(t, e)
	astTestMust(t, m.RefVectorResizeFill(v, 3, a))
	copy, e := m.CopyRefVector(v)
	astTestMust(t, e)
	astTestMust(t, m.RefVectorAppend(copy, v))
	n, e := m.RefVectorSize(copy)
	astTestMust(t, e)
	if n != 6 {
		t.Fatal("copy and append", n)
	}
	astTestMust(t, m.RefVectorReverse(copy))
	astTestMust(t, m.RefVectorSwapElements(copy, 0, 5))
	astTestMust(t, m.RefVectorPop(copy))
	astTestMust(t, m.RefVectorShrink(copy, 2))
	astTestMust(t, m.SwapRefVectors(v, copy))
	n, e = m.RefVectorSize(v)
	astTestMust(t, e)
	if n != 2 {
		t.Fatal("source table swap", n)
	}
	astTestMust(t, m.RefVectorAssign(v, v))
	astTestMust(t, m.RefVectorAssign(v, copy))
	astTestMust(t, m.RefVectorReset(copy))
	astTestMust(t, m.RefVectorFinalize(v))
	astTestMust(t, m.DelRefVector(&v))
	astTestMust(t, m.DelRefVector(&copy))
	rc, e := m.GetRefCount(a)
	astTestMust(t, e)
	if rc != 1 {
		t.Fatal("source vector RC audit", rc)
	}
}

func TestASTRefVectorSetIndexGuardAfterSourceIncrement(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkVar(80, b)
	astTestMust(t, e)
	astTestPin(t, m, h)
	v, e := m.NewRefVector()
	astTestMust(t, e)
	expect := error(arena.ErrBounds)
	if ASTSourceDebug {
		expect = ErrASTDomain
	}
	if e = m.RefVectorSet(v, 0, h); !errors.Is(e, expect) {
		t.Fatal("checked index guard", e)
	}
	rc, e := m.GetRefCount(h)
	astTestMust(t, e)
	n, e := m.RefVectorSize(v)
	astTestMust(t, e)
	if rc != 2 || n != 0 {
		t.Fatal("source incoming INC precedes index dereference", rc, n)
	}
	astTestMust(t, m.DelRefVector(&v))
	astTestMust(t, m.DecRef(h))
}
