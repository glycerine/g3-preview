package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTScopedExprSubstitutionActualTrailAndBorrowedDestructor(t *testing.T) {
	m := astTestManager(t)
	key := astDependencyExpr(t, m, 10100)
	a := astDependencyExpr(t, m, 10101)
	b := astDependencyExpr(t, m, 10102)
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	scope, e := m.NewScopedExprSubstitution(s)
	astTestMust(t, e)
	astTestMust(t, m.ScopedExprSubstitutionPush(scope))
	astTestMust(t, m.ScopedExprSubstitutionInsert(scope, key, a, ASTHandle{}, ASTDependency{}))
	astTestMust(t, m.ScopedExprSubstitutionInsert(scope, key, b, ASTHandle{}, ASTDependency{}))
	def, e := m.ScopedExprSubstitutionFindValue(scope, key)
	astTestMust(t, e)
	if def != a {
		t.Fatal(def)
	}
	astTestMust(t, m.ScopedExprSubstitutionPop(scope, 0))
	level, e := m.ScopedExprSubstitutionScopeLevel(scope)
	astTestMust(t, e)
	if level != 1 {
		t.Fatal(level)
	}
	astTestMust(t, m.ScopedExprSubstitutionPop(scope, 1))
	def, e = m.ScopedExprSubstitutionFindValue(scope, key)
	astTestMust(t, e)
	if def != key {
		t.Fatal(def)
	}
	astTestMust(t, m.ScopedExprSubstitutionInsert(scope, key, b, ASTHandle{}, ASTDependency{}))
	astTestMust(t, m.DelScopedExprSubstitution(&scope))
	found, e := m.ExprSubstitutionContains(s, key)
	astTestMust(t, e)
	if !found {
		t.Fatal("scoped destructor must not reset borrowed map")
	}
	astTestMust(t, m.DelExprSubstitution(&s))
}
func TestASTScopedExprSubstitutionCleanupKeepsTrailAndNestedScopes(t *testing.T) {
	m := astTestManager(t)
	key := astDependencyExpr(t, m, 11400)
	other := astDependencyExpr(t, m, 11401)
	value := astDependencyExpr(t, m, 11402)
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	scope, e := m.NewScopedExprSubstitution(s)
	astTestMust(t, e)
	astTestMust(t, m.ScopedExprSubstitutionPush(scope))
	astTestMust(t, m.ScopedExprSubstitutionInsert(scope, key, value, ASTHandle{}, ASTDependency{}))
	astTestMust(t, m.ScopedExprSubstitutionPush(scope))
	astTestMust(t, m.ScopedExprSubstitutionInsert(scope, other, value, ASTHandle{}, ASTDependency{}))
	astTestMust(t, m.ScopedExprSubstitutionPop(scope, 1))
	found, e := m.ScopedExprSubstitutionContains(scope, key)
	astTestMust(t, e)
	if !found {
		t.Fatal("outer entry")
	}
	astTestMust(t, m.ScopedExprSubstitutionCleanup(scope))
	a, e := m.scopedSubstState(scope)
	astTestMust(t, e)
	trail, e := m.vectorState(a.trail)
	astTestMust(t, e)
	if a.size != 1 || trail.size != 1 {
		t.Fatal("cleanup must not reset limits/trail", a, trail)
	}
	empty, e := m.ScopedExprSubstitutionEmpty(scope)
	astTestMust(t, e)
	if !empty {
		t.Fatal("cleanup forwards map")
	}
	var def, pr ASTHandle
	found, e = m.ScopedExprSubstitutionFindProof(scope, key, &def, &pr)
	astTestMust(t, e)
	if found {
		t.Fatal("miss")
	}
	var dep ASTDependency
	found, e = m.ScopedExprSubstitutionFindDependency(scope, key, &def, &pr, &dep)
	astTestMust(t, e)
	if found {
		t.Fatal("miss")
	}
	astTestMust(t, m.ScopedExprSubstitutionPop(scope, 1))
	astTestMust(t, m.ScopedExprSubstitutionReset(scope))
	astTestMust(t, m.DelScopedExprSubstitution(&scope))
	astTestMust(t, m.DelExprSubstitution(&s))
}
func TestASTScopedExprSubstitutionNullValueIsPresentAndReturned(t *testing.T) {
	m := astTestManager(t)
	key := astDependencyExpr(t, m, 11800)
	value := astDependencyExpr(t, m, 11801)
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	scope, e := m.NewScopedExprSubstitution(s)
	astTestMust(t, e)
	astTestMust(t, m.ScopedExprSubstitutionInsert(scope, key, ASTHandle{}, ASTHandle{}, ASTDependency{}))
	astTestMust(t, m.ScopedExprSubstitutionInsert(scope, key, value, ASTHandle{}, ASTDependency{}))
	def, e := m.ScopedExprSubstitutionFindValue(scope, key)
	astTestMust(t, e)
	if !def.IsNull() {
		t.Fatal("present NULL definition must not fall back", def)
	}
	astTestMust(t, m.ScopedExprSubstitutionReset(scope))
	astTestMust(t, m.DelScopedExprSubstitution(&scope))
	astTestMust(t, m.DelExprSubstitution(&s))
}
func TestASTScopedExprSubstitutionRefusedTrailCleanupResumesOnlyUnconsumed(t *testing.T) {
	m := astTestManager(t)
	first := astDependencyExpr(t, m, 15100)
	second := astDependencyExpr(t, m, 15101)
	value := astDependencyExpr(t, m, 15102)
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	scope, e := m.NewScopedExprSubstitution(s)
	astTestMust(t, e)
	astTestMust(t, m.ScopedExprSubstitutionInsert(scope, first, value, ASTHandle{}, ASTDependency{}))
	astTestMust(t, m.ScopedExprSubstitutionInsert(scope, second, value, ASTHandle{}, ASTDependency{}))
	astTestMust(t, m.ExprSubstitutionReset(s)) // Legal external map reset preserves trail roots.
	m.nodeAt(second.slot).rc = 0
	e = m.ScopedExprSubstitutionReset(scope)
	if !errors.Is(e, ErrASTDomain) {
		t.Fatal(e)
	}
	a, e := m.scopedSubstState(scope)
	astTestMust(t, e)
	if a.operation != ^uint32(0) || a.index != 1 {
		t.Fatal("only consumed trail prefix advances", a)
	}
	rc, e := m.GetRefCount(first)
	astTestMust(t, e)
	if rc != 1 {
		t.Fatal(rc)
	}
	if e = m.ScopedExprSubstitutionPush(scope); !errors.Is(e, ErrASTPending) {
		t.Fatal(e)
	}
	m.nodeAt(second.slot).rc = 2
	astTestMust(t, m.ScopedExprSubstitutionReset(scope))
	rc, e = m.GetRefCount(first)
	astTestMust(t, e)
	if rc != 1 {
		t.Fatal("retry DEC duplicated prefix", rc)
	}
	astTestMust(t, m.DelScopedExprSubstitution(&scope))
	astTestMust(t, m.DelExprSubstitution(&s))
}
func TestASTScopedExprSubstitutionRetainedLimitsReleaseResizePaths(t *testing.T) {
	m := astTestManager(t)
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	scope, e := m.NewScopedExprSubstitution(s)
	astTestMust(t, e)
	value := astDependencyExpr(t, m, 17000)
	for i := uint32(0); i < 2; i++ {
		key := astDependencyExpr(t, m, 17001+i)
		astTestMust(t, m.ScopedExprSubstitutionInsert(scope, key, value, ASTHandle{}, ASTDependency{}))
	}
	astTestMust(t, m.ScopedExprSubstitutionPush(scope))
	key := astDependencyExpr(t, m, 17003)
	astTestMust(t, m.ScopedExprSubstitutionInsert(scope, key, value, ASTHandle{}, ASTDependency{}))
	astTestMust(t, m.ScopedExprSubstitutionPush(scope))
	astTestMust(t, m.ScopedExprSubstitutionReset(scope))
	e = m.ScopedExprSubstitutionPop(scope, ^uint32(0))
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal(e)
		}
	} else {
		astTestMust(t, e)
		a, e := m.scopedSubstState(scope)
		astTestMust(t, e)
		trail, e := m.vectorState(a.trail)
		astTestMust(t, e)
		if trail.size != 3 || a.size != 1 {
			t.Fatal(trail, a)
		}
		for i := uint32(0); i < 3; i++ {
			h, e := m.RefVectorGet(a.trail, i)
			astTestMust(t, e)
			if !h.IsNull() {
				t.Fatal("source resize constructs NULL, not retained pointer", i, h)
			}
		}
		limit, e := m.scopedSubstLimit(a, 0)
		astTestMust(t, e)
		if limit != 0 {
			t.Fatal("source unsigned resize constructs zero", limit)
		}
	}
	astTestMust(t, m.DelScopedExprSubstitution(&scope))
	astTestMust(t, m.DelExprSubstitution(&s))
}
func TestASTScopedExprSubstitutionDestructorAfterConsumedPopError(t *testing.T) {
	m := astTestManager(t)
	s, e := m.NewExprSubstitutionCores(true)
	astTestMust(t, e)
	scope, e := m.NewScopedExprSubstitution(s)
	astTestMust(t, e)
	key := astDependencyExpr(t, m, 17100)
	value := astDependencyExpr(t, m, 17101)
	dep, e := m.DependencyLeaf(value)
	astTestMust(t, e)
	astTestMust(t, m.ScopedExprSubstitutionPush(scope))
	astTestMust(t, m.ScopedExprSubstitutionInsert(scope, key, value, ASTHandle{}, dep))
	m.BudgetField.FailAfter(0)
	e = m.ScopedExprSubstitutionPop(scope, 1)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	astTestMust(t, m.DelScopedExprSubstitution(&scope))
	if scope != (ASTScopedExprSubstitution{}) {
		t.Fatal(scope)
	}
	rc, e := m.GetRefCount(key)
	astTestMust(t, e)
	if rc != 1 {
		t.Fatal("release retained trail after caught error", rc)
	}
	astTestMust(t, m.DelExprSubstitution(&s))
}
func TestASTScopedExprSubstitutionDestructorInterruptedTailThenPrefix(t *testing.T) {
	limit := uint32((^uint32(0) - 12) / 8)
	if limit >= 1<<29 {
		t.Fatal("actual trail capacity must fit stage payload")
	}
	for _, target := range []uint32{0, 1, limit} {
		word, e := astScopedExprSubstDestructionWord(target)
		astTestMust(t, e)
		if word>>30 != 3 || word&((1<<30)-1) != target || word == ^uint32(0) {
			t.Fatal(word, target)
		}
	}
	if _, e := astScopedExprSubstDestructionWord(1 << 29); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	m := astTestManager(t)
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	scope, e := m.NewScopedExprSubstitution(s)
	astTestMust(t, e)
	first := astDependencyExpr(t, m, 18000)
	second := astDependencyExpr(t, m, 18001)
	third := astDependencyExpr(t, m, 18002)
	value := astDependencyExpr(t, m, 18003)
	astTestMust(t, m.ScopedExprSubstitutionInsert(scope, first, value, ASTHandle{}, ASTDependency{}))
	astTestMust(t, m.ScopedExprSubstitutionPush(scope))
	astTestMust(t, m.ScopedExprSubstitutionInsert(scope, second, value, ASTHandle{}, ASTDependency{}))
	astTestMust(t, m.ScopedExprSubstitutionInsert(scope, third, value, ASTHandle{}, ASTDependency{}))
	// Actual external map erases leave real trail owners. Refused counter is
	// seeded before the API call; every callback/progress action is production.
	astTestMust(t, m.ExprSubstitutionErase(s, second))
	astTestMust(t, m.ExprSubstitutionErase(s, third))
	m.nodeAt(third.slot).rc = 0
	e = m.ScopedExprSubstitutionPop(scope, 1)
	if !errors.Is(e, ErrASTDomain) {
		t.Fatal(e)
	}
	a, e := m.scopedSubstState(scope)
	astTestMust(t, e)
	if a.operation>>30 != 2 || a.index != 2 {
		t.Fatal(a)
	}
	e = m.DelScopedExprSubstitution(&scope)
	if !errors.Is(e, ErrASTDomain) {
		t.Fatal(e)
	}
	a, e = m.scopedSubstState(scope)
	astTestMust(t, e)
	if a.operation>>30 != 3 || a.operation&((1<<30)-1) != 1 || a.index != 2 || a.capacity != 0 {
		t.Fatal("limits destroyed before refused tail continuation", a)
	}
	m.nodeAt(third.slot).rc = 2
	astTestMust(t, m.DelScopedExprSubstitution(&scope))
	for _, key := range []ASTHandle{second, third} {
		rc, e := m.GetRefCount(key)
		astTestMust(t, e)
		if rc != 1 {
			t.Fatal("consumed tail callback replay", key, rc)
		}
	}
	found, e := m.ExprSubstitutionContains(s, first)
	astTestMust(t, e)
	if !found {
		t.Fatal("borrowed outer map must survive")
	}
	astTestMust(t, m.DelExprSubstitution(&s))
	rc, e := m.GetRefCount(first)
	astTestMust(t, e)
	if rc != 1 {
		t.Fatal(rc)
	}
}
