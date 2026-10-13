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
