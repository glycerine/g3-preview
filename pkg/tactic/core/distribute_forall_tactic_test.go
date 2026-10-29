package core

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestDistributeForallConjunctionAndNegatedDisjunction(t *testing.T) {
	for _, negated := range []bool{false, true} {
		t.Run(map[bool]string{false: "and", true: "not-or"}[negated], func(t *testing.T) {
			m, _, _ := simplifyFixture(t)
			boolean, err := m.MkBoolSort()
			simplifyMust(t, err)
			variable, err := m.MkVar(0, boolean)
			simplifyMust(t, err)
			name, err := m.Symbols.Intern("distribute-p")
			simplifyMust(t, err)
			constant, err := m.MkConst(name, boolean)
			simplifyMust(t, err)
			var body ast.ASTHandle
			if negated {
				disjunction, e := m.MkOr(variable, constant)
				simplifyMust(t, e)
				body, err = m.MkNot(disjunction)
			} else {
				body, err = m.MkAnd(variable, constant)
			}
			simplifyMust(t, err)
			bound, err := m.Symbols.Intern("distribute-x")
			simplifyMust(t, err)
			quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []usymbol.Symbol{bound}, Body: body})
			simplifyMust(t, err)
			goal, err := tactic.NewGoal(m, false, false, false)
			simplifyMust(t, err)
			t.Cleanup(func() { simplifyMust(t, goal.Close()) })
			simplifyMust(t, goal.Assert(quantifier))
			runner, err := NewDistributeForallTactic(m, nil)
			simplifyMust(t, err)
			t.Cleanup(func() { simplifyMust(t, runner.Close()) })
			var result tactic.TacticResult
			t.Cleanup(func() { simplifyMust(t, result.Close()) })
			simplifyMust(t, runner.Apply(goal, &result))
			if size, err := goal.Size(); err != nil || size != 2 {
				t.Fatal("distributed goal conjunct count", size, err)
			}
			quantified, err := goal.Form(0)
			simplifyMust(t, err)
			if kind, err := m.GetKind(quantified); err != nil || kind != ast.ASTQuantifier {
				t.Fatal("distributed quantified conjunct", kind, err)
			}
			ground, err := goal.Form(1)
			simplifyMust(t, err)
			if negated {
				if not, child, err := m.IsNot(ground); err != nil || !not || child != constant {
					t.Fatal("negated ground conjunct", not, child, err)
				}
			} else if ground != constant {
				t.Fatal("ground conjunct did not shed unused quantifier", ground, constant)
			}
		})
	}
}

func TestDistributeForallProofComposition(t *testing.T) {
	m, _, _ := simplifyFixture(t)
	simplifyMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	boolean, err := m.MkBoolSort()
	simplifyMust(t, err)
	variable, err := m.MkVar(0, boolean)
	simplifyMust(t, err)
	name, err := m.Symbols.Intern("distribute-proof-p")
	simplifyMust(t, err)
	constant, err := m.MkConst(name, boolean)
	simplifyMust(t, err)
	body, err := m.MkAnd(variable, constant)
	simplifyMust(t, err)
	bound, err := m.Symbols.Intern("distribute-proof-x")
	simplifyMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []usymbol.Symbol{bound}, Body: body})
	simplifyMust(t, err)
	proof, err := m.MkProofAsserted(quantifier)
	simplifyMust(t, err)
	goal, err := tactic.NewGoal(m, true, false, false)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	simplifyMust(t, goal.AssertWithProof(quantifier, proof, ast.ASTDependency{}))
	runner, err := NewDistributeForallTactic(m, nil)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, runner.Close()) })
	var result tactic.TacticResult
	t.Cleanup(func() { simplifyMust(t, result.Close()) })
	simplifyMust(t, runner.Apply(goal, &result))
	for i := uint32(0); i < 2; i++ {
		combined, err := goal.Proof(i)
		simplifyMust(t, err)
		if undefined, err := m.IsUndefProof(combined); err != nil || undefined {
			t.Fatal("distributed proof", i, combined, undefined, err)
		}
	}
}
