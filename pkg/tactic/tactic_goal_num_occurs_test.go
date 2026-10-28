package tactic

// Source: src/tactic/goal_num_occurs.cpp; src/tactic/goal_num_occurs.h.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestGoalNumOccursSharesVisitedAcrossFormulas(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	x, e := m.MkConst(astTestSymbol(t, m, "goal-occurs-x"), b)
	astTestMust(t, e)
	f, e := m.MkFuncDecl(astTestSymbol(t, m, "goal-occurs-f"), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	fx, e := m.MkApp(f, []ast.ASTHandle{x})
	astTestMust(t, e)
	g, e := NewGoal(m, false, false, false)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, g.Close()) })
	astTestMust(t, g.Assert(fx))
	astTestMust(t, g.Assert(fx))

	n, e := NewGoalNumOccurs(m, false, false)
	astTestMust(t, e)
	astTestMust(t, n.Count(g))
	if got := n.GetNumOccurrences(fx); got != 2 {
		t.Fatal("repeated formula count", got)
	}
	if got := n.GetNumOccurrences(x); got != 1 {
		t.Fatal("shared formula child must be traversed once", got)
	}
	n.Reset()
	if got := n.GetNumOccurrences(fx); got != 0 {
		t.Fatal("reset retained count", got)
	}

	astTestMust(t, g.Close())
	if e = n.Count(g); e == nil {
		t.Fatal("closed goal accepted")
	}
	if e = (*GoalNumOccurs)(nil).Count(g); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("nil counter", e)
	}
}
