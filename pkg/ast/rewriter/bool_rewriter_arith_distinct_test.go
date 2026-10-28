package rewriter

// Source: src/ast/arith_decl_plugin.cpp:602-626;
// src/ast/rewriter/bool_rewriter.cpp:650-659.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestBoolRewriterOriginalArithmeticOffsetEquality(t *testing.T) {
	sy, q, b := astTestOwners(t)
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	fid, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	n, e := q.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, q.Del(&n)) }()
	rw := boolRewriter{manager: m}
	no, e := m.MkFalse()
	astTestMust(t, e)
	out := astTestPin(t, m, no)
	for _, kind := range []int32{0, 1} {
		s, e := m.MkFamilySort(fid, kind, nil)
		astTestMust(t, e)
		x, e := m.MkConst(astTestSymbol(t, m, "rewriter-offset-base"), s)
		astTestMust(t, e)
		astTestPin(t, m, x)
		astTestMust(t, q.SetInt64(&n, -1))
		value, e := m.ArithmeticNumeral(&n, kind == 1)
		astTestMust(t, e)
		astTestPin(t, m, value)
		for _, args := range [][]ast.ASTHandle{{x, value}, {value, x}} {
			sum, e := m.MkFamilyApp(fid, 6, nil, args)
			astTestMust(t, e)
			astTestPin(t, m, sum)
			for _, pair := range [][2]ast.ASTHandle{{sum, x}, {x, sum}} {
				astTestMust(t, m.SetRef(out, x))
				status, e := rw.mkEqCore(pair[0], pair[1], out)
				astTestMust(t, e)
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if status != brDone || got != no {
					t.Fatal("offset equality must rewrite to false", kind, pair, status, got)
				}
			}
		}
	}
}
