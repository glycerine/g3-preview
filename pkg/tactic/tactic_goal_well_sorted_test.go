package tactic

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestTacticGoalWellSortedStopsAtFirstBadFormula(t *testing.T) {
	m := astTestManager(t)
	g := goalTestNew(t, m, false, false)
	good, e := g.IsWellSorted()
	astTestMust(t, e)
	if !good {
		t.Fatal("empty goal should be well sorted")
	}
	boolSort, e := m.MkBoolSort()
	astTestMust(t, e)
	otherSort, e := m.MkSort(usymbol.Numerical(42), ast.ASTSortInfo{})
	astTestMust(t, e)
	constantDecl, e := m.MkFuncDecl(usymbol.Numerical(43), nil, otherSort, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	constant, e := m.MkApp(constantDecl, nil)
	astTestMust(t, e)
	bad, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolSort}, Names: []usymbol.Symbol{usymbol.Numerical(0)}, Body: constant})
	astTestMust(t, e)
	atom := goalTestAtom(t, m, "well-sorted")
	astTestMust(t, g.Assert(atom))
	astTestMust(t, g.Assert(bad))
	astTestMust(t, g.Assert(bad))
	warnings := diagnostic.BorrowProcessWarningService()
	wasEnabled := warnings.Enabled()
	astTestMust(t, warnings.Enable(true))
	var output bytes.Buffer
	astTestMust(t, warnings.SetWarningStream(&output))
	t.Cleanup(func() {
		astTestMust(t, warnings.SetWarningStream(nil))
		astTestMust(t, warnings.Enable(wasEnabled))
	})
	good, e = g.IsWellSorted()
	astTestMust(t, e)
	if good || output.String() != "WARNING: quantifier's body must be a boolean.\n" {
		t.Fatalf("first bad formula: good=%v warning=%q", good, output.String())
	}
}
