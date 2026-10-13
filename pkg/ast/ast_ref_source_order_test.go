package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTRefAssignmentKeepsOldPointerOnConsumedDeletionStop(t *testing.T) {
	for _, viaRef := range []bool{false, true} {
		name := "pointer"
		if viaRef {
			name = "reference"
		}
		t.Run(name, func(t *testing.T) {
			m := astTestManager(t)
			sort, e := m.MkBoolSort()
			astTestMust(t, e)
			old, e := m.MkVar(11700, sort)
			astTestMust(t, e)
			incoming, e := m.MkVar(11701, sort)
			astTestMust(t, e)
			dst, e := m.NewRef(old)
			astTestMust(t, e)
			oldNode, e := m.internal(old)
			astTestMust(t, e)
			var src ASTRef
			if viaRef {
				src, e = m.NewRef(incoming)
			} else {
				e = m.IncRef(incoming)
			}
			astTestMust(t, e)
			m.ExprIDs.Cleanup(m.ExprIDs.Range())
			m.BudgetField.FailAfter(0)
			if viaRef {
				e = m.AssignRef(dst, src)
			} else {
				e = m.SetRef(dst, incoming)
			}
			m.BudgetField.FailAfter(-1)
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatalf("old decrement refusal: %v", e)
			}
			got, e := m.RefGet(dst)
			astTestMust(t, e)
			if got != old {
				t.Fatal("source assignment published incoming after stopped DEC", got, old)
			}
			record, e := m.reference(dst)
			astTestMust(t, e)
			if record.node != oldNode || record.owned || len(record.pending) != 1 {
				t.Fatalf("owning receipt after stopped assignment: %+v", record)
			}
			if rc, e := m.GetRefCount(incoming); e != nil || rc != 2 {
				t.Fatalf("incoming acquired reference = %d, %v", rc, e)
			}
			astTestMust(t, m.DelRef(&dst))
			if dst != (ASTRef{}) {
				t.Fatal("owning close retained reference record")
			}
			if rc, e := m.GetRefCount(incoming); e != nil || rc != 1 {
				t.Fatalf("incoming reference after owning close = %d, %v", rc, e)
			}
			if viaRef {
				astTestMust(t, m.DelRef(&src))
			} else {
				astTestMust(t, m.DecRef(incoming))
			}
		})
	}
}

func TestASTRefResetKeepsOldPointerOnConsumedDeletionStop(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	old, e := m.MkVar(11710, sort)
	astTestMust(t, e)
	ref, e := m.NewRef(old)
	astTestMust(t, e)
	m.ExprIDs.Cleanup(m.ExprIDs.Range())
	m.BudgetField.FailAfter(0)
	e = m.ResetRef(ref)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatalf("reset decrement refusal: %v", e)
	}
	got, e := m.RefGet(ref)
	astTestMust(t, e)
	if got != old {
		t.Fatal("source reset cleared pointer after stopped DEC", got, old)
	}
	record, e := m.reference(ref)
	astTestMust(t, e)
	if record.owned {
		t.Fatal("consumed old reference remained owned")
	}
	astTestMust(t, m.DelRef(&ref))
}
