package ast

import "testing"

func TestASTExprMapReplacementReleasesOldBeforePublishingNew(t *testing.T) {
	m := astTestManager(t)
	nodes := markTestNodes(t, m, 3)
	key, old, incoming := nodes[0], nodes[1], nodes[2]
	astTestMust(t, m.IncRef(key))
	astTestMust(t, m.IncRef(incoming))
	s, err := NewASTExprMapWithProofs(m, false)
	astTestMust(t, err)
	astTestMust(t, s.Insert(key, old, ASTHandle{}))
	m.Budget().FailAfter(0)
	astTestMust(t, s.Insert(key, incoming, ASTHandle{}))
	m.Budget().FailAfter(-1)
	got, _, found, err := s.Get(key)
	astTestMust(t, err)
	if !found || got != incoming || len(s.pending) != 0 {
		t.Fatal("native replacement did not publish incoming value", got, found, s.pending)
	}
	if _, err := m.internal(old); err == nil {
		t.Fatal("old value survived replacement")
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
	m.Budget().FailAfter(0)
	astTestMust(t, s.Insert(key, value, incoming))
	m.Budget().FailAfter(-1)
	gotValue, gotProof, found, err := s.Get(key)
	astTestMust(t, err)
	if !found || gotValue != value || gotProof != incoming || len(s.pending) != 0 {
		t.Fatal("native replacement did not publish incoming proof", gotValue, gotProof, found, s.pending)
	}
	if _, err := m.internal(old); err == nil {
		t.Fatal("old proof survived replacement")
	}
	astTestMust(t, s.Close())
	astTestMust(t, m.DecRef(incoming))
	astTestMust(t, m.DecRef(key))
	astTestMust(t, m.DecRef(value))
}
