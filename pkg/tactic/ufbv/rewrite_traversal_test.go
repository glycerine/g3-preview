package ufbv

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestRewriteTraversalSourceReverseChildrenAndOwningTodo(t *testing.T) {
	m := cacheFixture(t)
	sort, e := m.MkBoolSort()
	cacheMust(t, e)
	vars := make([]ast.ASTHandle, 3)
	for i := range vars {
		vars[i], e = m.MkVar(uint32(3001+i), sort)
		cacheMust(t, e)
		cacheMust(t, m.IncRef(vars[i]))
	}
	name, e := m.Symbols.Intern("visit")
	cacheMust(t, e)
	decl, e := m.MkFuncDecl(name, []ast.ASTHandle{sort, sort, sort}, sort, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	app, e := m.MkApp(decl, vars)
	cacheMust(t, e)
	cacheMust(t, m.IncRef(app))
	baseline := make([]uint32, len(vars))
	for i, h := range vars {
		baseline[i], e = m.GetRefCount(h)
		cacheMust(t, e)
	}
	r, e := newRewriteTraversal(m)
	cacheMust(t, e)
	ready, e := r.visitChildren(app)
	cacheMust(t, e)
	if ready || len(r.todo) != 3 || r.todo[0] != vars[2] || r.todo[1] != vars[1] || r.todo[2] != vars[0] {
		t.Fatal("source reverse child scheduling", ready, r.todo)
	}
	for i, h := range vars {
		refs, e := m.GetRefCount(h)
		cacheMust(t, e)
		if refs != baseline[i]+1 {
			t.Fatal("todo must own each child", i, refs, baseline[i])
		}
	}
	cacheMust(t, r.popTodo())
	if len(r.todo) != 2 {
		t.Fatal("pop-before-dec todo length", r.todo)
	}
	cacheMust(t, r.close())
	for i, h := range vars {
		refs, e := m.GetRefCount(h)
		cacheMust(t, e)
		if refs != baseline[i] {
			t.Fatal("todo destructor leaked child", i, refs, baseline[i])
		}
	}
	cacheMust(t, m.DecRef(app))
	for _, h := range vars {
		cacheMust(t, m.DecRef(h))
	}
}

func TestRewriteTraversalSourceCycleCacheUsesActualTodo(t *testing.T) {
	m := cacheFixture(t)
	sort, e := m.MkBoolSort()
	cacheMust(t, e)
	a, e := m.MkVar(4001, sort)
	cacheMust(t, e)
	b, e := m.MkVar(4002, sort)
	cacheMust(t, e)
	cacheMust(t, m.IncRef(a))
	cacheMust(t, m.IncRef(b))
	name, e := m.Symbols.Intern("cycle")
	cacheMust(t, e)
	decl, e := m.MkFuncDecl(name, []ast.ASTHandle{sort, sort, sort}, sort, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	app, e := m.MkApp(decl, []ast.ASTHandle{a, b, a})
	cacheMust(t, e)
	cacheMust(t, m.IncRef(app))
	r, e := newRewriteTraversal(m)
	cacheMust(t, e)
	cacheMust(t, r.pushTodo(a))
	ready, e := r.visitChildren(app)
	cacheMust(t, e)
	if ready || len(r.todo) != 2 || r.todo[0] != a || r.todo[1] != b {
		t.Fatal("cycle must cache A and schedule only B", ready, r.todo)
	}
	entry, found, e := r.cache.get(a)
	cacheMust(t, e)
	if !found || !entry.done || entry.expr != a {
		t.Fatal("actual array-map cycle entry", entry, found)
	}
	ready, e = r.visitChildren(app)
	cacheMust(t, e)
	if !ready || len(r.todo) != 2 {
		t.Fatal("second visit must mark B cycle and skip cached A", ready, r.todo)
	}
	cacheMust(t, r.close())
	cacheMust(t, m.DecRef(app))
	cacheMust(t, m.DecRef(a))
	cacheMust(t, m.DecRef(b))
}
