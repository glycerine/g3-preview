// Source: src/ast/rewriter/poly_rewriter_def.h:418-426;
// src/ast/rewriter/bv_rewriter.h:36.
package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
)

func TestBVPolynomialCoefficientProductSourceGuardsAndBorrowedPublication(t *testing.T) {
	sy, q, b := astTestOwners(t)
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	_, e = m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkVar(0, s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	y, e := m.MkVar(1, s)
	astTestMust(t, e)
	astTestPin(t, m, y)
	num := bvAddNum(t, m, 3, 8)
	binary, e := m.MkBitVectorMul(num, x)
	astTestMust(t, e)
	astTestPin(t, m, binary)
	symbolic, e := m.MkBitVectorMul(x, y)
	astTestMust(t, e)
	astTestPin(t, m, symbolic)
	nary, e := m.MkFamilyApp(fid, 6, nil, []ast.ASTHandle{num, x, y})
	astTestMust(t, e)
	astTestPin(t, m, nary)
	unary, e := m.MkFamilyApp(fid, 6, nil, []ast.ASTHandle{num})
	astTestMust(t, e)
	astTestPin(t, m, unary)
	foreign := bvPolynomialRaw(t, m, 0, 6, s, "coefficient-foreign-mul", []ast.ASTHandle{num, x})
	for _, h := range []ast.ASTHandle{x, num, symbolic, nary, unary, foreign} {
		before := m.Stats()
		m.Budget().FailAfter(0)
		yes, body := bvCoefficientProduct(m, h, nil, nil, nil)
		m.Budget().FailAfter(-1)
		if yes || body != nil || m.Stats() != before {
			t.Fatal("unmatched coefficient product touched outputs", yes, body)
		}
	}
	coefficient, e := q.NewValue()
	astTestMust(t, e)
	astTestMust(t, q.SetInt64(&coefficient, 99))
	before, e := m.GetRefCount(x)
	astTestMust(t, e)
	product := y
	var call util.RationalServiceInvocation
	yes, e := bvCoefficientProduct(m, binary, &coefficient, &product, &call)
	astTestMust(t, e)
	got, e := q.String(&coefficient)
	astTestMust(t, e)
	after, e := m.GetRefCount(x)
	astTestMust(t, e)
	if !yes || product != x || got != "3" || after != before {
		t.Fatal("coefficient/power product", yes, product, got, after, before)
	}
	astTestMust(t, call.SourceReady())
	astTestMust(t, call.Cleanup(false))
	astTestMust(t, call.Cleanup(true))
	astTestMust(t, call.EndContext())
	astTestMust(t, q.Del(&coefficient))
}

func TestBVPolynomialCoefficientProductWidthFailureLeavesProductUnchanged(t *testing.T) {
	sy, q, b := astTestOwners(t)
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	_, e = m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkVar(0, s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	n, e := q.NewValue()
	astTestMust(t, e)
	astTestMust(t, q.SetInt64(&n, 257))
	p, e := m.NewParameterRational(&n)
	astTestMust(t, e)
	info, e := m.NewFuncDeclInfo(fid, 0, []ast.ASTParameter{p}, ast.ASTFuncFlags{}, false)
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "coefficient-missing-width"), nil, s, info)
	astTestMust(t, e)
	astTestMust(t, m.DelFuncDeclInfo(&info))
	astTestMust(t, m.DelParameter(&p))
	astTestMust(t, q.Del(&n))
	raw, e := m.MkApp(d, nil)
	astTestMust(t, e)
	astTestPin(t, m, raw)
	mul, e := m.MkBitVectorMul(raw, x)
	astTestMust(t, e)
	astTestPin(t, m, mul)
	coefficient, e := q.NewValue()
	astTestMust(t, e)
	product := raw
	var call util.RationalServiceInvocation
	yes, body := bvCoefficientProduct(m, mul, &coefficient, &product, &call)
	if yes || body == nil || product != raw {
		t.Fatal("failed read published power product", yes, body, product)
	}
	got, e := q.String(&coefficient)
	astTestMust(t, e)
	if got != "257" {
		t.Fatal("coefficient copy prefix lost", got)
	}
	astTestMust(t, call.SourceReady())
	astTestMust(t, call.Cleanup(false))
	astTestMust(t, call.Cleanup(true))
	astTestMust(t, call.EndContext())
	astTestMust(t, q.Del(&coefficient))
}
