package rewriter

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestInstantiateQuantifierSourceOrderAndArity(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	v0, e := m.MkVar(0, b)
	astTestMust(t, e)
	v1, e := m.MkVar(1, b)
	astTestMust(t, e)
	body, e := m.MkAnd(v1, v0)
	astTestMust(t, e)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b, b}, Names: []usymbol.Symbol{usymbol.Numerical(0), usymbol.Numerical(1)}, Body: body})
	astTestMust(t, e)
	truth, e := m.MkTrue()
	astTestMust(t, e)
	falsity, e := m.MkFalse()
	astTestMust(t, e)
	out, e := m.NewRef(truth)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&out)) })
	if e = InstantiateQuantifier(m, q, []ast.ASTHandle{truth}, out); !errors.Is(e, ast.ErrASTDomain) {
		t.Fatal("short instantiation vector", e)
	}
	if got, e := m.RefGet(out); e != nil || got != truth {
		t.Fatal("short vector changed output", got, e)
	}
	astTestMust(t, InstantiateQuantifier(m, q, []ast.ASTHandle{truth, falsity}, out))
	got, e := m.RefGet(out)
	astTestMust(t, e)
	left, e := m.GetArg(got, 0)
	astTestMust(t, e)
	right, e := m.GetArg(got, 1)
	astTestMust(t, e)
	if left != truth || right != falsity {
		t.Fatal("standard substitution order", left, right)
	}
}

func TestInstantiateQuantifierSourceInverseShift(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	bound, e := m.MkVar(0, b)
	astTestMust(t, e)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []usymbol.Symbol{usymbol.Numerical(0)}, Body: bound})
	astTestMust(t, e)
	replacement, e := m.MkVar(1, b)
	astTestMust(t, e)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&out)) })
	astTestMust(t, InstantiateQuantifier(m, q, []ast.ASTHandle{replacement}, out))
	got, e := m.RefGet(out)
	astTestMust(t, e)
	index, e := m.GetVarIndex(got)
	astTestMust(t, e)
	if index != 0 {
		t.Fatal("inverse shifted replacement", index)
	}
}
