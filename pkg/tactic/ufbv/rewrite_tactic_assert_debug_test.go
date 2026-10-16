//go:build g3debug

package ufbv

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestUFBVRewriterSourceWellSortedAssertions(t *testing.T) {
	m := cacheFixture(t)
	boolSort, e := m.MkBoolSort()
	cacheMust(t, e)
	otherSort, e := m.MkSort(symbol.Numerical(990), ast.ASTSortInfo{})
	cacheMust(t, e)
	decl, e := m.MkFuncDecl(symbol.Numerical(991), nil, otherSort, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	constant, e := m.MkApp(decl, nil)
	cacheMust(t, e)
	bad, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolSort}, Names: []symbol.Symbol{symbol.Numerical(992)}, Body: constant})
	cacheMust(t, e)
	goal, e := tactic.NewGoal(m, false, false, false)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, goal.Close()) })
	cacheMust(t, goal.Assert(bad))
	ctx, globals := ufbvTacticServicesFixture(t, m)
	r, e := NewRewriterTactic(m, ctx, globals)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, r.Close()) })
	budget := arena.NewBudget(1 << 20)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	cacheMust(t, e)
	debug, e := diagnostic.NewDebugContext(region, budget)
	cacheMust(t, e)
	previous, e := diagnostic.BindProcessDebugContext(debug)
	cacheMust(t, e)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		cacheMust(t, e)
		cacheMust(t, debug.Close())
		cacheMust(t, region.Close())
		if budget.Used() != 0 {
			t.Fatal("debug owner leak", budget.Used())
		}
	})
	if result, e := r.Apply(goal); result != nil || !errors.Is(e, diagnostic.ErrDebugAssertion) {
		t.Fatal("entry well-sorted assertion", result, e)
	}
	size, e := goal.Size()
	cacheMust(t, e)
	depth, e := goal.Depth()
	cacheMust(t, e)
	if size != 1 || depth != 0 {
		t.Fatal("entry assertion changed the goal", size, depth)
	}
	if e := assertRewriterGoalWellSorted(goal, 68); !errors.Is(e, diagnostic.ErrDebugAssertion) {
		t.Fatal("post-publication source assertion site", e)
	}
	cacheMust(t, debug.EnableAssertions(false))
	cacheMust(t, assertRewriterGoalWellSorted(goal, 39))
	cacheMust(t, assertRewriterGoalWellSorted(goal, 68))
}
