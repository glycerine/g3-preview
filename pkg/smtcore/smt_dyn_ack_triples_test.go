package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestDynAckTripleOccurrencesThresholdInstantiationAndReset(t *testing.T) {
	m := astTestManager(t)
	s, e := m.MkSort(symbol.Numerical(701), ast.ASTSortInfo{})
	astTestMust(t, e)
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		astTestMust(t, m.IncRef(h))
		t.Cleanup(func() { astTestMust(t, m.DecRef(h)) })
		return h
	}
	app := func(id uint32, sort ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		h, e := m.MkConst(symbol.Numerical(id), sort)
		astTestMust(t, e)
		return pin(h)
	}
	a, b, c, d, f := app(702, s), app(703, s), app(704, s), app(705, s), app(706, s)
	base := map[ast.ASTHandle]uint32{}
	for _, h := range []ast.ASTHandle{a, b, c, d, f} {
		base[h], e = m.GetRefCount(h)
		astTestMust(t, e)
	}
	state, e := newDynAckTripleState(m, 3)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, state.close()) })
	for i := 0; i < 5; i++ {
		astTestMust(t, state.eqEvent(b, a, c)) // source canonicalizes ID order
	}
	for i := 0; i < 4; i++ {
		astTestMust(t, state.eqEvent(d, b, c))
	}
	for i := 0; i < 2; i++ {
		astTestMust(t, state.eqEvent(d, a, f))
	}
	if len(state.apps) != 3 || len(state.toInstantiate) != 2 || state.apps[0].key.first != a || state.apps[0].key.second != b {
		t.Fatal("source canonical triple keys and one-time threshold scheduling", state.apps, state.toInstantiate)
	}
	key1, key2, key3 := state.apps[0].key, state.apps[1].key, state.apps[2].key
	if state.counts[key1] != 5 || state.counts[key2] != 4 || state.counts[key3] != 2 {
		t.Fatal("source triple occurrence count", state.counts)
	}
	astTestMust(t, state.eqEvent(a, a, c))
	astTestMust(t, state.eqEvent(a, b, b))
	boolSort, e := m.MkBoolSort()
	astTestMust(t, e)
	p, q, r := app(707, boolSort), app(708, boolSort), app(709, boolSort)
	astTestMust(t, state.eqEvent(p, q, r))
	if len(state.apps) != 3 {
		t.Fatal("alias and Boolean-sort guards must precede insertion", state.apps)
	}
	astTestMust(t, state.markInstantiated(key1))
	astTestMust(t, state.eqEvent(a, b, c)) // instantiated triples are ignored
	if _, found := state.counts[key1]; found {
		t.Fatal("instantiated triple returned to occurrence map")
	}
	if len(state.apps) != 3 || len(state.toInstantiate) != 2 {
		t.Fatal("live source retains instantiated app roots until search reset", state.apps, state.toInstantiate)
	}
	astTestMust(t, state.clauseDeleted(key1))
	astTestMust(t, state.eqEvent(a, b, c))
	if len(state.apps) != 4 || state.counts[key1] != 1 {
		t.Fatal("clause deletion must permit a fresh triple", state.apps, state.counts)
	}
	astTestMust(t, state.markInstantiated(key2))
	astTestMust(t, state.initSearch())
	if len(state.apps) != 0 || len(state.counts) != 0 || len(state.toInstantiate) != 0 || len(state.instantiated) != 1 {
		t.Fatal("init_search must retain clause-instantiation state", state.apps, state.counts, state.toInstantiate, state.instantiated)
	}
	astTestMust(t, state.reset())
	if len(state.apps) != 0 || len(state.counts) != 0 || len(state.instantiated) != 0 {
		t.Fatal("reset did not clear triple state")
	}
	for h, want := range base {
		refs, e := m.GetRefCount(h)
		astTestMust(t, e)
		if refs != want {
			t.Fatal("owned triple app reference leaked", h, refs, want)
		}
	}
}
