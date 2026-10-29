package core

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
)

func TestNNFTacticFullModeProofComposition(t *testing.T) {
	m, context, globals := simplifyFixture(t)
	simplifyMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	a, b, _ := propagateAtoms(t, m)
	conjunction, err := m.MkAnd(a, b)
	simplifyMust(t, err)
	formula, err := m.MkNot(conjunction)
	simplifyMust(t, err)
	proof, err := m.MkProofAsserted(formula)
	simplifyMust(t, err)
	goal, err := tactic.NewGoal(m, true, false, false)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	simplifyMust(t, goal.AssertWithProof(formula, proof, ast.ASTDependency{}))
	runner, err := NewNNFTactic(m, context, globals, nil)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, runner.Close()) })
	var result tactic.TacticResult
	t.Cleanup(func() { simplifyMust(t, result.Close()) })
	simplifyMust(t, runner.Apply(goal, &result))
	if size, err := goal.Size(); err != nil || size != 1 {
		t.Fatal("nnf result size", size, err)
	}
	converted, err := goal.Form(0)
	simplifyMust(t, err)
	if family, err := m.GetFamilyID(converted); err != nil || family != m.BasicFamilyID() {
		t.Fatal("nnf family", family, err)
	}
	if operation, err := m.GetDeclKind(converted); err != nil || operation != ast.ASTOpOr {
		t.Fatal("nnf operation", operation, err)
	}
	combined, err := goal.Proof(0)
	simplifyMust(t, err)
	if combined.IsNull() {
		t.Fatal("nnf proof missing")
	}
}
