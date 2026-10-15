//go:build g3debug

package aig

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestAIGPassSourceEntryWellSortedAssertionPrecedesManager(t *testing.T) {
	m, budget := aigFixture(t)
	boolSort, err := m.MkBoolSort()
	aigMust(t, err)
	otherSort, err := m.MkSort(usymbol.Numerical(42), ast.ASTSortInfo{})
	aigMust(t, err)
	decl, err := m.MkFuncDecl(usymbol.Numerical(43), nil, otherSort, ast.ASTFuncDeclInfo{})
	aigMust(t, err)
	constant, err := m.MkApp(decl, nil)
	aigMust(t, err)
	bad, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{boolSort},
		Names: []usymbol.Symbol{usymbol.Numerical(0)}, Body: constant,
	})
	aigMust(t, err)
	goal, err := tactic.NewGoal(m, false, false, false)
	aigMust(t, err)
	defer func() { aigMust(t, goal.Close()) }()
	aigMust(t, goal.Assert(bad))

	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	aigMust(t, err)
	debug, err := diagnostic.NewDebugContext(region, budget)
	aigMust(t, err)
	previous, err := diagnostic.BindProcessDebugContext(debug)
	aigMust(t, err)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		aigMust(t, e)
		aigMust(t, debug.Close())
		aigMust(t, region.Close())
	})
	warnings := diagnostic.BorrowProcessWarningService()
	wasEnabled := warnings.Enabled()
	aigMust(t, warnings.Enable(false))
	t.Cleanup(func() { aigMust(t, warnings.Enable(wasEnabled)) })

	if out, e := NewPass().Apply(goal); out != nil || !errors.Is(e, diagnostic.ErrDebugAssertion) {
		t.Fatalf("entry assertion: result=%v err=%v", out, e)
	}
	if size, e := goal.Size(); e != nil || size != 1 {
		t.Fatalf("entry assertion changed goal: size=%d err=%v", size, e)
	}
	if depth, e := goal.Depth(); e != nil || depth != 0 {
		t.Fatalf("entry assertion changed depth: depth=%d err=%v", depth, e)
	}
}
