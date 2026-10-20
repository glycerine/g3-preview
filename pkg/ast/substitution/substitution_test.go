package substitution

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func substPin(t *testing.T, m *ast.ASTManager, h ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	if err := m.IncRef(h); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.DecRef(h); err != nil {
			t.Error(err)
		}
	})
	return h
}

func TestSubstitutionApplyOffsetsScopesAndCacheInvalidation(t *testing.T) {
	m := offsetManager(t)
	sort, err := m.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	v := func(i uint32) ast.ASTHandle {
		h, e := m.MkVar(i, sort)
		if e != nil {
			t.Fatal(e)
		}
		return substPin(t, m, h)
	}
	v0, v3, v5 := v(0), v(3), v(5)
	d, err := m.MkFuncDecl(symbol.Numerical(7001), []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
	if err != nil {
		t.Fatal(err)
	}
	substPin(t, m, d)
	root, err := m.MkApp(d, []ast.ASTHandle{v0})
	if err != nil {
		t.Fatal(err)
	}
	substPin(t, m, root)
	s, err := New(m)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err = s.Reserve(2, 8); err != nil {
		t.Fatal(err)
	}
	if err = s.PushScope(); err != nil {
		t.Fatal(err)
	}
	if err = s.Insert(0, 0, NewExprOffset(v3, 1)); err != nil {
		t.Fatal(err)
	}
	out, err := m.NewRef(ast.ASTHandle{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := m.DelRef(&out); err != nil {
			t.Error(err)
		}
	}()
	if err = s.Apply(2, []uint32{0, 2}, NewExprOffset(root, 0), out); err != nil {
		t.Fatal(err)
	}
	got, err := m.RefGet(out)
	if err != nil {
		t.Fatal(err)
	}
	want, err := m.MkApp(d, []ast.ASTHandle{v5})
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatal("offset application", got, want)
	}
	if yes, err := s.TopScopeHasBindings(); err != nil || !yes {
		t.Fatal("scope binding", yes, err)
	}
	if err = s.PopScope(1); err != nil {
		t.Fatal(err)
	}
	if err = s.Apply(2, []uint32{0, 2}, NewExprOffset(root, 0), out); err != nil {
		t.Fatal(err)
	}
	got, err = m.RefGet(out)
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Fatal("popped binding remained cached", got, root)
	}
	if err = s.PopScope(0); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("zero scope pop", err)
	}
}

func TestSubstitutionQuantifierShiftModuloAndCycles(t *testing.T) {
	m := offsetManager(t)
	sort, err := m.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	v := func(i uint32) ast.ASTHandle {
		h, e := m.MkVar(i, sort)
		if e != nil {
			t.Fatal(e)
		}
		return substPin(t, m, h)
	}
	v0, v1, v3, v4 := v(0), v(1), v(3), v(4)
	s, err := New(m)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err = s.Reserve(1, 8); err != nil {
		t.Fatal(err)
	}
	if err = s.Insert(0, 0, NewExprOffset(v3, 0)); err != nil {
		t.Fatal(err)
	}
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{sort}, Names: []symbol.Symbol{symbol.Numerical(7002)}, Body: v1})
	if err != nil {
		t.Fatal(err)
	}
	substPin(t, m, q)
	out, err := m.NewRef(ast.ASTHandle{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := m.DelRef(&out); err != nil {
			t.Error(err)
		}
	}()
	if err = s.Apply(1, []uint32{0}, NewExprOffset(q, 0), out); err != nil {
		t.Fatal(err)
	}
	got, err := m.RefGet(out)
	if err != nil {
		t.Fatal(err)
	}
	body, err := m.GetQuantifierBody(got)
	if err != nil {
		t.Fatal(err)
	}
	if body != v4 {
		t.Fatal("quantifier range shift", body, v4)
	}
	if err = s.ApplyModulo(1, []uint32{0}, NewExprOffset(v4, 0), NewExprOffset(v4, 0), NewExprOffset(v3, 0), out); err != nil {
		t.Fatal(err)
	}
	got, err = m.RefGet(out)
	if err != nil {
		t.Fatal(err)
	}
	if got != v3 {
		t.Fatal("modulo replacement must still pass through substitution", got, v3)
	}

	if err = s.Reset(); err != nil {
		t.Fatal(err)
	}
	if err = s.Reserve(1, 4); err != nil {
		t.Fatal(err)
	}
	if err = s.Insert(0, 0, NewExprOffset(v1, 0)); err != nil {
		t.Fatal(err)
	}
	if err = s.Insert(1, 0, NewExprOffset(v0, 0)); err != nil {
		t.Fatal(err)
	}
	if yes, err := s.Acyclic(); err != nil || yes {
		t.Fatal("cycle not detected", yes, err)
	}
	if err = s.Apply(1, []uint32{0}, NewExprOffset(v0, 0), out); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("cyclic apply guard", err)
	}
}

func TestSubstitutionBindingAndOwnerGuards(t *testing.T) {
	m := offsetManager(t)
	sort, err := m.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	v, err := m.MkVar(0, sort)
	if err != nil {
		t.Fatal(err)
	}
	substPin(t, m, v)
	s, err := New(m)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Reserve(1, 1); err != nil {
		t.Fatal(err)
	}
	if err = s.Insert(0, 0, NewExprOffset(v, 0)); err != nil {
		t.Fatal(err)
	}
	key, value, err := s.Binding(0)
	if err != nil || key != (VariableOffset{0, 0}) || value != NewExprOffset(v, 0) {
		t.Fatal("binding inspection", key, value, err)
	}
	copyOwner := *s
	if err = copyOwner.Reset(); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("copied owner accepted", err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if err = s.Reset(); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("closed owner accepted", err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
}
