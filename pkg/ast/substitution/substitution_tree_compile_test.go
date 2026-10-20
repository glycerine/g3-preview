package substitution

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSubstitutionTreeSourceCompilerInsertDuplicateEraseAndReset(t *testing.T) {
	m := offsetManager(t)
	sort, e := m.MkUninterpretedSort(symbol.Numerical(7300), nil)
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
	f, g := decl(7301, 2), decl(7302, 1)
	constant := func(id uint32) ast.ASTHandle {
		d := decl(id, 0)
		h, e := m.MkApp(d, nil)
		if e != nil {
			t.Fatal(e)
		}
		return substPin(t, m, h)
	}
	a, b, c := constant(7303), constant(7304), constant(7305)
	app := func(d ast.ASTHandle, args ...ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkApp(d, args)
		if e != nil {
			t.Fatal(e)
		}
		return substPin(t, m, h)
	}
	one, two, three := app(f, app(g, a), b), app(f, app(g, c), b), app(f, a, c)
	tree, e := NewSubstitutionTree(m)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := tree.Close(); e != nil {
			t.Error(e)
		}
	}()
	for _, h := range []ast.ASTHandle{one, two, three, one} {
		if e = tree.Insert(h); e != nil {
			t.Fatal("insert", e)
		}
	}
	if tree.size != 3 {
		t.Fatal("duplicate size", tree.size)
	}
	if regs, e := tree.ApproxRegisters(); e != nil || regs < 3 {
		t.Fatal("register compilation", regs, e)
	}
	if e = tree.Erase(two); e != nil {
		t.Fatal(e)
	}
	if tree.size != 2 {
		t.Fatal("erase branch", tree.size)
	}
	if e = tree.Erase(two); e != nil {
		t.Fatal("missing erase", e)
	}
	if e = tree.Erase(one); e != nil {
		t.Fatal(e)
	}
	if e = tree.Erase(three); e != nil {
		t.Fatal(e)
	}
	if empty, e := tree.Empty(); e != nil || !empty {
		t.Fatal("empty after erase", empty, e)
	}
	for _, h := range []ast.ASTHandle{one, two} {
		if e = tree.Insert(h); e != nil {
			t.Fatal(e)
		}
	}
	if e = tree.Reset(); e != nil {
		t.Fatal(e)
	}
	if empty, e := tree.Empty(); e != nil || !empty {
		t.Fatal("reset", empty, e)
	}
}

func TestSubstitutionTreeVariableIndexAndOwnerGuards(t *testing.T) {
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
	tree, e := NewSubstitutionTree(m)
	if e != nil {
		t.Fatal(e)
	}
	if e = tree.Insert(v); e != nil {
		t.Fatal(e)
	}
	if e = tree.Insert(v); e != nil {
		t.Fatal(e)
	}
	// Source m_size counts application leaves; a variable-only index is empty.
	if empty, e := tree.Empty(); e != nil || !empty {
		t.Fatal("source variable-only empty", empty, e)
	}
	if e = tree.Erase(v); e != nil {
		t.Fatal(e)
	}
	copyOwner := *tree
	if e = copyOwner.Reset(); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("copied owner", e)
	}
	if e = tree.Close(); e != nil {
		t.Fatal(e)
	}
	if e = tree.Insert(v); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("closed owner", e)
	}
	if e = tree.Close(); e != nil {
		t.Fatal(e)
	}
}
