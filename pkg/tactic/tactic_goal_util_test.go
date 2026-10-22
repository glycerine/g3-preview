// Source: src/tactic/goal_util.cpp:20-33.
package tactic

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestGoalHasTermITE(t *testing.T) {
	m := astTestManager(t)
	condition := goalTestAtom(t, m, "term-ite-condition")
	booleanITE, err := m.MkIte(condition, condition, condition)
	astTestMust(t, err)
	astTestPin(t, m, booleanITE)
	booleanGoal := goalTestNew(t, m, false, false)
	astTestMust(t, booleanGoal.Assert(booleanITE))
	yes, err := HasTermITE(booleanGoal)
	astTestMust(t, err)
	if yes {
		t.Fatal("Boolean ITE classified as a term ITE")
	}

	sort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "TermITEU"), nil)
	astTestMust(t, err)
	astTestPin(t, m, sort)
	x, err := m.MkConst(astTestSymbol(t, m, "term-ite-x"), sort)
	astTestMust(t, err)
	astTestPin(t, m, x)
	y, err := m.MkConst(astTestSymbol(t, m, "term-ite-y"), sort)
	astTestMust(t, err)
	astTestPin(t, m, y)
	termITE, err := m.MkIte(condition, x, y)
	astTestMust(t, err)
	astTestPin(t, m, termITE)
	formula, err := m.MkEq(termITE, x)
	astTestMust(t, err)
	astTestPin(t, m, formula)
	termGoal := goalTestNew(t, m, false, false)
	astTestMust(t, termGoal.Assert(formula))
	yes, err = HasTermITE(termGoal)
	astTestMust(t, err)
	if !yes {
		t.Fatal("term ITE was not found")
	}

	if _, err = HasTermITE(nil); err != ast.ErrASTHandle {
		t.Fatal("nil goal guard", err)
	}
}
