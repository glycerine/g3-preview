package ufbv

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestMatchSubstConcretePairCacheAndVariableBindings(t *testing.T) {
	m := cacheFixture(t)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		cacheMust(t, m.IncRef(h))
		t.Cleanup(func() { cacheMust(t, m.DecRef(h)) })
		return h
	}
	declare := func(name string, arity int) ast.ASTHandle {
		t.Helper()
		n, e := m.Symbols.Intern(name)
		cacheMust(t, e)
		domain := make([]ast.ASTHandle, arity)
		for i := range domain {
			domain[i] = b
		}
		d, e := m.MkFuncDecl(n, domain, b, ast.ASTFuncDeclInfo{})
		cacheMust(t, e)
		return pin(d)
	}
	app := func(d ast.ASTHandle, args ...ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		h, e := m.MkApp(d, args)
		cacheMust(t, e)
		return pin(h)
	}
	x, e := m.MkVar(0, b)
	cacheMust(t, e)
	pin(x)
	y, e := m.MkVar(1, b)
	cacheMust(t, e)
	pin(y)
	f, g, h := declare("match_f", 2), declare("match_g", 1), declare("match_h", 1)
	a, c := app(declare("match_a", 0)), app(declare("match_c", 0))
	gx, gy, ga, gc := app(g, x), app(g, y), app(g, a), app(g, c)
	pattern := app(f, gx, gx)
	instance := app(f, ga, ga)
	s, e := newMatchSubst(m)
	cacheMust(t, e)
	ok, e := s.isInstance(pattern, instance)
	cacheMust(t, e)
	if !ok || s.bindings[x] != a || len(s.cache) != 1 {
		t.Fatal("shared expression pair must be cached and bind x once", ok, s.bindings, s.cache)
	}

	ok, e = s.isInstance(pattern, app(f, ga, gc))
	cacheMust(t, e)
	if ok {
		t.Fatal("repeated variable bound to different arguments")
	}
	ok, e = s.isInstance(pattern, app(f, ga, app(h, a)))
	cacheMust(t, e)
	if ok {
		t.Fatal("quick declaration mismatch")
	}

	// The source processes the last argument first. It binds x to a in
	// (g x, g a), then the equal non-ground (g x, g x) must reject that
	// non-self binding.
	ok, e = s.isInstance(pattern, app(f, gx, ga))
	cacheMust(t, e)
	if ok {
		t.Fatal("identical non-ground branch ignored earlier non-self binding")
	}
	ok, e = s.isInstance(app(f, x, gy), app(f, a, gy))
	cacheMust(t, e)
	if !ok || s.bindings[x] != a || s.bindings[y] != y {
		t.Fatal("identical non-ground branch did not bind unbound y to itself", ok, s.bindings)
	}

	ok, e = s.isInstance(app(f, a, a), app(f, x, x))
	cacheMust(t, e)
	if ok {
		t.Fatal("ground pattern may not match a non-ground instance")
	}
	ok, e = s.isInstance(x, app(h, a))
	cacheMust(t, e)
	if !ok {
		t.Fatal("source direct variable instance check")
	}
	ok, e = s.isInstance(app(f, a, a), app(f, a, a))
	cacheMust(t, e)
	if !ok || !s.allEqual {
		t.Fatal("source equal-arguments fast path", ok, s.allEqual)
	}
}
