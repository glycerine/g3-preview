package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTExprArrayIndependentSnapshotAndOwnedValues(t *testing.T) {
	m := astTestManager(t)
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	x, err := m.MkVar(1, sort)
	astTestMust(t, err)
	y, err := m.MkVar(2, sort)
	astTestMust(t, err)
	z, err := m.MkVar(3, sort)
	astTestMust(t, err)
	a, err := NewASTExprArray(m)
	astTestMust(t, err)
	astTestMust(t, a.Push(x))
	astTestMust(t, a.Push(y))
	b, err := a.Clone()
	astTestMust(t, err)
	if refs, err := m.GetRefCount(x); err != nil || refs != 2 {
		t.Fatal("copy did not retain expression", refs, err)
	}
	astTestMust(t, b.Set(0, z))
	astTestMust(t, a.Pop())
	if got, err := a.Get(0); err != nil || got != x {
		t.Fatal("parent snapshot changed", got, err)
	}
	if got, err := b.Get(0); err != nil || got != z {
		t.Fatal("child replacement", got, err)
	}
	if got, err := b.Get(1); err != nil || got != y {
		t.Fatal("child retained popped expression", got, err)
	}
	if refs, err := m.GetRefCount(x); err != nil || refs != 1 {
		t.Fatal("replacement released sibling's reference", refs, err)
	}
	if err := b.Set(0, sort); !errors.Is(err, ErrASTDomain) {
		t.Fatal("non-expression accepted", err)
	}
	astTestMust(t, b.Close())
	astTestMust(t, a.Close())
	if _, err := m.GetKind(x); !errors.Is(err, ErrASTHandle) {
		t.Fatal("array reference survived close", err)
	}
	if _, err := a.Get(0); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed array read", err)
	}
}

func TestASTExprArrayStoppedReplacementKeepsOwningPrefix(t *testing.T) {
	m := astTestManager(t)
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	old, err := m.MkVar(11720, sort)
	astTestMust(t, err)
	incoming, err := m.MkVar(11721, sort)
	astTestMust(t, err)
	a, err := NewASTExprArray(m)
	astTestMust(t, err)
	astTestMust(t, a.Push(old))
	m.ExprIDs.Cleanup(m.ExprIDs.Range())
	m.BudgetField.FailAfter(0)
	err = a.Set(0, incoming)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) {
		t.Fatal("actual old-reference decrement", err)
	}
	if got, err := a.Get(0); err != nil || got != old {
		t.Fatal("failed replacement published incoming value", got, err)
	}
	astTestMust(t, a.Close())
	if _, err := m.GetKind(incoming); !errors.Is(err, ErrASTHandle) {
		t.Fatal("incoming pending owner survived close", err)
	}
}
