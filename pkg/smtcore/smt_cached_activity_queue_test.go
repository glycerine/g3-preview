package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func cachedActivityFixture(t *testing.T) (*ast.ASTManager, *SMTInternalizerState, *SMTActivityCaseSplitQueue) {
	t.Helper()
	m := astTestManager(t)
	_, _, s := smtInternalizerFixture(t, m, nil)
	params := SMTParamsState{CaseSplitStrategy: 2, InvDecay: 1.052}
	astTestMust(t, s.configureBooleanHeuristics(params))
	q, err := NewSMTActivityCaseSplitQueue(s)
	astTestMust(t, err)
	astTestMust(t, q.ConfigurePhases(params))
	return m, s, q
}

func TestSMTCachedActivityRetainsEveryPositiveDeletionReference(t *testing.T) {
	m, s, q := cachedActivityFixture(t)
	x := smtRelevancyBool(t, m, "cached-domain-repeat")
	astTestMust(t, m.IncRef(x))
	defer func() { astTestMust(t, m.DecRef(x)) }()
	before, err := m.GetRefCount(x)
	astTestMust(t, err)
	astTestMust(t, s.beginSearchPersistence(0, SMTParamsState{}))
	var last float64
	for cycle := 1; cycle <= 3; cycle++ {
		astTestMust(t, s.PushScope())
		astTestMust(t, s.Internalize(x, true))
		v, _ := s.BoolVar(x)
		if s.activity[v] != last {
			t.Fatal("cached value must restore before insertion", s.activity[v], last)
		}
		for i := 0; i < cycle; i++ {
			astTestMust(t, s.incrementBooleanActivity(v))
		}
		last = s.activity[v]
		got, _, err := q.NextCaseSplit()
		astTestMust(t, err)
		if got != v || s.basicActivity.Contains(v) {
			t.Fatal("cached variant must consume parent heap selection", got, v)
		}
		astTestMust(t, s.PopScopes(1))
		refs, err := m.GetRefCount(x)
		astTestMust(t, err)
		if refs != before+uint32(cycle) {
			t.Fatal("every deletion must retain another domain reference", refs, before+uint32(cycle))
		}
		if len(s.activityCache) != 1 || s.activityCache[x] != last {
			t.Fatal("cache value did not overwrite independently of reference domain")
		}
	}
	astTestMust(t, s.endSearchPersistence())
	refs, err := m.GetRefCount(x)
	astTestMust(t, err)
	if refs != before+3 {
		t.Fatal("end_search_eh must retain cache domain", refs)
	}
	astTestMust(t, s.resetBooleanActivityCache())
	refs, err = m.GetRefCount(x)
	astTestMust(t, err)
	if refs != before || len(s.activityCache) != 0 || len(s.activityCacheDomain) != 0 || s.activityCacheRetiring {
		t.Fatal("new search must release all duplicate references", refs, before)
	}
	astTestMust(t, s.PushScope())
	astTestMust(t, s.Internalize(x, true))
	v, _ := s.BoolVar(x)
	if s.activity[v] != 0 {
		t.Fatal("old search activity survived reset", s.activity[v])
	}
	astTestMust(t, s.PopScopes(1))
}

func TestSMTCachedActivityRestoredPriorityPrecedesHeapInsertion(t *testing.T) {
	m, s, q := cachedActivityFixture(t)
	x := smtRelevancyBool(t, m, "cached-priority-x")
	y := smtRelevancyBool(t, m, "cached-priority-y")
	astTestMust(t, s.beginSearchPersistence(0, SMTParamsState{}))
	t.Cleanup(func() { astTestMust(t, s.endSearchPersistence()) })
	astTestMust(t, s.PushScope())
	astTestMust(t, s.Internalize(x, true))
	xv, _ := s.BoolVar(x)
	for i := 0; i < 4; i++ {
		astTestMust(t, s.incrementBooleanActivity(xv))
	}
	astTestMust(t, s.PopScopes(1))
	astTestMust(t, s.Internalize(y, true))
	yv, _ := s.BoolVar(y)
	for i := 0; i < 3; i++ {
		astTestMust(t, s.incrementBooleanActivity(yv))
	}
	astTestMust(t, s.Internalize(x, true))
	xv, _ = s.BoolVar(x)
	got, _, err := q.NextCaseSplit()
	astTestMust(t, err)
	if got != xv || got == yv || s.activity[xv] != 4 {
		t.Fatal("cache restoration did not determine insertion priority", got, xv, yv)
	}
}

func TestSMTCachedActivitySelectionConsumesNativeHeap(t *testing.T) {
	m, s, q := cachedActivityFixture(t)
	a := smtRelevancyBool(t, m, "cached-equal-a")
	b := smtRelevancyBool(t, m, "cached-equal-b")
	astTestMust(t, s.Internalize(a, true))
	astTestMust(t, s.Internalize(b, true))
	av, _ := s.BoolVar(a)
	bv, _ := s.BoolVar(b)
	for _, want := range []int32{bv, av, SMTNullBoolVar} {
		got, _, err := q.NextCaseSplit()
		astTestMust(t, err)
		if got != want {
			t.Fatal("cached queue lost native heap order/consumption", got, want)
		}
	}
	available, err := q.hasCaseSplit()
	astTestMust(t, err)
	if available {
		t.Fatal("cached availability regenerated consumed variables")
	}
}
