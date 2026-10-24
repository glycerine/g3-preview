package core

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestCollectOccurrencesSharedDAGAndGoalRoots(t *testing.T) {
	m, _, _ := simplifyFixture(t)
	a, b, notB := propagateAtoms(t, m)
	boolean, err := m.MkBoolSort()
	simplifyMust(t, err)
	name, err := m.Symbols.Intern("f")
	simplifyMust(t, err)
	f, err := m.MkFuncDecl(name, []ast.ASTHandle{boolean}, boolean, ast.ASTFuncDeclInfo{})
	simplifyMust(t, err)
	fa, err := m.MkApp(f, []ast.ASTHandle{a})
	simplifyMust(t, err)
	simplifyPin(t, m, fa)
	left, err := m.MkOr(fa, b)
	simplifyMust(t, err)
	simplifyPin(t, m, left)
	right, err := m.MkOr(fa, notB)
	simplifyMust(t, err)
	simplifyPin(t, m, right)
	goal, err := tactic.NewGoal(m, false, false, false)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	simplifyMust(t, goal.Assert(left))
	simplifyMust(t, goal.Assert(right))
	variables, err := collectSingleOccurrenceVariables(goal)
	simplifyMust(t, err)
	if len(variables) != 1 || variables[0] != a {
		t.Fatal("a should occur once through the shared f(a), while b repeats", variables)
	}
	// A separate root visits a again, even though the shared compound is
	// traversed only once. A new collector must see the current goal.
	simplifyMust(t, goal.Assert(a))
	variables, err = collectSingleOccurrenceVariables(goal)
	simplifyMust(t, err)
	if len(variables) != 0 {
		t.Fatal("separate root occurrence was ignored", variables)
	}
}

func TestCollectOccurrencesQuantifierBodySkipsPatterns(t *testing.T) {
	m, _, _ := simplifyFixture(t)
	a, b, _ := propagateAtoms(t, m)
	boolean, err := m.MkBoolSort()
	simplifyMust(t, err)
	x, err := m.MkVar(0, boolean)
	simplifyMust(t, err)
	simplifyPin(t, m, x)
	body, err := m.MkOr(a, x)
	simplifyMust(t, err)
	simplifyPin(t, m, body)
	name, err := m.Symbols.Intern("trigger")
	simplifyMust(t, err)
	f, err := m.MkFuncDecl(name, []ast.ASTHandle{boolean, boolean}, boolean, ast.ASTFuncDeclInfo{})
	simplifyMust(t, err)
	trigger, err := m.MkApp(f, []ast.ASTHandle{x, b})
	simplifyMust(t, err)
	simplifyPin(t, m, trigger)
	pattern, err := m.MkPattern([]ast.ASTHandle{trigger})
	simplifyMust(t, err)
	simplifyPin(t, m, pattern)
	for _, patterns := range []bool{false, true} {
		var positive, negative []ast.ASTHandle
		if patterns {
			positive = []ast.ASTHandle{pattern}
		} else {
			negative = []ast.ASTHandle{trigger}
		}
		q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []usymbol.Symbol{usymbol.Numerical(0)}, Body: body, Weight: 1, Patterns: positive, NoPatterns: negative})
		simplifyMust(t, err)
		simplifyPin(t, m, q)
		goal, err := tactic.NewGoal(m, false, false, false)
		simplifyMust(t, err)
		t.Cleanup(func() { simplifyMust(t, goal.Close()) })
		simplifyMust(t, goal.Assert(q))
		simplifyMust(t, goal.Assert(b))
		variables, err := collectSingleOccurrenceVariables(goal)
		simplifyMust(t, err)
		if len(variables) != 2 || variables[0] != a || variables[1] != b {
			t.Fatal("patterns must not make b repeated; free a in the body must be visited", patterns, variables)
		}
	}
}

func TestCollectOccurrencesLeafBodiesAndRepeatedEdges(t *testing.T) {
	m, _, _ := simplifyFixture(t)
	a, b, _ := propagateAtoms(t, m)
	boolean, err := m.MkBoolSort()
	simplifyMust(t, err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []usymbol.Symbol{usymbol.Numerical(0)}, Body: a, Weight: 1})
	simplifyMust(t, err)
	simplifyPin(t, m, q)
	eq, err := m.MkEq(b, b)
	simplifyMust(t, err)
	simplifyPin(t, m, eq)
	goal, err := tactic.NewGoal(m, false, false, false)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	simplifyMust(t, goal.Assert(q))
	simplifyMust(t, goal.Assert(eq))
	variables, err := collectSingleOccurrenceVariables(goal)
	simplifyMust(t, err)
	if len(variables) != 1 || variables[0] != a {
		t.Fatal("leaf quantifier body is single, repeated argument edges are not", variables)
	}
	if _, err = collectSingleOccurrenceVariables(nil); err == nil {
		t.Fatal("nil goal accepted")
	}
}
