package substitution

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestUnifierBanksExistingBindingsAndRepeatedReverse(t *testing.T) {
	m := offsetManager(t)
	sort, e := m.MkBoolSort()
	if e != nil {
		t.Fatal(e)
	}
	v := func(i uint32) ast.ASTHandle {
		h, e := m.MkVar(i, sort)
		if e != nil {
			t.Fatal(e)
		}
		return substPin(t, m, h)
	}
	v0, v1 := v(0), v(1)
	s, e := New(m)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Reserve(2, 4); e != nil {
		t.Fatal(e)
	}
	u, e := NewUnifier(m)
	if e != nil {
		t.Fatal(e)
	}
	defer u.Close()
	ok, e := u.UnifyPair(v0, v1, s, false)
	if e != nil || !ok {
		t.Fatal("same-bank unify", ok, e)
	}
	r, found, e := s.Find(0, 0)
	if e != nil || !found || r != NewExprOffset(v1, 0) {
		t.Fatal("same-bank binding", r, found, e)
	}
	ok, e = u.UnifyPair(v1, v0, s, false)
	if e != nil || !ok {
		t.Fatal("reverse existing binding", ok, e)
	}

	if e = s.Reset(); e != nil {
		t.Fatal(e)
	}
	if e = s.Reserve(2, 4); e != nil {
		t.Fatal(e)
	}
	ok, e = u.UnifyPair(v0, v0, s, true)
	if e != nil || !ok {
		t.Fatal("banked unify", ok, e)
	}
	r, found, e = s.Find(0, 0)
	if e != nil || !found || r != NewExprOffset(v0, 1) {
		t.Fatal("banked binding", r, found, e)
	}
}

func TestUnifierApplicationsMismatchMultipleAndOccursCheck(t *testing.T) {
	m := offsetManager(t)
	sort, e := m.MkBoolSort()
	if e != nil {
		t.Fatal(e)
	}
	v := func(i uint32) ast.ASTHandle {
		h, e := m.MkVar(i, sort)
		if e != nil {
			t.Fatal(e)
		}
		return substPin(t, m, h)
	}
	v0, v1, v2 := v(0), v(1), v(2)
	decl := func(id uint32) ast.ASTHandle {
		d, e := m.MkFuncDecl(symbol.Numerical(id), []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
		if e != nil {
			t.Fatal(e)
		}
		return substPin(t, m, d)
	}
	f, g := decl(7101), decl(7102)
	app := func(d, a ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkApp(d, []ast.ASTHandle{a})
		if e != nil {
			t.Fatal(e)
		}
		return substPin(t, m, h)
	}
	fv0, fv1, gv1 := app(f, v0), app(f, v1), app(g, v1)
	s, e := New(m)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Reserve(3, 8); e != nil {
		t.Fatal(e)
	}
	u, e := NewUnifier(m)
	if e != nil {
		t.Fatal(e)
	}
	defer u.Close()
	if ok, e := u.UnifyPair(fv0, gv1, s, false); e != nil || ok {
		t.Fatal("declaration mismatch", ok, e)
	}
	if e = s.Reset(); e != nil {
		t.Fatal(e)
	}
	if e = s.Reserve(3, 8); e != nil {
		t.Fatal(e)
	}
	if ok, e := u.Unify([]ast.ASTHandle{fv0, fv1, app(f, v2)}, s, true); e != nil || !ok {
		t.Fatal("multiple unification", ok, e)
	}
	if e = s.Reset(); e != nil {
		t.Fatal(e)
	}
	if e = s.Reserve(1, 8); e != nil {
		t.Fatal(e)
	}
	if ok, e := u.UnifyPair(v0, fv0, s, false); e != nil || ok {
		t.Fatal("occurs check", ok, e)
	}
}

func TestUnifierQuantifierAndOwnerGuards(t *testing.T) {
	m := offsetManager(t)
	sort, e := m.MkBoolSort()
	if e != nil {
		t.Fatal(e)
	}
	v, e := m.MkVar(0, sort)
	if e != nil {
		t.Fatal(e)
	}
	substPin(t, m, v)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{sort}, Names: []symbol.Symbol{symbol.Numerical(7103)}, Body: v})
	if e != nil {
		t.Fatal(e)
	}
	substPin(t, m, q)
	s, e := New(m)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Reserve(2, 2); e != nil {
		t.Fatal(e)
	}
	u, e := NewUnifier(m)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = u.UnifyPair(q, v, s, false); !errors.Is(e, ast.ErrASTDomain) {
		t.Fatal("quantifier guard", e)
	}
	copyOwner := *u
	if _, e = copyOwner.UnifyPair(v, v, s, false); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("copied owner", e)
	}
	if e = u.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = u.UnifyPair(v, v, s, false); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("closed owner", e)
	}
	if e = u.Close(); e != nil {
		t.Fatal(e)
	}
}
