package ufbv

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestRewriteBackIndexReschedulesProcessedAcrossQuantifierAnnotations(t *testing.T) {
	m := cacheFixture(t)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		cacheMust(t, m.IncRef(h))
		t.Cleanup(func() { cacheMust(t, m.DecRef(h)) })
		return h
	}
	makeDecl := func(name uint32, arity int) ast.ASTHandle {
		t.Helper()
		domain := make([]ast.ASTHandle, arity)
		for i := range domain {
			domain[i] = b
		}
		h, e := m.MkFuncDecl(symbol.Numerical(name), domain, b, ast.ASTFuncDeclInfo{})
		cacheMust(t, e)
		return pin(h)
	}
	f, g, constant := makeDecl(61, 1), makeDecl(62, 1), makeDecl(63, 0)
	a, e := m.MkApp(constant, nil)
	cacheMust(t, e)
	pin(a)
	fa, e := m.MkApp(f, []ast.ASTHandle{a})
	cacheMust(t, e)
	pin(fa)
	ga, e := m.MkApp(g, []ast.ASTHandle{a})
	cacheMust(t, e)
	pin(ga)
	both, e := m.MkAnd(fa, ga)
	cacheMust(t, e)
	pin(both)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{b},
		Names: []symbol.Symbol{symbol.Numerical(64)}, Body: fa,
		NoPatterns: []ast.ASTHandle{ga},
	})
	cacheMust(t, e)
	pin(q)
	base := make(map[ast.ASTHandle]uint32)
	for _, h := range []ast.ASTHandle{both, q, ga} {
		base[h], e = m.GetRefCount(h)
		cacheMust(t, e)
	}
	r, e := newRewriteBackIndex(m)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, r.close()) })
	for _, h := range []ast.ASTHandle{both, q, ga} {
		cacheMust(t, r.addProcessed(h))
	}
	if len(r.byDecl[f]) != 2 || len(r.byDecl[g]) != 3 || len(r.byDecl[constant]) != 0 || len(r.processed) != 3 {
		t.Fatal("back index must visit body and no-pattern, but skip constants and interpreted conjunctions")
	}
	cacheMust(t, r.rescheduleProcessed(f))
	if len(r.processed) != 1 || len(r.todo) != 2 || len(r.byDecl[f]) != 0 || len(r.byDecl[g]) != 1 {
		t.Fatal("reschedule must snapshot f occurrences then erase all their back-index memberships")
	}
	if _, found := r.processed[ga]; !found {
		t.Fatal("unrelated g-only expression was rescheduled")
	}
	queued := map[ast.ASTHandle]bool{}
	for _, h := range r.todo {
		queued[h] = true
	}
	if !queued[both] || !queued[q] {
		t.Fatal("reschedule lost processed formula or quantified annotation owner", r.todo)
	}
	for h, refs := range base {
		got, e := m.GetRefCount(h)
		cacheMust(t, e)
		if got != refs+1 {
			t.Fatal("todo transfer and remaining processed roots must retain one reference", h, got, refs)
		}
	}
	cacheMust(t, r.close())
	for h, refs := range base {
		got, e := m.GetRefCount(h)
		cacheMust(t, e)
		if got != refs {
			t.Fatal("back-index owner leaked expression", h, got, refs)
		}
	}
}
