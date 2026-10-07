// Licensed under the MIT license.
package ast

import (
	"testing"
)

func TestASTDeletionActualIDRecycleLIFOAndEmergencyOrder(t *testing.T) {
	for _, fail := range []bool{false, true} {
		m := astTestManager(t)
		b, e := m.MkBoolSort()
		astTestMust(t, e)
		var args [32]ASTHandle
		var ids [32]uint32
		for i := range args {
			args[i], e = m.MkVar(uint32(i+1000), b)
			astTestMust(t, e)
			ids[i], e = m.GetASTID(args[i])
			astTestMust(t, e)
		}
		h, e := m.MkFamilyApp(0, ASTOpAnd, nil, args[:])
		astTestMust(t, e)
		id, e := m.GetASTID(h)
		astTestMust(t, e)
		ref, e := m.NewRef(h)
		astTestMust(t, e)
		before := len(m.ExprIDs.Free)
		if fail {
			m.BudgetField.FailAfter(0)
		}
		e = m.DelRef(&ref)
		if fail {
			if e == nil {
				t.Fatal("emergency path not exercised")
			}
		} else {
			astTestMust(t, e)
		}
		m.BudgetField.FailAfter(-1)
		got := m.ExprIDs.Free[before:]
		if len(got) != 33 || got[0] != id {
			t.Fatal("source parent recycled before children", got, id)
		}
		for i := 0; i < 32; i++ {
			if got[i+1] != ids[31-i] {
				t.Fatal("ascending push must produce real LIFO child destruction", fail, i, got[i+1], ids[31-i])
			}
		}
		if ref != (ASTRef{}) || m.pendingDeletionCurrent != (astNodeRef{}) || m.pendingDeletionHead != (astNodeRef{}) {
			t.Fatal("consumed reference/continuation not cleared")
		}
	}
}
func TestASTDeletionInlineWorklistWarmZeroAllocations(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	var refs [101]ASTRef
	for i := range refs {
		var args [6]ASTHandle
		for j := range args {
			args[j], e = m.MkVar(uint32(2000+i*6+j), b)
			astTestMust(t, e)
		}
		h, e := m.MkFamilyApp(0, ASTOpAnd, nil, args[:])
		astTestMust(t, e)
		refs[i], e = m.NewRef(h)
		astTestMust(t, e)
	}
	index := 0
	var failure error
	m.BudgetField.FailAfter(0)
	allocations := testing.AllocsPerRun(100, func() { failure = m.DelRef(&refs[index]); index++ })
	m.BudgetField.FailAfter(-1)
	if failure != nil || allocations != 0 || index != 101 {
		t.Fatal("inline16 worklist must not escape/reserve hidden heap", failure, allocations, index)
	}
}
