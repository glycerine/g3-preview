package core

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestDERTacticUnitQuantifierAndOwnership(t *testing.T) {
	m, _, _ := simplifyFixture(t)
	boolean, err := m.MkBoolSort()
	simplifyMust(t, err)
	variable, err := m.MkVar(0, boolean)
	simplifyMust(t, err)
	name, err := m.Symbols.Intern("der-x")
	simplifyMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []usymbol.Symbol{name}, Body: variable})
	simplifyMust(t, err)
	goal, err := tactic.NewGoal(m, false, false, false)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	simplifyMust(t, goal.Assert(quantifier))
	runner, err := NewDERTactic(m)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, runner.Close()) })
	var result tactic.TacticResult
	t.Cleanup(func() { simplifyMust(t, result.Close()) })
	simplifyMust(t, runner.Apply(goal, &result))
	if len(result.Goals) != 1 || result.Goals[0] != goal {
		t.Fatal("DER did not publish its mutated input goal", len(result.Goals))
	}
	inconsistent, err := goal.Inconsistent()
	simplifyMust(t, err)
	if !inconsistent {
		t.Fatal("unit universal DER did not derive false")
	}
	if depth, err := goal.Depth(); err != nil || depth != 1 {
		t.Fatal("DER depth", depth, err)
	}
}

func TestDERTacticProofComposition(t *testing.T) {
	m, _, _ := simplifyFixture(t)
	simplifyMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	boolean, err := m.MkBoolSort()
	simplifyMust(t, err)
	variable, err := m.MkVar(0, boolean)
	simplifyMust(t, err)
	name, err := m.Symbols.Intern("der-proof-x")
	simplifyMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []usymbol.Symbol{name}, Body: variable})
	simplifyMust(t, err)
	proof, err := m.MkProofAsserted(quantifier)
	simplifyMust(t, err)
	goal, err := tactic.NewGoal(m, true, false, false)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	simplifyMust(t, goal.AssertWithProof(quantifier, proof, ast.ASTDependency{}))
	runner, err := NewDERTactic(m)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, runner.Close()) })
	var result tactic.TacticResult
	t.Cleanup(func() { simplifyMust(t, result.Close()) })
	simplifyMust(t, runner.Apply(goal, &result))
	combined, err := goal.Proof(0)
	simplifyMust(t, err)
	if undefined, err := m.IsUndefProof(combined); err != nil || undefined {
		t.Fatal("DER proof composition", combined, undefined, err)
	}
}
