package ufbv

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestUFBVDemodulatorClassificationOrderingAndBooleanForms(t *testing.T) {
	m := cacheFixture(t)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		cacheMust(t, m.IncRef(h))
		t.Cleanup(func() { cacheMust(t, m.DecRef(h)) })
		return h
	}
	decl := func(id uint32, arity int) ast.ASTHandle {
		t.Helper()
		domain := make([]ast.ASTHandle, arity)
		for i := range domain {
			domain[i] = b
		}
		h, e := m.MkFuncDecl(symbol.Numerical(id), domain, b, ast.ASTFuncDeclInfo{})
		cacheMust(t, e)
		return pin(h)
	}
	app := func(d ast.ASTHandle, args ...ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		h, e := m.MkApp(d, args)
		cacheMust(t, e)
		return pin(h)
	}
	quant := func(body ast.ASTHandle, forall bool) ast.ASTHandle {
		t.Helper()
		h, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: forall, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(105)}, Body: body})
		cacheMust(t, e)
		return pin(h)
	}
	x, e := m.MkVar(0, b)
	cacheMust(t, e)
	pin(x)
	y, e := m.MkVar(1, b)
	cacheMust(t, e)
	pin(y)
	f, g, h := decl(101, 1), decl(102, 1), decl(103, 2)
	fx, gx := app(f, x), app(g, x)
	c := demodulatorClassifier{manager: m}
	for _, tc := range []struct {
		left, right ast.ASTHandle
		want        int
	}{
		{fx, gx, 2},
		{x, app(h, x, y), 1},
		{app(h, x, y), x, -1},
		{x, y, 0},
	} {
		got, e := c.subset(tc.left, tc.right)
		cacheMust(t, e)
		if got != tc.want {
			t.Fatal("source variable-set subset", got, tc.want)
		}
	}
	truth, e := m.MkTrue()
	cacheMust(t, e)
	if got, e := c.subset(truth, truth); e != nil || got != 1 {
		t.Fatal("source value fast path", got, e)
	}
	if got, e := c.smaller(fx, gx); e != nil || got != 1 {
		t.Fatal("declaration ID order", got, e)
	}
	if got, e := c.smaller(gx, fx); e != nil || got != -1 {
		t.Fatal("reverse declaration ID order", got, e)
	}
	if got, e := c.smaller(truth, fx); e != nil || got != 1 {
		t.Fatal("value ordering", got, e)
	}
	if got, e := c.smaller(fx, truth); e != nil || got != -1 {
		t.Fatal("reverse value ordering", got, e)
	}
	notG, e := m.MkNot(gx)
	cacheMust(t, e)
	pin(notG)
	eqFG, e := m.MkIff(fx, gx)
	cacheMust(t, e)
	pin(eqFG)
	notTruth, e := m.MkNot(truth)
	cacheMust(t, e)
	pin(notTruth)
	eqTrueNotG, e := m.MkIff(truth, notG)
	cacheMust(t, e)
	pin(eqTrueNotG)
	eqNotGTrue, e := m.MkIff(notG, truth)
	cacheMust(t, e)
	pin(eqNotGTrue)
	falsity, e := m.MkFalse()
	cacheMust(t, e)
	nested := quant(gx, true)
	for _, tc := range []struct {
		name                    string
		body, large, small      ast.ASTHandle
		forall, wantDemodulator bool
	}{
		{"ordered equality", eqFG, gx, fx, true, true},
		{"negated RHS", eqTrueNotG, gx, notTruth, true, true},
		{"negated LHS", eqNotGTrue, gx, notTruth, true, true},
		{"negative predicate", notG, gx, falsity, true, true},
		{"positive predicate", gx, gx, truth, true, true},
		{"variable refusal", x, ast.ASTHandle{}, ast.ASTHandle{}, true, false},
		{"nested quantifier refusal", nested, ast.ASTHandle{}, ast.ASTHandle{}, true, false},
		{"existential refusal", gx, ast.ASTHandle{}, ast.ASTHandle{}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := quant(tc.body, tc.forall)
			large, small, matched, e := c.classify(q)
			cacheMust(t, e)
			if matched != tc.wantDemodulator {
				t.Fatal("classification", matched)
			}
			if !matched {
				return
			}
			gotLarge, e := m.RefGet(large)
			cacheMust(t, e)
			gotSmall, e := m.RefGet(small)
			cacheMust(t, e)
			if gotLarge != tc.large || gotSmall != tc.small {
				t.Fatal("rule orientation", gotLarge, gotSmall, tc.large, tc.small)
			}
			cacheMust(t, m.DelRef(&small))
			cacheMust(t, m.DelRef(&large))
		})
	}
}
