package ast

import "testing"

func TestASTExprMapConcreteReferenceCleanup(t *testing.T) {
	m := astTestManager(t)
	nodes := markTestNodes(t, m, 3)
	key, value, replacement := nodes[0], nodes[1], nodes[2]
	for _, h := range nodes {
		astTestMust(t, m.IncRef(h)) // caller retains each expression
	}
	refCount := func(h ASTHandle, want uint32) {
		t.Helper()
		got, e := m.GetRefCount(h)
		if e != nil || got != want {
			t.Fatalf("ref count for %v = %d, want %d: %v", h, got, want, e)
		}
	}
	s, e := NewASTExprMapWithProofs(m, false)
	astTestMust(t, e)
	if e := s.Insert(key, value, ASTHandle{origin: 999}); e != nil {
		t.Fatal("disabled proof must not be read", e)
	}
	refCount(key, 2)
	refCount(value, 2)
	got, proof, found, e := s.Get(key)
	if e != nil || !found || got != value || !proof.IsNull() {
		t.Fatal("source map lookup", got, proof, found, e)
	}
	astTestMust(t, s.Insert(key, replacement, ASTHandle{}))
	refCount(key, 2)
	refCount(value, 1)
	refCount(replacement, 2)
	astTestMust(t, s.Insert(key, replacement, ASTHandle{})) // INC before DEC
	refCount(replacement, 2)
	astTestMust(t, s.Erase(key))
	refCount(key, 1)
	refCount(replacement, 1)
	if yes, e := s.Contains(key); e != nil || yes {
		t.Fatal("erased entry", yes, e)
	}

	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	p, e := m.MkProofTrue()
	astTestMust(t, e)
	astTestMust(t, m.IncRef(p))
	proofBase, e := m.GetRefCount(p)
	astTestMust(t, e)
	astTestMust(t, s.SetStoreProofs(true))
	astTestMust(t, s.Insert(key, value, p))
	refCount(key, 2)
	refCount(value, 2)
	refCount(p, proofBase+1)
	got, proof, found, e = s.Get(key)
	if e != nil || !found || got != value || proof != p {
		t.Fatal("proof map lookup", got, proof, found, e)
	}
	astTestMust(t, s.Reset()) // proof table before key/value table
	refCount(key, 1)
	refCount(value, 1)
	refCount(p, proofBase)
	astTestMust(t, s.Insert(key, value, p))
	astTestMust(t, s.SetStoreProofs(false)) // source flush on policy change
	refCount(key, 1)
	refCount(value, 1)
	refCount(p, proofBase)
	astTestMust(t, s.Close())
	for _, h := range nodes {
		astTestMust(t, m.DecRef(h))
	}
	astTestMust(t, m.DecRef(p))
}
