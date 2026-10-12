package ufbv

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestMatchSubstTwoOffsetRHSAndQuantifierShift(t *testing.T) {
	m := cacheFixture(t)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		cacheMust(t, m.IncRef(h))
		t.Cleanup(func() { cacheMust(t, m.DecRef(h)) })
		return h
	}
	vars := make([]ast.ASTHandle, 3)
	for i := range vars {
		vars[i], e = m.MkVar(uint32(i), b)
		cacheMust(t, e)
		pin(vars[i])
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
	app := func(d ast.ASTHandle, args ...ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		h, e := m.MkApp(d, args)
		cacheMust(t, e)
		return pin(h)
	}
	f, g, aDecl := makeDecl(41, 2), makeDecl(42, 1), makeDecl(43, 0)
	a := app(aDecl)
	lhs := app(f, vars[0], vars[1])
	rhs := app(g, vars[0])
	want := app(g, vars[1])
	s, e := newMatchSubst(m)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, s.close()) })
	out, ok, e := s.rewriteRule(lhs, rhs, []ast.ASTHandle{vars[1], a})
	cacheMust(t, e)
	if !ok {
		t.Fatal("rule should match")
	}
	got, e := m.RefGet(out)
	cacheMust(t, e)
	if got != want {
		t.Fatal("offset-1 replacement must not recursively substitute y with a", got, want)
	}
	cacheMust(t, m.DelRef(&out))

	inside := app(g, vars[1])
	quant, e := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{b},
		Names: []symbol.Symbol{symbol.Numerical(44)}, Body: inside,
		NoPatterns: []ast.ASTHandle{inside},
	})
	cacheMust(t, e)
	pin(quant)
	shifted := app(g, vars[2])
	wantQuant, e := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{b},
		Names: []symbol.Symbol{symbol.Numerical(44)}, Body: shifted,
		NoPatterns: []ast.ASTHandle{shifted},
	})
	cacheMust(t, e)
	pin(wantQuant)
	out, ok, e = s.rewriteRule(lhs, quant, []ast.ASTHandle{vars[1], a})
	cacheMust(t, e)
	if !ok {
		t.Fatal("quantified RHS should match")
	}
	got, e = m.RefGet(out)
	cacheMust(t, e)
	if got != wantQuant {
		t.Fatal("matched free variable must shift above quantifier binder in body and annotation", got, wantQuant)
	}
	cacheMust(t, s.close())
	got, e = m.RefGet(out)
	cacheMust(t, e)
	if got != wantQuant {
		t.Fatal("caller result must survive matcher close")
	}
	cacheMust(t, m.DelRef(&out))
}
