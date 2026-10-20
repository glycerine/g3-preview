package substitution

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestMatcherSourceExamplesAndApplication(t *testing.T) {
	m := offsetManager(t)
	sort, e := m.MkUninterpretedSort(symbol.Numerical(7200), nil)
	if e != nil {
		t.Fatal(e)
	}
	substPin(t, m, sort)
	decl := func(id uint32, arity int) ast.ASTHandle {
		domain := make([]ast.ASTHandle, arity)
		for i := range domain {
			domain[i] = sort
		}
		d, e := m.MkFuncDecl(symbol.Numerical(id), domain, sort, ast.ASTFuncDeclInfo{})
		if e != nil {
			t.Fatal(e)
		}
		return substPin(t, m, d)
	}
	f, g, h := decl(7201, 2), decl(7202, 1), decl(7203, 1)
	constant := func(id uint32) ast.ASTHandle {
		d := decl(id, 0)
		x, e := m.MkApp(d, nil)
		if e != nil {
			t.Fatal(e)
		}
		return substPin(t, m, x)
	}
	a := constant(7204)
	v := func(i uint32) ast.ASTHandle {
		x, e := m.MkVar(i, sort)
		if e != nil {
			t.Fatal(e)
		}
		return substPin(t, m, x)
	}
	x, y := v(0), v(1)
	app := func(d ast.ASTHandle, args ...ast.ASTHandle) ast.ASTHandle {
		x, e := m.MkApp(d, args)
		if e != nil {
			t.Fatal(e)
		}
		return substPin(t, m, x)
	}
	gx, ha := app(g, x), app(h, a)
	pattern := app(f, gx, x)
	instance := app(f, app(g, ha), ha)
	s, e := New(m)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Reserve(2, 10); e != nil {
		t.Fatal(e)
	}
	var matcher Matcher
	ok, e := matcher.Match(pattern, instance, s)
	if e != nil || !ok {
		t.Fatal("source success", ok, e)
	}
	r, found, e := s.Find(0, 0)
	if e != nil || !found || r != NewExprOffset(ha, 1) {
		t.Fatal("source binding", r, found, e)
	}
	out, e := m.NewRef(ast.ASTHandle{})
	if e != nil {
		t.Fatal(e)
	}
	defer m.DelRef(&out)
	probe := app(f, x, y)
	if e = s.Apply(2, []uint32{0, 0}, NewExprOffset(probe, 0), out); e != nil {
		t.Fatal(e)
	}
	got, e := m.RefGet(out)
	if e != nil {
		t.Fatal(e)
	}
	want := app(f, ha, y)
	if got != want {
		t.Fatal("matched substitution application", got, want)
	}

	if e = s.Reset(); e != nil {
		t.Fatal(e)
	}
	if e = s.Reserve(2, 10); e != nil {
		t.Fatal(e)
	}
	bad := app(f, app(g, ha), app(g, ha))
	if ok, e = matcher.Match(pattern, bad, s); e != nil || ok {
		t.Fatal("repeated variable mismatch", ok, e)
	}
}

func TestMatcherRigidVariableAndDeclarationRefusals(t *testing.T) {
	m := offsetManager(t)
	sort, e := m.MkBoolSort()
	if e != nil {
		t.Fatal(e)
	}
	v0, e := m.MkVar(0, sort)
	if e != nil {
		t.Fatal(e)
	}
	v1, e := m.MkVar(1, sort)
	if e != nil {
		t.Fatal(e)
	}
	substPin(t, m, v0)
	substPin(t, m, v1)
	s, e := New(m)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Reserve(2, 4); e != nil {
		t.Fatal(e)
	}
	var matcher Matcher
	truth, e := m.MkTrue()
	if e != nil {
		t.Fatal(e)
	}
	falsity, e := m.MkFalse()
	if e != nil {
		t.Fatal(e)
	}
	if ok, e := matcher.Match(truth, v0, s); e != nil || ok {
		t.Fatal("rigid pattern matched variable", ok, e)
	}
	if ok, e := matcher.Match(truth, falsity, s); e != nil || ok {
		t.Fatal("different declarations matched", ok, e)
	}
	if ok, e := matcher.Match(v0, v1, s); e != nil || !ok {
		t.Fatal("variable pattern", ok, e)
	}
}
