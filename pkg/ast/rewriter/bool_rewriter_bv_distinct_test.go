// Source: src/ast/bv_decl_plugin.cpp:636-669;
// src/ast/rewriter/bool_rewriter.cpp:650-659.
package rewriter

import (
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBoolRewriterOriginalBitVectorOffsetEquality(t *testing.T) {
	sy, q, b := astTestOwners(t)
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	_, e = m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	n, e := q.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, q.Del(&n)) }()
	rw := boolRewriter{manager: m}
	no, e := m.MkFalse()
	astTestMust(t, e)
	out := astTestPin(t, m, no)
	for _, width := range []uint32{1, 8, 65} {
		s, e := m.BitVectorSort(width)
		astTestMust(t, e)
		x, e := m.MkConst(astTestSymbol(t, m, "rewriter-bv-offset"), s)
		astTestMust(t, e)
		astTestPin(t, m, x)
		astTestMust(t, q.SetInt64(&n, -1))
		value, e := m.BitVectorNumeral(&n, width)
		astTestMust(t, e)
		astTestPin(t, m, value)
		sum, e := m.MkFamilyApp(fid, 4, nil, []ast.ASTHandle{value, x})
		astTestMust(t, e)
		astTestPin(t, m, sum)
		for _, pair := range [][2]ast.ASTHandle{{sum, x}, {x, sum}} {
			astTestMust(t, m.SetRef(out, x))
			status, e := rw.mkEqCore(pair[0], pair[1], out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			if status != brDone || got != no {
				t.Fatal("original BV offset equality", width, status, got)
			}
		}
	}
}
