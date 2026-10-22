// Source: src/tactic/goal.cpp:635-655; src/tactic/goal.h:145.
package tactic

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestTacticGoalTranslatePreservesConfigurationAndDependencies(t *testing.T) {
	from := astTestManager(t)
	to, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: from.Symbols, Numerals: from.NumeralService(), Budget: from.Budget()})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, to.Close()) })
	p := goalTestAtom(t, from, "goal-translate-p")
	dependency, err := from.DependencyLeaf(p)
	astTestMust(t, err)
	astTestMust(t, from.DependencyInc(dependency))
	t.Cleanup(func() { astTestMust(t, from.DependencyDec(dependency)) })
	g := goalTestNew(t, from, false, true)
	astTestMust(t, g.AssertWithProof(p, ast.ASTHandle{}, dependency))
	astTestMust(t, g.SetDepth(9))
	astTestMust(t, g.SetPrecision(GoalUnder))
	translation, err := ast.NewASTTranslation(from, to, true)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, translation.Close()) })
	translated, err := g.Translate(translation)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, translated.Close()) })
	if translated.manager != to || translated.depth != 9 || translated.precision != GoalUnder || !translated.modelsEnabled || translated.proofsEnabled || !translated.coreEnabled {
		t.Fatal("translated goal configuration")
	}
	want, err := translation.Translate(p)
	astTestMust(t, err)
	goalTestForms(t, translated, want)
	dep, err := translated.Dependency(0)
	astTestMust(t, err)
	leaves, err := to.DependencyLinearize(dep, nil)
	astTestMust(t, err)
	if len(leaves) != 1 || leaves[0] != want {
		t.Fatal("translated dependency", leaves, want)
	}
}
