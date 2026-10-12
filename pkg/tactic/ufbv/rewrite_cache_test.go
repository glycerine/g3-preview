package ufbv

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func cacheFixture(t *testing.T) *ast.ASTManager {
	return cacheFixtureProofMode(t, ast.ASTProofDisabled)
}

func cacheFixtureProofMode(t *testing.T, proofMode ast.ASTProofMode) *ast.ASTManager {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	s, e := symbol.NewManager(symbol.Config{Budget: b})
	cacheMust(t, e)
	q, e := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	cacheMust(t, e)
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b, ProofMode: proofMode})
	cacheMust(t, e)
	t.Cleanup(func() {
		cacheMust(t, m.Close())
		cacheMust(t, q.Close())
		cacheMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatalf("owner graph retained %d bytes", b.Used())
		}
	})
	return m
}

func cacheMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}

func TestRewriteCacheSourceLazyResetAndReferenceOrder(t *testing.T) {
	m := cacheFixture(t)
	sort, e := m.MkBoolSort()
	cacheMust(t, e)
	key, e := m.MkVar(1001, sort)
	cacheMust(t, e)
	v1, e := m.MkVar(1002, sort)
	cacheMust(t, e)
	v2, e := m.MkVar(1003, sort)
	cacheMust(t, e)
	for _, h := range []ast.ASTHandle{key, v1, v2} {
		cacheMust(t, m.IncRef(h))
	}
	refs := func(h ast.ASTHandle, want uint32) {
		t.Helper()
		rc, e := m.GetRefCount(h)
		if e != nil || rc != want {
			t.Fatalf("ref count %d, want %d: %v", rc, want, e)
		}
	}
	c, e := newRewriteCache(m)
	cacheMust(t, e)
	cacheMust(t, c.insert(key, v1, false))
	refs(key, 2)
	refs(v1, 2)
	if got, ok, e := c.get(key); e != nil || !ok || got.expr != v1 || got.done {
		t.Fatal("initial cache entry", got, ok, e)
	}
	cacheMust(t, c.reset())
	if present, e := c.contains(key); e != nil || present {
		t.Fatal("reset must invalidate current timestamp", present, e)
	}
	refs(key, 2) // reset is lazy and keeps the stale entry owned
	refs(v1, 2)
	cacheMust(t, c.insert(key, v2, true))
	refs(key, 2)
	refs(v1, 1)
	refs(v2, 2)
	cacheMust(t, c.insert(key, v2, false)) // self replacement
	refs(v2, 2)
	if got, ok, e := c.get(key); e != nil || !ok || got.expr != v2 || got.done {
		t.Fatal("replacement cache entry", got, ok, e)
	}
	cacheMust(t, c.reset())
	refs(key, 2)
	refs(v2, 2)
	cacheMust(t, c.insert(key, v1, true))
	refs(v2, 1)
	cacheMust(t, c.close())
	refs(key, 1)
	refs(v1, 1)
	for _, h := range []ast.ASTHandle{key, v1, v2} {
		cacheMust(t, m.DecRef(h))
	}
}

func TestRewriteCacheSourceTimestampWrap(t *testing.T) {
	m := cacheFixture(t)
	sort, e := m.MkBoolSort()
	cacheMust(t, e)
	key, e := m.MkVar(2001, sort)
	cacheMust(t, e)
	value, e := m.MkVar(2002, sort)
	cacheMust(t, e)
	for _, h := range []ast.ASTHandle{key, value} {
		cacheMust(t, m.IncRef(h))
	}
	c, e := newRewriteCache(m)
	cacheMust(t, e)
	cacheMust(t, c.insert(key, value, true))
	cacheMust(t, c.reset())
	if present, e := c.contains(key); e != nil || present {
		t.Fatal("reset timestamp", present, e)
	}
	for _, h := range []ast.ASTHandle{key, value} {
		rc, e := m.GetRefCount(h)
		if e != nil || rc != 2 {
			t.Fatal("source reset retains stale references", rc, e)
		}
	}
	c.timestamp = ^uint32(0)
	cacheMust(t, c.reset())
	if c.timestamp != 0 || len(c.entries) != 0 {
		t.Fatal("timestamp wrap must physically flush", c.timestamp, len(c.entries))
	}
	for _, h := range []ast.ASTHandle{key, value} {
		rc, e := m.GetRefCount(h)
		if e != nil || rc != 1 {
			t.Fatal("wrap cleanup", rc, e)
		}
	}
	cacheMust(t, c.insert(key, value, false))
	cacheMust(t, c.close())
	for _, h := range []ast.ASTHandle{key, value} {
		rc, e := m.GetRefCount(h)
		if e != nil || rc != 1 {
			t.Fatal("destructor cleanup", rc, e)
		}
		cacheMust(t, m.DecRef(h))
	}
}
