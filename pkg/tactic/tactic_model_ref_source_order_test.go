package tactic

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
)

func TestTacticModelRefKeepsOldOnConsumedDestructionStop(t *testing.T) {
	m := astTestManager(t)
	old, e := model.NewModel(m)
	satOK(t, e)
	s, e := m.MkUninterpretedSort(astTestSymbol(t, m, "model-ref-source-order"), nil)
	satOK(t, e)
	v, e := m.GetSomeValue(s)
	satOK(t, e)
	satOK(t, old.RegisterUniverse(s, []ast.ASTHandle{v}))
	incoming, e := model.NewModelCore(m)
	satOK(t, e)
	var r ModelRef
	satOK(t, r.Set(&old.CoreField))
	satOK(t, incoming.IncRef())
	m.ExprIDs.Cleanup(m.ExprIDs.Range())
	m.BudgetField.FailAfter(0)
	e = r.Set(incoming)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatalf("old final destruction refusal: %v", e)
	}
	if r.value != &old.CoreField || r.owned || len(r.pending) != 1 || r.pending[0] != incoming || old.CoreField.References != 0 || incoming.References != 2 {
		t.Fatalf("source raw pointer and acquired incoming ref: ref %+v, old %d, incoming %d", r, old.CoreField.References, incoming.References)
	}
	satOK(t, r.Close())
	if !r.closed || !old.Closed || incoming.References != 1 {
		t.Fatal("owning cleanup of stopped assignment", r, incoming.References)
	}
	satOK(t, old.Close())
	satOK(t, incoming.DecRef())
}

func TestTacticModelRefCloseResumesConsumedFinalModelCleanup(t *testing.T) {
	m := astTestManager(t)
	md, e := model.NewModel(m)
	satOK(t, e)
	s, e := m.MkUninterpretedSort(astTestSymbol(t, m, "model-ref-close-stop"), nil)
	satOK(t, e)
	v, e := m.GetSomeValue(s)
	satOK(t, e)
	satOK(t, md.RegisterUniverse(s, []ast.ASTHandle{v}))
	var r ModelRef
	satOK(t, r.Set(&md.CoreField))
	m.ExprIDs.Cleanup(m.ExprIDs.Range())
	m.BudgetField.FailAfter(0)
	e = r.Close()
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || r.closed || r.owned || r.cleanup != &md.CoreField || md.CoreField.References != 0 {
		t.Fatal("consumed final model release", e, r, md.CoreField.References)
	}
	satOK(t, r.Close())
	if !r.closed || !md.Closed || r.cleanup != nil {
		t.Fatal("model cleanup did not resume", r, md.Closed)
	}
}

func TestTacticModelRefPendingIncomingFinalCleanupResumes(t *testing.T) {
	m := astTestManager(t)
	old, e := model.NewModel(m)
	satOK(t, e)
	oldSort, e := m.MkUninterpretedSort(astTestSymbol(t, m, "pending-old-sort"), nil)
	satOK(t, e)
	oldValue, e := m.GetSomeValue(oldSort)
	satOK(t, e)
	satOK(t, old.RegisterUniverse(oldSort, []ast.ASTHandle{oldValue}))
	incoming, e := model.NewModel(m)
	satOK(t, e)
	incomingSort, e := m.MkUninterpretedSort(astTestSymbol(t, m, "pending-incoming-sort"), nil)
	satOK(t, e)
	incomingValue, e := m.GetSomeValue(incomingSort)
	satOK(t, e)
	satOK(t, incoming.RegisterUniverse(incomingSort, []ast.ASTHandle{incomingValue}))
	var r ModelRef
	satOK(t, r.Set(&old.CoreField))
	m.ExprIDs.Cleanup(m.ExprIDs.Range())
	m.BudgetField.FailAfter(0)
	e = r.Set(&incoming.CoreField)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || r.cleanup != &old.CoreField || len(r.pending) != 1 || incoming.CoreField.References != 1 {
		t.Fatal("old final cleanup prefix", e, r, incoming.CoreField.References)
	}
	// Finish the old model so the next refusal is reached by the independently
	// acquired incoming model while ModelRef.Close releases its pending receipt.
	satOK(t, old.Close())
	m.ExprIDs.Cleanup(m.ExprIDs.Range())
	m.BudgetField.FailAfter(0)
	e = r.Close()
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || r.closed || r.cleanup != &incoming.CoreField || len(r.pending) != 0 || incoming.CoreField.References != 0 {
		t.Fatal("incoming final cleanup prefix", e, r, incoming.CoreField.References)
	}
	satOK(t, r.Close())
	if !r.closed || !incoming.Closed || r.cleanup != nil {
		t.Fatal("pending incoming model cleanup", r, incoming.Closed)
	}
}

