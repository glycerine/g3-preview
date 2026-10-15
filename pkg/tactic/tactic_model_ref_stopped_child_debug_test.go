//go:build g3debug

package tactic

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
)

// The model's last reference is consumed before its derived destructor can
// finish releasing a universe value. The ref assignment must retain the
// incoming model separately and never decrement the old model twice.
func TestModelRefStoppedDerivedChildRelease(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.EnableDebugRefCount())
	old, e := model.NewModel(m)
	satOK(t, e)
	sort, e := m.MkUninterpretedSort(astTestSymbol(t, m, "stopped-model-sort"), nil)
	satOK(t, e)
	value, e := m.GetSomeValue(sort)
	satOK(t, e)
	satOK(t, old.RegisterUniverse(sort, []ast.ASTHandle{value}))
	id, e := m.GetASTID(value)
	satOK(t, e)
	incoming, e := model.NewModel(m)
	satOK(t, e)
	incomingSort, e := m.MkUninterpretedSort(astTestSymbol(t, m, "pending-incoming-sort"), nil)
	satOK(t, e)
	incomingValue, e := m.GetSomeValue(incomingSort)
	satOK(t, e)
	satOK(t, incoming.RegisterUniverse(incomingSort, []ast.ASTHandle{incomingValue}))
	var ref ModelRef
	satOK(t, ref.Set(&old.CoreField))
	// The checked AST deletion guard stops after the model has consumed its
	// reference to value, at the real child-release boundary.
	m.DebugFreeIndices[id] = struct{}{}
	e = ref.Set(&incoming.CoreField)
	if !errors.Is(e, ast.ErrASTDomain) || old.CoreField.References != 0 || ref.owned ||
		ref.cleanup != &old.CoreField || len(ref.pending) != 1 || ref.pending[0] != &incoming.CoreField || incoming.CoreField.References != 1 {
		t.Fatalf("stopped model assignment: error=%v old=%d ref=%+v incoming=%d", e, old.CoreField.References, ref, incoming.CoreField.References)
	}
	delete(m.DebugFreeIndices, id)
	satOK(t, ref.Close())
	if !ref.closed || !old.Closed || !old.CoreField.Closed || !incoming.Closed || !incoming.CoreField.Closed || incoming.CoreField.References != 0 {
		t.Fatalf("owning completion replayed a consumed reference: ref=%+v old=%v incoming=%v/%d", ref, old.Closed, incoming.Closed, incoming.CoreField.References)
	}
}

func TestModelRefCloseAfterStoppedDerivedChildRelease(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.EnableDebugRefCount())
	md, e := model.NewModel(m)
	satOK(t, e)
	sort, e := m.MkUninterpretedSort(astTestSymbol(t, m, "stopped-close-sort"), nil)
	satOK(t, e)
	value, e := m.GetSomeValue(sort)
	satOK(t, e)
	satOK(t, md.RegisterUniverse(sort, []ast.ASTHandle{value}))
	id, e := m.GetASTID(value)
	satOK(t, e)
	var ref ModelRef
	satOK(t, ref.Set(&md.CoreField))
	m.DebugFreeIndices[id] = struct{}{}
	e = ref.Close()
	if !errors.Is(e, ast.ErrASTDomain) || md.CoreField.References != 0 || ref.cleanup != &md.CoreField || ref.owned || ref.closed {
		t.Fatalf("stopped final reference: error=%v ref=%+v count=%d", e, ref, md.CoreField.References)
	}
	delete(m.DebugFreeIndices, id)
	satOK(t, ref.Close())
	if !ref.closed || !md.Closed || !md.CoreField.Closed || md.CoreField.References != 0 {
		t.Fatalf("final cleanup replayed consumed decrement: ref=%+v model=%v count=%d", ref, md.Closed, md.CoreField.References)
	}
}
