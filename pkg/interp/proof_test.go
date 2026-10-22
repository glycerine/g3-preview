package interp

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestInterpolationProofRetainedLemmaSelectionAndOwnership(t *testing.T) {
	m, manager := interpProofManagerFixture(t)
	boolSort, err := manager.BoolType()
	interpMust(t, err)
	makeVar := func(name string) ast.ASTHandle {
		h, e := manager.MakeVariable(name, boolSort)
		interpMust(t, e)
		return interpPin(t, m, h)
	}
	a, b, c := makeVar("proof-frame-a"), makeVar("proof-frame-b"), makeVar("proof-frame-c")
	base, err := NewBase(manager, []ast.ASTHandle{a, b, c}, nil, nil)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, base.Close()) })
	proof, err := NewInterpolationProof(base)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, proof.Close()) })

	makeOwned := func(name string) (ast.ASTHandle, ast.ASTRef) {
		h, e := manager.MakeVariable(name, boolSort)
		interpMust(t, e)
		ref, e := m.NewRef(h)
		interpMust(t, e)
		return h, ref
	}
	first, firstRef := makeOwned("proof-itp-first")
	second, secondRef := makeOwned("proof-itp-second")
	conclusion, conclusionRef := makeOwned("proof-conclusion")
	node, err := proof.MakeLemma([]ast.ASTHandle{conclusion}, []ast.ASTHandle{first, second})
	interpMust(t, err)
	if node != 0 {
		t.Fatal("lemma node", node)
	}
	gotConclusion, err := proof.Conclusion(node)
	interpMust(t, err)
	if len(gotConclusion) != 1 || gotConclusion[0] != conclusion {
		t.Fatal("lemma conclusion", gotConclusion)
	}
	gotConclusion[0] = a
	again, err := proof.Conclusion(node)
	interpMust(t, err)
	if again[0] != conclusion {
		t.Fatal("conclusion slice aliases proof storage")
	}

	interpMust(t, m.DelRef(&firstRef))
	interpMust(t, m.DelRef(&secondRef))
	interpMust(t, m.DelRef(&conclusionRef))

	selected0, err := proof.Interpolate(node, DownwardScopeRange(0), false)
	interpMust(t, err)
	got0, err := m.RefGet(selected0)
	interpMust(t, err)
	if got0 != first {
		t.Fatal("first interpolant", got0)
	}
	interpMust(t, m.DelRef(&selected0))
	selected1, err := proof.Interpolate(node, DownwardScopeRange(1), true)
	interpMust(t, err)
	got1, err := m.RefGet(selected1)
	interpMust(t, err)
	if got1 != second {
		t.Fatal("second interpolant", got1)
	}
	interpMust(t, m.DelRef(&selected1))
}

func TestInterpolationProofGuardsAndClose(t *testing.T) {
	_, manager := interpProofManagerFixture(t)
	base, err := NewBase(manager, nil, nil, nil)
	interpMust(t, err)
	proof, err := NewInterpolationProof(base)
	interpMust(t, err)
	if _, err = proof.Interpolate(0, DownwardScopeRange(0), false); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("node guard", err)
	}
	node, err := proof.MakeLemma(nil, nil)
	interpMust(t, err)
	if _, err = proof.Interpolate(node, DownwardScopeRange(0), false); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("frame guard", err)
	}
	copy := *proof
	if err = copy.Close(); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("copy guard", err)
	}
	interpMust(t, proof.Close())
	if _, err = proof.MakeLemma(nil, nil); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("closed proof guard", err)
	}
	interpMust(t, base.Close())
	if (ProofError{}).Error() != "proof_error" || NullProofNode != -1 {
		t.Fatal("proof error contract")
	}
}
