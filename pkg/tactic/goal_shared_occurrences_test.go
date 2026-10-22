// Source: src/tactic/goal_shared_occs.cpp; src/tactic/goal_shared_occs.h.
package tactic

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestGoalSharedOccurrencesRetainsVisitedAcrossFormulas(t *testing.T) {
	m := astTestManager(t)
	p := goalTestAtom(t, m, "shared-p")
	q := goalTestAtom(t, m, "shared-q")
	r := goalTestAtom(t, m, "shared-r")
	s := goalTestAtom(t, m, "shared-s")
	sub := goalTestApp(t, m, ast.ASTOpOr, p, q)
	left := goalTestApp(t, m, ast.ASTOpOr, sub, r)
	right := goalTestApp(t, m, ast.ASTOpOr, sub, s)
	g := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(left))
	astTestMust(t, g.Assert(right))
	occs, err := NewDefaultGoalSharedOccurrences(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, occs.Close()) })
	astTestMust(t, occs.Collect(g))
	shared, err := occs.IsShared(sub)
	astTestMust(t, err)
	if !shared || occs.NumShared() != 1 {
		t.Fatal("cross-formula shared occurrence", shared, occs.NumShared())
	}
	astTestMust(t, occs.Reset())
	shared, err = occs.IsShared(sub)
	astTestMust(t, err)
	if shared || occs.NumShared() != 0 {
		t.Fatal("reset shared occurrences", shared, occs.NumShared())
	}
}

func TestGoalSharedOccurrencesAtomicPolicy(t *testing.T) {
	m := astTestManager(t)
	p := goalTestAtom(t, m, "shared-atomic")
	g := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(p))
	astTestMust(t, g.Assert(p))
	occs, err := NewGoalSharedOccurrences(m, true, true, false)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, occs.Close()) })
	astTestMust(t, occs.Collect(g))
	shared, err := occs.IsShared(p)
	astTestMust(t, err)
	if !shared {
		t.Fatal("tracked atomic occurrence was not shared")
	}
}
