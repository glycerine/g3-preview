package tactic

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
)

func TestTacticModelRefReplacesOldWithNativeIDRecycle(t *testing.T) {
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
	satOK(t, e)
	if r.value != incoming || !r.owned || len(r.pending) != 0 || !old.Closed || incoming.References != 2 {
		t.Fatalf("source old-then-new publication: ref %+v, old closed %v, incoming %d", r, old.Closed, incoming.References)
	}
	satOK(t, r.Close())
	if !r.closed || !old.Closed || incoming.References != 1 {
		t.Fatal("owning cleanup of stopped assignment", r, incoming.References)
	}
	satOK(t, old.Close())
	satOK(t, incoming.DecRef())
}

func TestTacticModelRefCloseUsesNativeIDRecycle(t *testing.T) {
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
	satOK(t, e)
	if !r.closed || !md.Closed || r.cleanup != nil || md.CoreField.References != 0 {
		t.Fatal("native final model release", r, md.Closed)
	}
}

func TestTacticModelRefReplacesTwoDerivedModelsWithoutIDBudget(t *testing.T) {
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
	satOK(t, e)
	if !old.Closed || r.value != &incoming.CoreField || !r.owned || len(r.pending) != 0 || incoming.CoreField.References != 1 {
		t.Fatal("old released before incoming publication", r, incoming.CoreField.References)
	}
	m.ExprIDs.Cleanup(m.ExprIDs.Range())
	m.BudgetField.FailAfter(0)
	e = r.Close()
	m.BudgetField.FailAfter(-1)
	satOK(t, e)
	if !r.closed || !incoming.Closed || r.cleanup != nil || incoming.CoreField.References != 0 {
		t.Fatal("incoming final model cleanup", r, incoming.Closed)
	}
}

func TestTacticStoredModelCopyAssignmentUsesNativeIDRecycle(t *testing.T) {
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
	satOK(t, e)
	if out.value != incoming || !out.owned || len(out.pending) != 0 || !old.Closed || incoming.References != 2 {
		t.Fatalf("model2mc source copy assignment: %+v, old closed %v, incoming %d", out, old.Closed, incoming.References)
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

func TestTacticFilterModelAssignmentUsesNativeIDRecycle(t *testing.T) {
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
	m.BudgetField.FailAfter(1) // new model header; AST ID recycle is native
	e = filter.ApplyModel(&out, 0)
	m.BudgetField.FailAfter(-1)
	satOK(t, e)
	next := out.value
	if next == nil || !out.owned || len(out.pending) != 0 || !old.Closed || next.References != 1 || next.Closed || next.DerivedModel() == nil {
		t.Fatalf("filter output publication and old release: %+v", out)
	}
	satOK(t, out.Close())
	if !next.Closed || !old.Closed {
		t.Fatal("output owning cleanup retained next model")
	}
	satOK(t, old.Close())
	satOK(t, filter.Close())
}

func TestTacticModelConverterToModelAssignmentUsesNativeIDRecycle(t *testing.T) {
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
	m.BudgetField.FailAfter(1) // new model header; AST ID recycle is native
	e = ModelConverterToModel(m, converter, &out)
	m.BudgetField.FailAfter(-1)
	satOK(t, e)
	next := out.value
	if len(calls) != 1 || calls[0] != "model:unreached:0" || next == nil || !out.owned || len(out.pending) != 0 || !old.Closed || next.References != 1 || next.Closed || next.DerivedModel() == nil {
		t.Fatalf("converter callback and source publication: calls %v, ref %+v", calls, out)
	}
	satOK(t, out.Close())
	if !next.Closed || !old.Closed {
		t.Fatal("output cleanup did not retire acquired new model")
	}
	satOK(t, old.Close())
}
