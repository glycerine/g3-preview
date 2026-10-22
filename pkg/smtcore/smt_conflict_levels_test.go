package smtcore

import "testing"

func TestSMTConflictJustificationMaximumAssignmentLevels(t *testing.T) {
	levels := []uint32{0, 2, 7, 4, 9, 3}
	expander := NewSMTAntecedentExpander()
	simple, err := NewSMTSimpleJustification(true, []SMTLiteral{NewSMTLiteral(1, false), NewSMTLiteral(3, true)})
	if err != nil {
		t.Fatal(err)
	}
	level, err := SMTJustificationMaxLevel(expander, simple, levels)
	if err != nil || level != 4 {
		t.Fatal("simple maximum", level, err)
	}
	object, err := NewSMTBooleanJustification(simple)
	if err != nil {
		t.Fatal(err)
	}
	level, err = SMTBooleanJustificationMaxLevel(expander, NewSMTLiteral(2, false), object, levels)
	if err != nil || level != 7 {
		t.Fatal("object maximum with consequent", level, err)
	}
	level, err = SMTBooleanJustificationMaxLevel(expander, SMTFalseLiteral, NewSMTBooleanBinaryClause(NewSMTLiteral(4, true)), levels)
	if err != nil || level != 9 {
		t.Fatal("binary maximum", level, err)
	}
}

func TestSMTConflictClauseMaximumLevelWatchConsequentAndAttachedJustification(t *testing.T) {
	manager := astTestManager(t)
	attached, err := NewSMTSimpleJustification(false, []SMTLiteral{NewSMTLiteral(5, false)})
	if err != nil {
		t.Fatal(err)
	}
	literals := []SMTLiteral{NewSMTLiteral(1, false), NewSMTLiteral(2, false), NewSMTLiteral(3, false), NewSMTLiteral(4, false)}
	clause, err := NewSMTClause(manager, literals, SMTClauseLearned, attached, nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	levels := []uint32{0, 2, 8, 6, 10, 12}
	expander := NewSMTAntecedentExpander()
	level, err := SMTBooleanJustificationMaxLevel(expander, literals[1], NewSMTBooleanClause(clause), levels)
	if err != nil || level != 12 {
		t.Fatal("clause maximum", level, err)
	}
	level, err = SMTBooleanJustificationMaxLevel(expander, SMTFalseLiteral, NewSMTBooleanClause(clause), levels)
	if err != nil || level != 12 {
		t.Fatal("conflict clause maximum", level, err)
	}
	if err = clause.Close(); err != nil {
		t.Fatal(err)
	}
}
