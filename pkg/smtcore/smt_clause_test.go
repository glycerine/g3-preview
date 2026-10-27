package smtcore

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

type smtClauseTestJustification struct {
	SMTJustificationBase
	region   bool
	releases int
}

func (j *smtClauseTestJustification) InRegion() bool { return j.region }
func (j *smtClauseTestJustification) Release(*ast.ASTManager) error {
	j.releases++
	return nil
}

type smtClauseTestDeleteHandler struct{ calls int }

func (h *smtClauseTestDeleteHandler) ClauseDeleted(*ast.ASTManager, *SMTClause) error {
	h.calls++
	return nil
}

func smtClauseTestAtom(t *testing.T, manager *ast.ASTManager, name uint32, boolean ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	declaration, err := manager.MkFuncDecl(symbol.Numerical(name), nil, boolean, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	result, err := manager.MkApp(declaration, nil)
	astTestMust(t, err)
	return result
}

func TestSMTClauseConstructionAtomsActivityDisplayAndClose(t *testing.T) {
	manager := astTestManager(t)
	boolean, err := manager.MkBoolSort()
	astTestMust(t, err)
	first := smtClauseTestAtom(t, manager, 2501, boolean)
	second := smtClauseTestAtom(t, manager, 2502, boolean)
	expressions := []ast.ASTHandle{{}, first, second}
	literals := []SMTLiteral{NewSMTLiteral(1, false), NewSMTLiteral(2, true)}
	justification := &smtClauseTestJustification{}
	handler := &smtClauseTestDeleteHandler{}
	clause, err := NewSMTClause(manager, literals, SMTClauseLearned, justification, handler, true, expressions)
	astTestMust(t, err)
	literals[0] = SMTFalseLiteral
	if !clause.IsLemma() || !clause.IsLearned() || clause.IsAuxLemma() || !clause.InReinitStack() || !clause.ReinternalizeAtoms() || clause.NumAtoms() != 2 {
		t.Fatal("source clause flags")
	}
	if got, err := clause.Literal(0); err != nil || got != NewSMTLiteral(1, false) {
		t.Fatal("owned literal copy", got, err)
	}
	if activity, err := clause.Activity(); err != nil || activity != 1 {
		t.Fatal("initial lemma activity", activity, err)
	}
	astTestMust(t, clause.IncrementActivity())
	if activity, _ := clause.Activity(); activity != 2 {
		t.Fatal("incremented activity", activity)
	}
	if atom, sign, err := clause.Atom(1); err != nil || atom != second || !sign {
		t.Fatal("saved atom", atom, sign, err)
	}
	firstID, err := manager.GetASTID(first)
	astTestMust(t, err)
	secondID, err := manager.GetASTID(second)
	astTestMust(t, err)
	var compact bytes.Buffer
	astTestMust(t, clause.DisplayCompact(&compact, expressions))
	if want := fmt.Sprintf("(clause #%d (not #%d))", firstID, secondID); compact.String() != want {
		t.Fatal("compact clause", compact.String(), want)
	}
	var verbose bytes.Buffer
	astTestMust(t, clause.Display(&verbose, expressions))
	if verbose.String() != "(clause k!2501 (not k!2502))" {
		t.Fatal("verbose clause", verbose.String())
	}
	if !clause.IsWatch(NewSMTLiteral(2, true)) {
		t.Fatal("watch literal")
	}
	astTestMust(t, clause.SwapLiterals(0, 1))
	if got := clause.Literals(); got[0] != NewSMTLiteral(2, true) || got[1] != NewSMTLiteral(1, false) {
		t.Fatal("literal swap", got)
	}
	astTestMust(t, clause.MarkDeleted())
	if !clause.Deleted() || handler.calls != 1 {
		t.Fatal("delete callback", clause.Deleted(), handler.calls)
	}
	astTestMust(t, clause.Close())
	if justification.releases != 1 || handler.calls != 1 {
		t.Fatal("close ownership", justification.releases, handler.calls)
	}
	astTestMust(t, clause.Close())
}

func TestSMTClauseMutationAndConstructionGuards(t *testing.T) {
	manager := astTestManager(t)
	aux, err := NewSMTClause(manager, []SMTLiteral{SMTTrueLiteral, SMTFalseLiteral, NewSMTLiteral(1, false)}, SMTClauseAux, nil, nil, false, nil)
	astTestMust(t, err)
	if _, err = aux.Activity(); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("aux activity", err)
	}
	astTestMust(t, aux.SetLiteral(2, NewSMTLiteral(2, false)))
	astTestMust(t, aux.SetNumLiterals(2))
	if len(aux.Literals()) != 2 {
		t.Fatal("literal shrink")
	}
	if err = aux.SetNumLiterals(3); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("literal growth", err)
	}
	astTestMust(t, aux.Close())

	region := &smtClauseTestJustification{region: true}
	if _, err = NewSMTClause(manager, []SMTLiteral{SMTTrueLiteral, SMTFalseLiteral}, SMTClauseLearned, region, nil, false, nil); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("region lemma", err)
	}
	if _, err = NewSMTClause(manager, []SMTLiteral{SMTTrueLiteral}, SMTClauseAux, nil, nil, false, nil); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("short clause", err)
	}
	astTestMust(t, manager.ToggleProofMode(ast.ASTProofEnabled))
	if _, err = NewSMTClause(manager, []SMTLiteral{SMTTrueLiteral, SMTFalseLiteral}, SMTClauseAux, nil, nil, false, nil); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("proof clause without justification", err)
	}
}
