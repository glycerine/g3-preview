package ufbv

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestRewriteTraversalSourceEntryAssertionAfterEmptyIndexBypass(t *testing.T) {
	m := cacheFixture(t)
	if diagnostic.DebugBuild {
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
		})
	}
	sort, e := m.MkBoolSort()
	cacheMust(t, e)
	root, e := m.MkVar(362, sort)
	cacheMust(t, e)
	cacheMust(t, m.IncRef(root))
	t.Cleanup(func() { cacheMust(t, m.DecRef(root)) })
	r, e := newRewriteTraversal(m)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, r.close()) })
	index, e := newRewriteRuleIndex(m)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, index.close()) })
	cacheMust(t, r.pushTodo(root))
	ref, e := r.rewrite(index, nil, root)
	cacheMust(t, e)
	h, e := m.RefGet(ref)
	cacheMust(t, e)
	if h != root {
		t.Fatal("empty forward index must bypass the source assertion")
	}
	cacheMust(t, m.DelRef(&ref))
	index.byFunc[sort] = &rewriteForwardSet{}
	_, e = r.rewrite(index, nil, root)
	if diagnostic.DebugBuild {
		if !errors.Is(e, diagnostic.ErrDebugAssertion) {
			t.Fatal("source cpp362 assertion", e)
		}
	} else if !errors.Is(e, ast.ErrASTPending) {
		t.Fatal("erased assertion must retain the safe pending guard", e)
	}
	if len(r.todo) != 1 || r.todo[0] != root {
		t.Fatal("entry assertion consumed the pending traversal", r.todo)
	}
	delete(index.byFunc, sort)
}
