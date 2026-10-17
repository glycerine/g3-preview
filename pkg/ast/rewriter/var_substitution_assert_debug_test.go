//go:build g3debug

package rewriter

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestVarSubstitutionSourceWellSortedAssertionSites(t *testing.T) {
	m := astTestManager(t)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	otherSort, err := m.MkSort(astTestSymbol(t, m, "subst-other"), ast.ASTSortInfo{})
	astTestMust(t, err)
	decl, err := m.MkFuncDecl(astTestSymbol(t, m, "subst-bad-body"), nil, otherSort, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	constant, err := m.MkApp(decl, nil)
	astTestMust(t, err)
	bad, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolSort}, Names: []usymbol.Symbol{astTestSymbol(t, m, "bound")}, Body: constant})
	astTestMust(t, err)
	astTestPin(t, m, bad)
	v, err := m.MkVar(0, boolSort)
	astTestMust(t, err)
	astTestPin(t, m, v)
	truth, err := m.MkTrue()
	astTestMust(t, err)
	out := astTestPin(t, m, v)
	s, err := NewVarSubstitution(m, true)
	astTestMust(t, err)
	defer func() { astTestMust(t, s.Close()) }()
	budget := arena.NewBudget(1 << 20)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	astTestMust(t, err)
	debug, err := diagnostic.NewDebugContext(region, budget)
	astTestMust(t, err)
	previous, err := diagnostic.BindProcessDebugContext(debug)
	astTestMust(t, err)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		astTestMust(t, e)
		astTestMust(t, debug.Close())
		astTestMust(t, region.Close())
		if budget.Used() != 0 {
			t.Fatal("debug owner leak", budget.Used())
		}
	})
	astTestMust(t, s.Substitute(v, []ast.ASTHandle{truth}, out))
	if got, e := m.RefGet(out); e != nil || got != truth {
		t.Fatal("valid substitution", got, e)
	}
	bindings := len(s.reducer.engine.bindings)
	if err := s.Substitute(bad, nil, out); !errors.Is(err, diagnostic.ErrDebugAssertion) {
		t.Fatal("entry source assertion", err)
	}
	if got, e := m.RefGet(out); e != nil || got != truth || len(s.reducer.engine.bindings) != bindings {
		t.Fatal("entry assertion changed reducer/output", got, e, len(s.reducer.engine.bindings), bindings)
	}
	if err := s.assertWellSorted(bad, 34); !errors.Is(err, diagnostic.ErrDebugAssertion) {
		t.Fatal("post-rewrite source assertion site", err)
	}
	astTestMust(t, debug.EnableAssertions(false))
	astTestMust(t, s.assertWellSorted(bad, 27))
	astTestMust(t, s.assertWellSorted(bad, 34))
}
