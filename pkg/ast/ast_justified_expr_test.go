// Source: src/ast/justified_expr.h.
package ast

import (
	"errors"
	"testing"
)

func TestASTJustifiedExprConstructCopyAssignAndOwnership(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	a := proofTestBool(t, m, "justified-a")
	b := proofTestBool(t, m, "justified-b")
	pa, e := m.MkProofAsserted(a)
	astTestMust(t, e)
	pb, e := m.MkProofAsserted(b)
	astTestMust(t, e)
	for _, h := range []ASTHandle{a, b, pa, pb} {
		astTestPin(t, m, h)
	}
	refs := func(h ASTHandle) uint32 { n, err := m.GetRefCount(h); astTestMust(t, err); return n }
	baseA, baseB, basePA, basePB := refs(a), refs(b), refs(pa), refs(pb)
	left, e := NewJustifiedExpr(m, a, pa)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, left.Close()) })
	if refs(a) != baseA+1 || refs(pa) != basePA+1 {
		t.Fatal("constructor pins")
	}
	copyOwner, e := left.Copy()
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, copyOwner.Close()) })
	if refs(a) != baseA+2 || refs(pa) != basePA+2 {
		t.Fatal("copy pins")
	}
	right, e := NewJustifiedExpr(m, b, pb)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, right.Close()) })
	astTestMust(t, left.Assign(right))
	formula, e := left.Formula()
	astTestMust(t, e)
	proof, e := left.Proof()
	astTestMust(t, e)
	if formula != b || proof != pb || refs(a) != baseA+1 || refs(pa) != basePA+1 || refs(b) != baseB+2 || refs(pb) != basePB+2 {
		t.Fatal("assignment publication and pins", formula, proof, refs(a), refs(pa), refs(b), refs(pb))
	}
	astTestMust(t, left.Assign(left))
	copyValue := *left
	if _, e = copyValue.Formula(); !errors.Is(e, ErrASTHandle) {
		t.Fatal("copied owner accepted", e)
	}
}

func TestASTJustifiedExprNullProofAndGuards(t *testing.T) {
	m := astTestManager(t)
	a := proofTestBool(t, m, "justified-null-proof")
	astTestPin(t, m, a)
	j, e := NewJustifiedExpr(m, a, ASTHandle{})
	astTestMust(t, e)
	proof, e := j.Proof()
	astTestMust(t, e)
	if !proof.IsNull() {
		t.Fatal("null proof changed", proof)
	}
	astTestMust(t, j.Close())
	if _, e = j.Formula(); !errors.Is(e, ErrASTHandle) {
		t.Fatal("closed owner", e)
	}
	if _, e = NewJustifiedExpr(m, ASTHandle{}, ASTHandle{}); !errors.Is(e, ErrASTHandle) {
		t.Fatal("null formula", e)
	}
	if _, e = NewJustifiedExpr(m, a, a); !errors.Is(e, ErrASTDomain) {
		t.Fatal("non-proof justification", e)
	}
}
