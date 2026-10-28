package ast

// Source: src/ast/ast.h:1026; src/ast/bv_decl_plugin.h:271-285;
// src/ast/arith_decl_plugin.h:202-217.

import "testing"

func TestASTOriginalPluginUninterpretedClassification(t *testing.T) {
	m := astTestManager(t)
	arith, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	bv, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	for _, fid := range []FamilyID{NullFamilyID, 0, arith, bv, 42} {
		for _, kind := range []int32{0, 7, 11, 12, 13, 14, 15, 16, 17, 33, 34, 35, 36, 60} {
			info, e := m.NewFuncDeclInfo(fid, kind, nil, ASTFuncFlags{}, false)
			astTestMust(t, e)
			d, e := m.MkFuncDecl(astTestSymbol(t, m, "plugin-uninterpreted"), nil, s, info)
			astTestMust(t, e)
			astTestMust(t, m.DelFuncDeclInfo(&info))
			astTestPin(t, m, d)
			before := m.Stats()
			m.Budget().FailAfter(0)
			got, e := m.IsConsideredUninterpreted(d)
			astTestMust(t, e)
			numeral, e := m.IsNumeralDeclaration(d)
			m.Budget().FailAfter(-1)
			astTestMust(t, e)
			want := fid == bv && kind >= 12 && kind <= 16 || fid == arith && kind >= 33 && kind <= 35
			wantNumeral := (fid == bv || fid == arith) && kind == 0
			if got != want || numeral != wantNumeral || m.Stats() != before {
				t.Fatal(fid, kind, got, want)
			}
		}
	}
}
