package ast

import (
	"errors"
	"testing"
)

func TestASTExprMapStoppedProofCleanupPrecedesExpressionEntries(t *testing.T) {
	if !ASTSourceDebug {
		t.Skip("source table assertion is compiled out")
	}
	m := astTestManager(t)
	nodes := markTestNodes(t, m, 2)
	for _, h := range nodes {
		astTestMust(t, m.IncRef(h))
	}
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	proof, e := m.MkProofTrue()
	astTestMust(t, e)
	s, e := NewASTExprMapWithProofs(m, true)
	astTestMust(t, e)
	astTestMust(t, s.Insert(nodes[0], nodes[1], proof))
	proofNode, e := m.proofAppNode(proof)
	astTestMust(t, e)
	astTestMust(t, m.tableEraseIdentity(proofNode))
	if e = s.Reset(); !errors.Is(e, ErrASTDomain) {
		t.Fatalf("stopped proof release: %v", e)
	}
	for _, h := range nodes {
		if refs, e := m.GetRefCount(h); e != nil || refs != 2 {
			t.Fatalf("expression released before proof phase: %d, %v", refs, e)
		}
	}
	if _, inserted, e := m.tableInsert(proofNode); e != nil || !inserted {
		t.Fatalf("restore proof table entry: %v, %v", inserted, e)
	}
	astTestMust(t, s.Reset())
	for _, h := range nodes {
		if refs, e := m.GetRefCount(h); e != nil || refs != 1 {
			t.Fatalf("expression cleanup did not resume: %d, %v", refs, e)
		}
	}
	astTestMust(t, m.deleteNode(proofNode))
	astTestMust(t, s.Close())
	for _, h := range nodes {
		astTestMust(t, m.DecRef(h))
	}
}

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

func TestASTExprMapBorrowedPairsForAPITransfer(t *testing.T) {
	m := astTestManager(t)
	nodes := markTestNodes(t, m, 3)
	for _, h := range nodes {
		astTestMust(t, m.IncRef(h))
	}
	s, e := NewASTExprMapWithProofs(m, false)
	astTestMust(t, e)
	astTestMust(t, s.Insert(nodes[0], nodes[1], ASTHandle{}))
	astTestMust(t, s.Insert(nodes[2], nodes[0], ASTHandle{}))
	before := make([]uint32, len(nodes))
	for i, h := range nodes {
		before[i], e = m.GetRefCount(h)
		astTestMust(t, e)
	}
	pairs, e := s.Pairs()
	astTestMust(t, e)
	if len(pairs) != 2 {
		t.Fatal("API transfer pair count", len(pairs))
	}
	got := make(map[ASTHandle]ASTHandle, len(pairs))
	for _, pair := range pairs {
		got[pair.Key] = pair.Value
	}
	if got[nodes[0]] != nodes[1] || got[nodes[2]] != nodes[0] {
		t.Fatal("API transfer pair contents", got)
	}
	for i, h := range nodes {
		rc, e := m.GetRefCount(h)
		if e != nil || rc != before[i] {
			t.Fatal("borrowed snapshot changed ownership", i, rc, before[i], e)
		}
	}
	// API transfer increments both sides before the local expression map dies.
	for _, pair := range pairs {
		astTestMust(t, m.IncRef(pair.Key))
		astTestMust(t, m.IncRef(pair.Value))
	}
	astTestMust(t, s.Reset())
	if pairs, e = s.Pairs(); e != nil || len(pairs) != 0 {
		t.Fatal("reset retained API pairs", pairs, e)
	}
	for i, h := range nodes {
		rc, e := m.GetRefCount(h)
		if e != nil || rc != before[i] {
			t.Fatal("API transfer lost a live pair", i, rc, before[i], e)
		}
	}
	astTestMust(t, s.Close())
	for _, pair := range got {
		astTestMust(t, m.DecRef(pair))
	}
	for key := range got {
		astTestMust(t, m.DecRef(key))
	}
	for _, h := range nodes {
		astTestMust(t, m.DecRef(h))
	}
}
