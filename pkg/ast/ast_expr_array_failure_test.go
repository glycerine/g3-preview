package ast

import (
	"testing"
)

func TestASTExprArrayNativeDeletionConsumesRefAndFinishesOwnedGraph(t *testing.T) {
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
	astTestMust(t, e)
	if a != (ASTExprArray{}) {
		t.Fatal("consumed array handle was not cleared", a)
	}
	if m.exprArrays.liveCells != 0 || m.exprArrays.liveBlocks != 0 || m.liveNodes != baseline {
		t.Fatal("native deletion did not release the entire graph", m.exprArrays.liveCells, m.exprArrays.liveBlocks, m.liveNodes, baseline)
	}
	m.BudgetField.FailAfter(-1)
}