func TestTacticStoredModelCopyAssignmentKeepsSourceOrderOnStoppedDeletion(t *testing.T) {
	m := astTestManager(t)
	old, e := model.NewModel(m)
	satOK(t, e)
	s, e := m.MkUninterpretedSort(astTestSymbol(t, m, "stored-model-source-order"), nil)
	satOK(t, e)
	v, e := m.GetSomeValue(s)
	satOK(t, e)
	satOK(t, old.RegisterUniverse(s, []ast.ASTHandle{v}))
	incoming, e := model.NewModelCore(m)
	satOK(t, e)
	stored, e := NewStoredModel(incoming)
	satOK(t, e)
	var out ModelRef
	satOK(t, out.Set(&old.CoreField))
	m.ExprIDs.Cleanup(m.ExprIDs.Range())
	m.BudgetField.FailAfter(0)
	e = stored.ApplyModel(&out, 0)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatalf("model2mc old model deletion refusal: %v", e)
	}
	if out.value != &old.CoreField || out.owned || len(out.pending) != 1 || out.pending[0] != incoming || incoming.References != 2 {
		t.Fatalf("model2mc source copy assignment: %+v, incoming %d", out, incoming.References)
	}
	satOK(t, out.Close())
	if !old.Closed || incoming.References != 1 {
		t.Fatal("model2mc incoming owner lost after abort cleanup", incoming.References)
	}
	satOK(t, old.Close())
	satOK(t, stored.Close())
	if !incoming.Closed {
		t.Fatal("stored model reference leaked")
	}
}

func TestTacticFilterModelStoppedAssignmentKeepsAcquiredNextModel(t *testing.T) {
	m := astTestManager(t)
	old, e := model.NewModel(m)
	satOK(t, e)
	d := modelTestDecl(t, m, "filtered-source-order")
	b, e := m.MkBoolSort()
	satOK(t, e)
	v, e := m.MkVar(11741, b)
	satOK(t, e)
	satOK(t, old.CoreField.RegisterConstant(d, v))
	filter, e := NewFilterModel(m)
	satOK(t, e)
	satOK(t, filter.Insert(d))
	var out ModelRef
	satOK(t, out.Set(&old.CoreField))
	m.ExprIDs.Cleanup(m.ExprIDs.Range())
	m.BudgetField.FailAfter(1) // new model header, then old AST ID recycle
	e = filter.ApplyModel(&out, 0)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatalf("filtered old model deletion refusal: %v", e)
	}
	if out.value != &old.CoreField || out.owned || len(out.pending) != 1 {
		t.Fatalf("source output publication and next receipt: %+v", out)
	}
	next := out.pending[0]
	if next.References != 1 || next.Closed || next.DerivedModel() == nil {
		t.Fatalf("filter prematurely destroyed acquired next model: refs %d closed %v", next.References, next.Closed)
	}
	satOK(t, out.Close())
	if !next.Closed || !old.Closed {
		t.Fatal("output owning cleanup retained next model")
	}
	satOK(t, old.Close())
	satOK(t, filter.Close())
}

func TestTacticModelConverterToModelStoppedAssignmentKeepsAcquiredModel(t *testing.T) {
	m := astTestManager(t)
	old, e := model.NewModel(m)
	satOK(t, e)
	s, e := m.MkUninterpretedSort(astTestSymbol(t, m, "converter-to-model-source-order"), nil)
	satOK(t, e)
	v, e := m.GetSomeValue(s)
	satOK(t, e)
	satOK(t, old.RegisterUniverse(s, []ast.ASTHandle{v}))
	var out ModelRef
	satOK(t, out.Set(old.Core()))
	calls := []string{}
	converter := &modelConverterTestLeaf{name: "unreached", calls: &calls}
	m.ExprIDs.Cleanup(m.ExprIDs.Range())
	m.BudgetField.FailAfter(1) // new model header, then old AST ID recycle
	e = ModelConverterToModel(m, converter, &out)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatalf("old model deletion refusal: %v", e)
	}
	if len(calls) != 0 || out.value != old.Core() || out.owned || len(out.pending) != 1 {
		t.Fatalf("converter callback or source raw publication: calls %v, ref %+v", calls, out)
	}
	next := out.pending[0]
	if next.References != 1 || next.Closed || next.DerivedModel() == nil {
		t.Fatalf("new model destroyed despite output receipt: refs %d closed %v", next.References, next.Closed)
	}
	satOK(t, out.Close())
	if !next.Closed || !old.Closed {
		t.Fatal("output cleanup did not retire acquired new model")
	}
	satOK(t, old.Close())
}
