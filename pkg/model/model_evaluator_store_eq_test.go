package model_test

// Source: src/ast/rewriter/array_rewriter.cpp:381-417;
// src/model/model_evaluator.cpp:282-290,565-569.

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
	"testing"
)

func TestOriginalStoreEqualityExpansionIndependentBooleanModels(t *testing.T) {
	m, md, ev, c, p, config := evaluatorFixture(t)
	fid, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	key := "expand_store_eq"
	astTestMust(t, c.SetBoolName(p, &key, true))
	rw, e := rewriter.NewArrayRewriter(m, c, config.Globals, p)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, rw.Close()) })
	bs, e := m.MkBoolSort()
	astTestMust(t, e)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	s, e := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{bs}), bs)
	astTestMust(t, e)
	bd, e := m.MkFuncDecl(astTestSymbol(t, m, "store-eq-model-base"), nil, s, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	base, e := m.MkApp(bd, nil)
	astTestMust(t, e)
	astTestPin(t, m, base)
	atoms := []ast.ASTHandle{goalTestAtom(t, m, "store-eq-x"), goalTestAtom(t, m, "store-eq-y"), goalTestAtom(t, m, "store-eq-u"), goalTestAtom(t, m, "store-eq-v")}
	declarations := make([]ast.ASTHandle, 4)
	for i, h := range atoms {
		declarations[i], e = m.GetDecl(h)
		astTestMust(t, e)
	}
	lhs, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{base, atoms[0], atoms[2]})
	astTestMust(t, e)
	astTestPin(t, m, lhs)
	rhs, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{base, atoms[1], atoms[3]})
	astTestMust(t, e)
	astTestPin(t, m, rhs)
	original, e := m.MkEq(lhs, rhs)
	astTestMust(t, e)
	astTestPin(t, m, original)
	expanded := astTestPin(t, m, ast.ASTHandle{})
	status, e := rw.MkEqCore(lhs, rhs, expanded)
	astTestMust(t, e)
	if status != rewriter.RewriteFull {
		t.Fatal("store equality expansion status", status)
	}
	formula, e := m.RefGet(expanded)
	astTestMust(t, e)
	out := astTestPin(t, m, ast.ASTHandle{})
	// Independent two-element array semantics, across both common defaults and
	// all four Boolean index/value assignments. No reducer builds the oracle.
	for bits := 0; bits < 32; bits++ {
		common := bits&16 != 0
		defaultValue := no
		if common {
			defaultValue = yes
		}
		constant, e := m.MkConstArray(s, defaultValue)
		astTestMust(t, e)
		astTestPin(t, m, constant)
		astTestMust(t, md.Core().RegisterConstant(bd, constant))
		for i, d := range declarations {
			value := no
			if bits&(1<<i) != 0 {
				value = yes
			}
			astTestMust(t, md.Core().RegisterConstant(d, value))
		}
		x, y, u, v := bits&1 != 0, bits&2 != 0, bits&4 != 0, bits&8 != 0
		equal := true
		for _, index := range []bool{false, true} {
			left, right := common, common
			if index == x {
				left = u
			}
			if index == y {
				right = v
			}
			if left != right {
				equal = false
			}
		}
		want := no
		if equal {
			want = yes
		}
		for _, input := range []ast.ASTHandle{original, formula} {
			astTestMust(t, ev.Reset(nil))
			astTestMust(t, ev.Evaluate(input, out))
			got, e := m.RefGet(out)
			astTestMust(t, e)
			if got != want {
				t.Fatal("independent store equality model", bits, input, got, want)
			}
		}
	}
}
