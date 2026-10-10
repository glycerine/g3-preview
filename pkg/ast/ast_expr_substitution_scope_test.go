package ast

import (
	"errors"
	"testing"
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
	if len(a.limits) != 1 || len(a.trail) != 1 {
		t.Fatal("cleanup must not reset limits/trail", a)
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
	if a.operation != astScopedSubstDestroyPrefix || a.index != 1 {
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
func TestASTScopedExprSubstitutionNativeLimitsRejectInvalidPop(t *testing.T) {
	m := astTestManager(t)
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	scope, e := m.NewScopedExprSubstitution(s)
	astTestMust(t, e)
	key := astDependencyExpr(t, m, 17001)
	value := astDependencyExpr(t, m, 17002)
	astTestMust(t, m.ScopedExprSubstitutionPush(scope))
	astTestMust(t, m.ScopedExprSubstitutionInsert(scope, key, value, ASTHandle{}, ASTDependency{}))
	for _, n := range []uint32{2, ^uint32(0)} {
		if e = m.ScopedExprSubstitutionPop(scope, n); !errors.Is(e, ErrASTDomain) {
			t.Fatal(n, e)
		}
		level, e := m.ScopedExprSubstitutionScopeLevel(scope)
		astTestMust(t, e)
		rc, e := m.GetRefCount(key)
		astTestMust(t, e)
		if level != 1 || rc != 3 {
			t.Fatal("invalid pop changed scopes or references", level, rc)
		}
	}
	astTestMust(t, m.ScopedExprSubstitutionPop(scope, 1))
	astTestMust(t, m.ScopedExprSubstitutionReset(scope))
	if e = m.ScopedExprSubstitutionPop(scope, ^uint32(0)); !errors.Is(e, ErrASTDomain) {
		t.Fatal(e)
	}
	astTestMust(t, m.DelScopedExprSubstitution(&scope))
	astTestMust(t, m.DelExprSubstitution(&s))
}
func TestASTScopedExprSubstitutionDestructorAfterNativeDependencyPop(t *testing.T) {
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
	astTestMust(t, e)
	astTestMust(t, m.DelScopedExprSubstitution(&scope))
	if scope != (ASTScopedExprSubstitution{}) {
		t.Fatal(scope)
	}
	rc, e := m.GetRefCount(key)
	astTestMust(t, e)
	if rc != 1 {
		t.Fatal("release trail after native dependency cleanup", rc)
	}
	astTestMust(t, m.DelExprSubstitution(&s))
}
func TestASTScopedExprSubstitutionDestructorInterruptedTailThenPrefix(t *testing.T) {
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
	if a.operation != astScopedSubstRelease || a.index != 2 {
		t.Fatal(a)
	}
	e = m.DelScopedExprSubstitution(&scope)
	if !errors.Is(e, ErrASTDomain) {
		t.Fatal(e)
	}
	a, e = m.scopedSubstState(scope)
	astTestMust(t, e)
	if a.operation != astScopedSubstDestroyTail || a.target != 1 || a.index != 2 || len(a.limits) != 0 {
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

func TestASTScopedExprSubstitutionNativeGrowthIndependentModel(t *testing.T) {
	m := astTestManager(t)
	keys := markTestNodes(t, m, 257)
	for _, key := range keys {
		astTestMust(t, m.IncRef(key))
		defer m.DecRef(key)
	}
	value := astDependencyExpr(t, m, 19000)
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	scope, e := m.NewScopedExprSubstitution(s)
	astTestMust(t, e)
	// The oracle records the source algorithm using ordinary integer identities.
	present := make(map[int]bool)
	var trail, limits []int
	push := func() {
		astTestMust(t, m.ScopedExprSubstitutionPush(scope))
		limits = append(limits, len(trail))
	}
	insert := func(i int) {
		astTestMust(t, m.ScopedExprSubstitutionInsert(scope, keys[i], value, ASTHandle{}, ASTDependency{}))
		if !present[i] {
			present[i] = true
			trail = append(trail, i)
		}
	}
	check := func() {
		counts := make([]uint32, len(keys))
		for _, i := range trail {
			counts[i]++
		}
		level, e := m.ScopedExprSubstitutionScopeLevel(scope)
		astTestMust(t, e)
		if int(level) != len(limits) {
			t.Fatal("scope level", level, len(limits))
		}
		for i, key := range keys {
			got, e := m.ScopedExprSubstitutionFindValue(scope, key)
			astTestMust(t, e)
			want, rcWant := key, 1+counts[i]
			if present[i] {
				want = value
				rcWant++
			}
			rc, e := m.GetRefCount(key)
			astTestMust(t, e)
			if got != want || rc != rcWant {
				t.Fatal("oracle", i, got, want, rc, rcWant)
			}
		}
		rc, e := m.GetRefCount(value)
		astTestMust(t, e)
		if rc != uint32(1+len(present)) {
			t.Fatal("value owners", rc, len(present))
		}
	}
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	for i := range keys {
		push()
		insert(i)
	}
	check()
	seed := uint32(0x193ac5)
	for step := 0; step < 2000; step++ {
		seed = seed*1664525 + 1013904223
		switch seed % 11 {
		case 0, 1:
			push()
		case 2, 3:
			n := 0
			if len(limits) != 0 {
				n = 1 + int((seed>>8)%uint32(len(limits)))
			}
			astTestMust(t, m.ScopedExprSubstitutionPop(scope, uint32(n)))
			if n != 0 {
				level := len(limits) - n
				target := limits[level]
				for _, i := range trail[target:] {
					delete(present, i)
				}
				trail, limits = trail[:target], limits[:level]
			}
		case 4:
			astTestMust(t, m.ScopedExprSubstitutionCleanup(scope))
			clear(present)
		case 5:
			astTestMust(t, m.ScopedExprSubstitutionReset(scope))
			clear(present)
			trail, limits = nil, nil
		default:
			insert(int((seed >> 8) % uint32(len(keys))))
		}
		if step%17 == 0 {
			check()
		}
	}
	check()
	astTestMust(t, m.ScopedExprSubstitutionReset(scope))
	clear(present)
	trail, limits = nil, nil
	check()
	m.BudgetField.FailAfter(-1)
	if m.BudgetField.Used() != before {
		t.Fatal("native growth changed charged storage", m.BudgetField.Used(), before)
	}
	astTestMust(t, m.DelScopedExprSubstitution(&scope))
	astTestMust(t, m.DelExprSubstitution(&s))
}
