package proofs

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestProofPostOrderParentOrderSharingAndExhaustion(t *testing.T) {
	m := proofManager(t)
	proofMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	b, err := m.MkBoolSort()
	proofMust(t, err)
	a, err := m.MkConst(symbol.Numerical(1701), b)
	proofMust(t, err)
	c, err := m.MkConst(symbol.Numerical(1702), b)
	proofMust(t, err)
	fact, err := m.MkEq(a, c)
	proofMust(t, err)
	leaf, err := m.MkProofHypothesis(fact)
	proofMust(t, err)
	proofMust(t, m.IncRef(leaf))
	t.Cleanup(func() { proofMust(t, m.DecRef(leaf)) })
	left, err := m.MkProofSymmetry(leaf)
	proofMust(t, err)
	proofMust(t, m.IncRef(left))
	t.Cleanup(func() { proofMust(t, m.DecRef(left)) })
	right, err := m.MkProofTransitivity(leaf, left)
	proofMust(t, err)
	proofMust(t, m.IncRef(right))
	t.Cleanup(func() { proofMust(t, m.DecRef(right)) })
	root := right

	it, err := NewProofPostOrder(root, m)
	proofMust(t, err)
	var got []ast.ASTHandle
	for {
		yes, e := it.HasNext()
		proofMust(t, e)
		if !yes {
			break
		}
		h, e := it.Next()
		proofMust(t, e)
		got = append(got, h)
	}
	// Parents are pushed in increasing index order, so the last parent is
	// traversed first. The shared leaf is returned exactly once.
	want := []ast.ASTHandle{leaf, left, root}
	if len(got) != len(want) {
		t.Fatal("postorder length", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatal("postorder", i, got, want)
		}
	}
	if h, e := it.Next(); e != nil || !h.IsNull() {
		t.Fatal("exhausted iterator", h, e)
	}
	copy := *it
	if _, e := copy.HasNext(); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("copied iterator guard", e)
	}
}

func TestProofPostOrderRejectsNonProofAndClosedManager(t *testing.T) {
	m := proofManager(t)
	truth, err := m.MkTrue()
	proofMust(t, err)
	if _, err = NewProofPostOrder(truth, m); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("non-proof root", err)
	}
	proofMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	root, err := m.MkProofAsserted(truth)
	proofMust(t, err)
	proofMust(t, m.IncRef(root))
	it, err := NewProofPostOrder(root, m)
	proofMust(t, err)
	proofMust(t, m.DecRef(root))
	proofMust(t, m.Close())
	if _, err = it.Next(); err == nil {
		t.Fatal("closed manager guard", err)
	}
}
