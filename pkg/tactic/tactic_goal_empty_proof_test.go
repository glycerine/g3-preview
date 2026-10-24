package tactic

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

// Source goal::update intentionally disables the proof-presence assertion.
// card2bv is a retained caller that can install a null proof in a proof goal.
func TestTacticGoalUpdateCopyAndTranslationPreserveEmptyProof(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	g := goalTestNew(t, m, true, false)
	p := goalTestAtom(t, m, "empty-proof-p")
	q := goalTestAtom(t, m, "empty-proof-q")
	astTestMust(t, g.Assert(p))
	astTestMust(t, g.Update(0, q, ast.ASTHandle{}, ast.ASTDependency{}))
	proof, err := g.Proof(0)
	astTestMust(t, err)
	if !proof.IsNull() {
		t.Fatal("update invented a proof")
	}
	copy, err := NewGoalCopy(g)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, copy.Close()) })
	goalTestForms(t, copy, q)
	proof, err = copy.Proof(0)
	astTestMust(t, err)
	if !proof.IsNull() {
		t.Fatal("copy invented a proof")
	}
	target, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: m.Symbols, Numerals: m.NumeralService(), Budget: m.Budget(), ProofMode: ast.ASTProofEnabled})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, target.Close()) })
	tr, err := ast.NewASTTranslation(m, target, true)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, tr.Close()) })
	translated, err := g.Translate(tr)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, translated.Close()) })
	proof, err = translated.Proof(0)
	astTestMust(t, err)
	if !proof.IsNull() {
		t.Fatal("translation invented a proof")
	}
}
