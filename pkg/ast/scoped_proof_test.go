package ast

import (
	"errors"
	"testing"
)

func proofModeMust(t *testing.T, m *ASTManager, want ASTProofMode) {
	t.Helper()
	got, err := m.ProofMode()
	astTestMust(t, err)
	if got != want {
		t.Fatalf("proof mode: got %d, want %d", got, want)
	}
}

func TestASTScopedProofModesNestAndRestore(t *testing.T) {
	m := astTestManager(t)
	proofModeMust(t, m, ASTProofDisabled)
	proofs, err := NewScopedProof(m)
	astTestMust(t, err)
	proofModeMust(t, m, ASTProofEnabled)
	without, err := NewScopedNoProof(m)
	astTestMust(t, err)
	proofModeMust(t, m, ASTProofDisabled)
	astTestMust(t, without.Close())
	proofModeMust(t, m, ASTProofEnabled)
	astTestMust(t, proofs.Close())
	proofModeMust(t, m, ASTProofDisabled)
	astTestMust(t, proofs.Close())
}

func TestASTScopedRestoreProofAndGuards(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	restore, err := NewScopedRestoreProof(m)
	astTestMust(t, err)
	proofModeMust(t, m, ASTProofEnabled)
	astTestMust(t, m.ToggleProofMode(ASTProofDisabled))
	astTestMust(t, restore.Close())
	proofModeMust(t, m, ASTProofEnabled)

	if _, err = NewScopedProofMode(nil, ASTProofEnabled); !errors.Is(err, ErrASTHandle) {
		t.Fatalf("nil manager guard: %v", err)
	}
	if _, err = NewScopedProofMode(m, ASTProofMode(2)); !errors.Is(err, ErrASTDomain) {
		t.Fatalf("invalid mode guard: %v", err)
	}
	copyScope, err := NewScopedNoProof(m)
	astTestMust(t, err)
	copied := *copyScope
	if err = copied.Close(); !errors.Is(err, ErrASTHandle) {
		t.Fatalf("copied scope guard: %v", err)
	}
	astTestMust(t, copyScope.Close())
	proofModeMust(t, m, ASTProofEnabled)
}
