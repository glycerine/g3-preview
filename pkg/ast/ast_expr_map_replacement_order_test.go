package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTExprMapReplacementReleasesOldBeforePublishingNew(t *testing.T) {
	m := astTestManager(t)
	nodes := markTestNodes(t, m, 3)
	key, old, incoming := nodes[0], nodes[1], nodes[2]
	astTestMust(t, m.IncRef(key))
	astTestMust(t, m.IncRef(incoming))
	s, err := NewASTExprMapWithProofs(m, false)
	astTestMust(t, err)
	astTestMust(t, s.Insert(key, old, ASTHandle{}))
	oldRef, err := m.internal(old)
	astTestMust(t, err)
	m.ExprIDs.Cleanup(m.ExprIDs.Range())
	if m.ExprIDs.FreeCapacity() != 0 {
		t.Fatal("fixture must reach the first source ID recycle allocation")
	}
	m.Budget().FailAfter(0)
	err = s.Insert(key, incoming, ASTHandle{})
	m.Budget().FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) {
		t.Fatalf("old-value deletion refusal: %v", err)
	}
	k, err := m.internal(key)
	astTestMust(t, err)
	entry := s.entries[k]
	_, err = m.internal(old)
	if err == nil || entry.value != oldRef || entry.valueOwned {
		t.Fatalf("source old pointer must remain unpublished after its DEC: entry=%+v, old error=%v", entry, err)
	}
	if len(s.pending) != 1 {
		t.Fatalf("incoming acquired reference must have an owning cleanup receipt: %d", len(s.pending))
	}
	astTestMust(t, s.Reset())
	if rc, err := m.GetRefCount(incoming); err != nil || rc != 1 {
		t.Fatalf("incoming reference cleanup: rc=%d, err=%v", rc, err)
	}
	astTestMust(t, s.Close())
	astTestMust(t, m.DecRef(key))
	astTestMust(t, m.DecRef(incoming))
}

func TestASTExprMapProofReplacementReleasesOldBeforePublishingNew(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	nodes := markTestNodes(t, m, 2)
	key, value := nodes[0], nodes[1]
	astTestMust(t, m.IncRef(key))
	astTestMust(t, m.IncRef(value))
	old, err := m.MkProofTrue()
	astTestMust(t, err)
	incoming, err := m.MkProofAsserted(key)
	astTestMust(t, err)
	astTestMust(t, m.IncRef(incoming))
	s, err := NewASTExprMapWithProofs(m, true)
	astTestMust(t, err)
	astTestMust(t, s.Insert(key, value, old))
	oldRef, err := m.internal(old)
	astTestMust(t, err)
	m.ExprIDs.Cleanup(m.ExprIDs.Range())
	m.Budget().FailAfter(0)
	err = s.Insert(key, value, incoming)
	m.Budget().FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) {
		t.Fatalf("old-proof deletion refusal: %v", err)
	}
	k, err := m.internal(key)
	astTestMust(t, err)
	entry := s.entries[k]
	_, err = m.internal(old)
	if err == nil || entry.proof != oldRef || entry.proofOwned {
		t.Fatalf("source old proof must remain unpublished after its DEC: entry=%+v, old error=%v", entry, err)
	}
	if len(s.pending) != 1 {
		t.Fatalf("incoming proof reference needs an owning cleanup receipt: %d", len(s.pending))
	}
	astTestMust(t, s.Reset())
	if rc, err := m.GetRefCount(incoming); err != nil || rc != 1 {
		t.Fatalf("incoming proof reference cleanup: rc=%d, err=%v", rc, err)
	}
	astTestMust(t, s.Close())
	astTestMust(t, m.DecRef(incoming))
	astTestMust(t, m.DecRef(key))
	astTestMust(t, m.DecRef(value))
}
