package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:52-208.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestBVRewriterDispatchesOriginalDeclarationFamilies(t *testing.T) {
	m, fid := bvAddFixture(t)
	c, g, _ := bvPolynomialParamsFixture(t)
	p, err := NewBVRewriter(m, c, g, nil)
	astTestMust(t, err)
	defer func() { astTestMust(t, p.Close()) }()
	astTestMust(t, p.SetMkBVToNumeral(true))

	zero := bvAddNum(t, m, 0, 8)
	one := bvAddNum(t, m, 1, 8)
	two := bvAddNum(t, m, 2, 8)
	three := bvAddNum(t, m, 3, 8)
	trueValue, err := m.MkTrue()
	astTestMust(t, err)
	astTestPin(t, m, trueValue)
	f, err := m.MkFalse()
	astTestMust(t, err)
	astTestPin(t, m, f)

	integerValue, err := m.NumeralService().NewValue()
	astTestMust(t, err)
	astTestMust(t, m.NumeralService().SetInt64(&integerValue, 257))
	integer, err := m.ArithmeticNumeral(&integerValue, true)
	astTestMust(t, err)
	astTestPin(t, m, integer)
	astTestMust(t, m.NumeralService().Del(&integerValue))

	parameterized := func(kind, value int32, args []ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		parameter, e := m.NewParameterInt(value)
		astTestMust(t, e)
		app, e := m.MkFamilyApp(fid, kind, []ast.ASTParameter{parameter}, args)
		astTestMust(t, e)
		astTestMust(t, m.DelParameter(&parameter))
		astTestPin(t, m, app)
		return app
	}
	ordinary := func(kind int32, args []ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		app, e := m.MkFamilyApp(fid, kind, nil, args)
		astTestMust(t, e)
		astTestPin(t, m, app)
		return app
	}
	extract, err := m.MkBitVectorExtract(5, 2, zero)
	astTestMust(t, err)
	astTestPin(t, m, extract)

	rows := []struct {
		name string
		app  ast.ASTHandle
		args []ast.ASTHandle
	}{
		{"bit0", ordinary(2, nil), nil},
		{"neg", ordinary(3, []ast.ASTHandle{one}), []ast.ASTHandle{one}},
		{"add", ordinary(4, []ast.ASTHandle{one, two}), []ast.ASTHandle{one, two}},
		{"sub", ordinary(5, []ast.ASTHandle{three, one}), []ast.ASTHandle{three, one}},
		{"mul", ordinary(6, []ast.ASTHandle{two, three}), []ast.ASTHandle{two, three}},
		{"udiv-internal", ordinary(18, []ast.ASTHandle{three, one}), []ast.ASTHandle{three, one}},
		{"ugt", ordinary(28, []ast.ASTHandle{three, two}), []ast.ASTHandle{three, two}},
		{"and", ordinary(30, []ast.ASTHandle{one, three}), []ast.ASTHandle{one, three}},
		{"concat", ordinary(37, []ast.ASTHandle{one, two}), []ast.ASTHandle{one, two}},
		{"sign-extend", parameterized(38, 2, []ast.ASTHandle{one}), []ast.ASTHandle{one}},
		{"zero-extend", parameterized(39, 2, []ast.ASTHandle{one}), []ast.ASTHandle{one}},
		{"extract", extract, []ast.ASTHandle{zero}},
		{"repeat", parameterized(41, 2, []ast.ASTHandle{one}), []ast.ASTHandle{one}},
		{"redor", ordinary(42, []ast.ASTHandle{one}), []ast.ASTHandle{one}},
		{"compare", ordinary(44, []ast.ASTHandle{one, one}), []ast.ASTHandle{one, one}},
		{"shift-left", ordinary(45, []ast.ASTHandle{one, two}), []ast.ASTHandle{one, two}},
		{"rotate-left", parameterized(48, 2, []ast.ASTHandle{one}), []ast.ASTHandle{one}},
		{"extended-rotate-right", ordinary(51, []ast.ASTHandle{one, two}), []ast.ASTHandle{one, two}},
		{"unsigned-mul-bound", ordinary(52, []ast.ASTHandle{two, three}), []ast.ASTHandle{two, three}},
		{"mkbv", ordinary(56, []ast.ASTHandle{trueValue, f}), []ast.ASTHandle{trueValue, f}},
		{"int2bv", parameterized(57, 8, []ast.ASTHandle{integer}), []ast.ASTHandle{integer}},
		{"bv2int", ordinary(58, []ast.ASTHandle{three}), []ast.ASTHandle{three}},
	}

	out, err := m.NewRef(zero)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			decl, e := m.GetDecl(row.app)
			astTestMust(t, e)
			status, e := p.RewriteApp(decl, ast.SliceArguments(row.args), out)
			astTestMust(t, e)
			if status == RewriteFailed {
				t.Fatal("valid source dispatch refused")
			}
			if _, e = m.RefGet(out); e != nil {
				t.Fatal(e)
			}
		})
	}
}

func TestBVRewriterDispatchRefusalsPreserveOutput(t *testing.T) {
	m, fid := bvAddFixture(t)
	c, g, _ := bvPolynomialParamsFixture(t)
	p, err := NewBVRewriter(m, c, g, nil)
	astTestMust(t, err)
	defer func() { astTestMust(t, p.Close()) }()
	one := bvAddNum(t, m, 1, 8)
	add, err := m.MkFamilyApp(fid, 4, nil, []ast.ASTHandle{one, one})
	astTestMust(t, err)
	decl, err := m.GetDecl(add)
	astTestMust(t, err)
	out := astTestPin(t, m, one)

	status, err := p.RewriteApp(decl, ast.AppArguments{}, out)
	astTestMust(t, err)
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if status != RewriteFailed || got != one {
		t.Fatal(status, got)
	}

	trueApp, err := m.MkTrue()
	astTestMust(t, err)
	trueDecl, err := m.GetDecl(trueApp)
	astTestMust(t, err)
	status, err = p.RewriteApp(trueDecl, ast.SliceArguments([]ast.ASTHandle{one}), out)
	astTestMust(t, err)
	got, err = m.RefGet(out)
	astTestMust(t, err)
	if status != RewriteFailed || got != one {
		t.Fatal(status, got)
	}
}
