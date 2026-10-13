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
	if !r.closed || incoming.References != 1 {
		t.Fatal("owning cleanup of stopped assignment", r, incoming.References)
	}
	satOK(t, old.Close())
	satOK(t, incoming.DecRef())
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
	if incoming.References != 1 {
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
	if !next.Closed {
		t.Fatal("output owning cleanup retained next model")
	}
	satOK(t, old.Close())
	satOK(t, filter.Close())
}
