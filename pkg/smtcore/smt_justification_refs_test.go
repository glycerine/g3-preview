package smtcore

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

type smtTestJustification struct{}

func (*smtTestJustification) smtJustification() {}

func TestSMTCompactBooleanAndEqualityJustifications(t *testing.T) {
	if NullSMTBooleanJustification().Kind() != SMTBooleanJustificationClause || NewSMTBooleanAxiom().Kind() != SMTBooleanJustificationAxiom {
		t.Fatal("Boolean source defaults")
	}
	clause := &SMTClause{}
	clauseValue := NewSMTBooleanClause(clause)
	if got, ok := clauseValue.Clause(); !ok || got != clause {
		t.Fatal("clause payload", got, ok)
	}
	literal := NewSMTLiteral(7, true)
	binary := NewSMTBooleanBinaryClause(literal)
	if got, ok := binary.Literal(); !ok || got != literal || binary.Kind().String() != "bin_clause" {
		t.Fatal("binary payload", got, ok)
	}
	object := &smtTestJustification{}
	theory, err := NewSMTBooleanJustification(object)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := theory.Justification(); !ok || got != object || theory.Kind().String() != "theory" {
		t.Fatal("theory payload", got, ok)
	}
	if _, err = NewSMTBooleanJustification(nil); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("nil Boolean justification", err)
	}
	for kind, text := range map[SMTBooleanJustificationKind]string{SMTBooleanJustificationClause: "clause", SMTBooleanJustificationBinaryClause: "bin_clause", SMTBooleanJustificationAxiom: "axiom", SMTBooleanJustificationObject: "theory"} {
		if kind.String() != text {
			t.Fatal(kind, kind.String(), text)
		}
	}
	if SMTBooleanJustificationKind(255).String() != "" {
		t.Fatal("unknown Boolean justification display")
	}
	if NewSMTEqualityAxiom().Kind() != SMTEqualityJustificationAxiom || NullSMTEqualityJustification().Kind() != SMTEqualityJustificationObject {
		t.Fatal("equality source defaults")
	}
	congruence := NewSMTEqualityCongruence(true)
	if commuted, ok := congruence.UsedCommutativity(); !ok || !commuted {
		t.Fatal("congruence payload", commuted, ok)
	}
	equation := NewSMTEqualityEquation(literal)
	if got, ok := equation.Literal(); !ok || got != literal {
		t.Fatal("equation payload", got, ok)
	}
	fallback := NewSMTEqualityJustification(object)
	if got, ok := fallback.Justification(); !ok || got != object {
		t.Fatal("equality fallback", got, ok)
	}
}

func TestSMTBoolVarDataSourceFlagTransitions(t *testing.T) {
	var data SMTBoolVarData
	data.Initialize(19)
	if data.InternalLevel() != 19 || data.IsAtom() || data.Justification().Kind() != SMTBooleanJustificationClause || data.ScopeLevel != 0 || data.Mark || data.Assumption || data.PhaseAvailable || data.Phase {
		t.Fatal("initial state", data)
	}
	data.SetAxiom()
	if data.Justification().Kind() != SMTBooleanJustificationAxiom {
		t.Fatal("axiom")
	}
	data.SetEquality()
	data.SetENode()
	data.SetQuantifier()
	if !data.IsAtom() || !data.IsEquality() || !data.IsENode() || !data.IsQuantifier() {
		t.Fatal("atom flags")
	}
	data.ResetEquality()
	data.ResetENode()
	if !data.IsAtom() {
		t.Fatal("quantifier must retain atom")
	}
	if err := data.SetNotifyTheory(7); err != nil || data.Theory() != 7 || !data.IsTheoryAtom() {
		t.Fatal("theory atom", data.Theory(), err)
	}
	if err := data.SetNotifyTheory(0); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("zero theory", err)
	}
	if err := data.SetNotifyTheory(256); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("wide theory", err)
	}
	data.ResetNotifyTheory()
	if data.IsTheoryAtom() || data.Theory() != SMTNullTheoryID || !data.IsAtom() {
		t.Fatal("theory reset with quantifier")
	}
	data.SetTrueFirst()
	if !data.TryTrueFirst() {
		t.Fatal("true first")
	}
	data.ResetTrueFirst()
	if data.TryTrueFirst() {
		t.Fatal("true first reset")
	}
	data.Initialize(3)
	if data.InternalLevel() != 3 || data.IsAtom() || data.TryTrueFirst() {
		t.Fatal("reinitialize", data)
	}
}
