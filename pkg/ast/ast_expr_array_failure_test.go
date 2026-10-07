package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTExprArrayDeletionFailureConsumesRefAndFinishesOwnedGraph(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	baseline := m.liveNodes
	var args [32]ASTHandle
	for i := range args {
		args[i], e = m.MkVar(uint32(i), b)
		astTestMust(t, e)
	}
	h, e := m.MkFamilyApp(0, ASTOpAnd, nil, args[:])
	astTestMust(t, e)
	var a ASTExprArray
	astTestMust(t, m.ExprArrayPush(&a, h))
	m.BudgetField.FailAfter(0)
	e = m.ExprArrayDel(&a)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("worklist failure expected", e)
	}
	if a != (ASTExprArray{}) {
		t.Fatal("consumed array handle was not cleared", a)
	}
	if m.exprArrays.liveCells != 0 || m.exprArrays.liveBlocks != 0 || m.liveNodes != baseline {
		t.Fatal("immediate graph contains lost source-unsafe deletion continuation", m.exprArrays.liveCells, m.exprArrays.liveBlocks, m.liveNodes, baseline)
	}
	m.BudgetField.FailAfter(-1)
}
