package ast

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func TestASTSparseMarkNativeGrowthIdentityAndNoRetains(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 257)
	s, e := NewASTExprSparseMark(m)
	astTestMust(t, e)
	defer s.Close()
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	for _, n := range h {
		astTestMust(t, s.Mark(n, true))
		astTestMust(t, s.Mark(n, true))
		yes, e := s.IsMarked(n)
		astTestMust(t, e)
		rc, e := m.GetRefCount(n)
		astTestMust(t, e)
		if !yes || rc != 0 {
			t.Fatal("native sparse keys must borrow each expression", yes, rc)
		}
	}
	m.BudgetField.FailAfter(-1)
	if len(s.entries) != len(h) || m.BudgetField.Used() != before {
		t.Fatal("native growth changed object storage")
	}
}

func TestASTSparseMarkNativeResetAndCloseAfterBorrowedNodesDie(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 128)
	s, e := NewASTExprSparseMark(m)
	astTestMust(t, e)
	for _, n := range h {
		astTestMust(t, s.Mark(n, true))
		astTestMust(t, m.IncRef(n))
		astTestMust(t, m.DecRef(n))
	}
	m.BudgetField.FailAfter(0)
	astTestMust(t, s.Reset())
	if len(s.entries) != 0 {
		t.Fatal("reset retained dead borrowed keys")
	}
	astTestMust(t, s.Close())
	m.BudgetField.FailAfter(-1)
}

func TestASTSparseMarkBoundedIdentityOracle(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 100)
	s, e := NewASTExprSparseMark(m)
	astTestMust(t, e)
	defer s.Close()
	var expected [100]bool
	seed := uint32(17)
	for step := 0; step < 1500; step++ {
		seed = seed*1664525 + 1013904223
		idx := seed % 100
		flag := seed&256 != 0
		astTestMust(t, s.Mark(h[idx], flag))
		expected[idx] = flag
		if step%29 == 0 {
			count := uint32(0)
			for i, n := range h {
				yes, e := s.IsMarked(n)
				astTestMust(t, e)
				if yes != expected[i] {
					t.Fatal(step, i, yes, expected[i])
				}
				if expected[i] {
					count++
				}
			}
			if uint32(len(s.entries)) != count {
				t.Fatal("count", step)
			}
		}
	}
}

func TestASTSparseMarkNativeEraseAndReinsertWithReservationsDisabled(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 257)
	s, e := NewASTExprSparseMark(m)
	astTestMust(t, e)
	defer s.Close()
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	for _, n := range h {
		astTestMust(t, s.Mark(n, true))
	}
	for i, n := range h {
		if i%2 == 0 {
			astTestMust(t, s.Mark(n, false))
		}
	}
	for i, n := range h {
		yes, e := s.IsMarked(n)
		astTestMust(t, e)
		if yes != (i%2 != 0) {
			t.Fatal("erase changed another identity", i, yes)
		}
		astTestMust(t, s.Mark(n, true))
	}
	m.BudgetField.FailAfter(-1)
	if len(s.entries) != len(h) || m.BudgetField.Used() != before {
		t.Fatal("native churn changed object storage")
	}
}

func TestASTSparseMarkNativeRejectsForeignAndStaleQueries(t *testing.T) {
	m := astTestManager(t)
	foreign := astTestManager(t)
	good := markTestNodes(t, m, 1)[0]
	bad := markTestNodes(t, foreign, 1)[0]
	s, e := NewASTExprSparseMark(m)
	astTestMust(t, e)
	defer s.Close()
	astTestMust(t, s.Mark(good, true))
	astTestMust(t, m.IncRef(good))
	astTestMust(t, m.DecRef(good))
	for _, n := range []ASTHandle{good, bad} {
		if _, e := s.IsMarked(n); !errors.Is(e, ErrASTHandle) {
			t.Fatal("invalid query accepted", e)
		}
		for _, flag := range []bool{false, true} {
			if e := s.Mark(n, flag); !errors.Is(e, ErrASTHandle) {
				t.Fatal("invalid mutation accepted", e)
			}
		}
	}
	astTestMust(t, s.Reset())
}

func TestASTSparseMarkNativeColdHeaderRefusalAndCopyGuard(t *testing.T) {
	m := astTestManager(t)
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	s, e := NewASTExprSparseMark(m)
	m.BudgetField.FailAfter(-1)
	if s != nil || !errors.Is(e, arena.ErrInjectedFailure) || m.BudgetField.Used() != before {
		t.Fatal("actual cold header refusal", s, e)
	}
	s, e = NewASTExprSparseMark(m)
	astTestMust(t, e)
	copy := *s
	if e := copy.Reset(); !errors.Is(e, ErrASTHandle) {
		t.Fatal("copied owner accepted", e)
	}
	astTestMust(t, s.Close())
	if e := s.Reset(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed owner accepted", e)
	}
}

func TestASTSparseMarkNativeRecycledSlotGenerationIdentity(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 1)[0]
	s, e := NewASTExprSparseMark(m)
	astTestMust(t, e)
	defer s.Close()
	astTestMust(t, s.Mark(h, true))
	astTestMust(t, m.IncRef(h))
	astTestMust(t, m.DecRef(h))
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	fresh, e := m.MkVar(65000, b)
	astTestMust(t, e)
	if fresh.slot != h.slot || fresh.generation == h.generation {
		t.Fatal("actual physical slot generation fixture", h, fresh)
	}
	yes, e := s.IsMarked(fresh)
	astTestMust(t, e)
	if yes {
		t.Fatal("old borrowed key matched recycled node")
	}
	astTestMust(t, s.Mark(fresh, true))
	astTestMust(t, s.Reset())
}
