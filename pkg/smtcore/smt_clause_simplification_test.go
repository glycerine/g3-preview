package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTClauseSimplificationDropsFalseTailLiteral(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	expressions := []ast.ASTHandle{
		smtRelevancyBool(t, m, "simplify-tail-a"),
		smtRelevancyBool(t, m, "simplify-tail-b"),
		smtRelevancyBool(t, m, "simplify-tail-c"),
	}
	literals := make([]SMTLiteral, len(expressions))
	for index, expression := range expressions {
		astTestMust(t, state.Internalize(expression, true))
		variable, _ := state.BoolVar(expression)
		literals[index] = NewSMTLiteral(variable, false)
	}
	astTestMust(t, state.addClauseKind(literals, false, SMTClauseLearned, ast.ASTHandle{}))
	astTestMust(t, state.AssignLiteral(literals[2].Negated(), NewSMTBooleanAxiom()))
	consistent, err := state.PropagateCore()
	astTestMust(t, err)
	if !consistent {
		t.Fatal("unexpected conflict before simplification")
	}
	astTestMust(t, state.simplifyClausesAtBase(0))
	if len(state.clauses) != 1 || len(state.clauses[0].clause.literals) != 2 {
		t.Fatal("false tail literal was not removed")
	}
	for _, literal := range state.clauses[0].clause.literals {
		if literal == literals[2] {
			t.Fatal("false tail literal survived simplification")
		}
	}
}

func TestSMTClauseSimplificationDeletesSatisfiedReasonSafely(t *testing.T) {
	m := astTestManager(t)
	_, _, state := smtInternalizerFixture(t, m, nil)
	first := smtRelevancyBool(t, m, "simplify-reason-a")
	second := smtRelevancyBool(t, m, "simplify-reason-b")
	for _, expression := range []ast.ASTHandle{first, second} {
		astTestMust(t, state.Internalize(expression, true))
	}
	firstVar, _ := state.BoolVar(first)
	secondVar, _ := state.BoolVar(second)
	firstLiteral := NewSMTLiteral(firstVar, false)
	secondLiteral := NewSMTLiteral(secondVar, false)
	astTestMust(t, state.addClause([]SMTLiteral{firstLiteral, secondLiteral}, false, ast.ASTHandle{}))
	astTestMust(t, state.AssignLiteral(secondLiteral.Negated(), NewSMTBooleanAxiom()))
	consistent, err := state.PropagateCore()
	astTestMust(t, err)
	if !consistent {
		t.Fatal("unexpected propagation conflict")
	}
	if clause, ok := state.booleanData[firstVar].Justification().Clause(); !ok || clause == nil {
		t.Fatal("clause did not justify propagated literal")
	}
	astTestMust(t, state.simplifyClausesAtBase(0))
	if len(state.clauses) != 0 {
		t.Fatal("satisfied clause was not deleted")
	}
	if state.booleanData[firstVar].Justification().Kind() != SMTBooleanJustificationAxiom {
		t.Fatal("deleted clause remained an assignment reason")
	}
}

func TestSMTClauseSimplificationRetainsProofReasons(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	_, _, state := smtInternalizerFixture(t, m, nil)
	first := smtRelevancyBool(t, m, "simplify-proof-a")
	second := smtRelevancyBool(t, m, "simplify-proof-b")
	for _, expression := range []ast.ASTHandle{first, second} {
		astTestMust(t, state.Internalize(expression, true))
	}
	firstVar, _ := state.BoolVar(first)
	secondVar, _ := state.BoolVar(second)
	firstLiteral := NewSMTLiteral(firstVar, false)
	secondLiteral := NewSMTLiteral(secondVar, false)
	fact, err := m.MkOr(first, second)
	astTestMust(t, err)
	proof, err := m.MkProofHypothesis(fact)
	astTestMust(t, err)
	astTestMust(t, state.addClause([]SMTLiteral{firstLiteral, secondLiteral}, false, proof))
	astTestMust(t, state.AssignLiteral(secondLiteral.Negated(), NewSMTBooleanAxiom()))
	consistent, err := state.PropagateCore()
	astTestMust(t, err)
	if !consistent {
		t.Fatal("unexpected propagation conflict")
	}
	astTestMust(t, state.simplifyClausesAtBase(0))
	justification, ok := state.booleanData[firstVar].Justification().Justification()
	if !ok {
		t.Fatal("proof reason was replaced by an axiom")
	}
	unit, ok := justification.(*SMTUnitResolutionJustification)
	if !ok || len(unit.Literals) != 1 || unit.Literals[0] != secondLiteral.Negated() {
		t.Fatal("unexpected simplified proof reason", justification)
	}
	if wrapper, ok := unit.Antecedent.(*SMTJustificationProofWrapper); !ok {
		t.Fatal("clause proof was not retained")
	} else if _, err = wrapper.Proof(); err != nil {
		t.Fatal("retained clause proof is closed", err)
	}
}
