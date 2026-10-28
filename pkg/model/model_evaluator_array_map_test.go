package model_test

// Source: src/model/model_evaluator.cpp:183-196;
// src/ast/rewriter/array_rewriter.cpp:220-323.

import (
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestModelEvaluatorOriginalArrayMapTraversalAndSelection(t *testing.T) {
	m, _, ev, _, _, _ := evaluatorFixture(t)
	fid, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	bs, e := m.MkBoolSort()
	astTestMust(t, e)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	bp, e := m.NewParameterAST(bs)
	astTestMust(t, e)
	sort, e := m.MkFamilySort(fid, 0, []ast.ASTParameter{bp, bp})
	astTestMust(t, e)
	astTestMust(t, m.DelParameter(&bp))
	sp, e := m.NewParameterAST(sort)
	astTestMust(t, e)
	base, e := m.MkFamilyApp(fid, ast.ASTArrayOpConst, []ast.ASTParameter{sp}, []ast.ASTHandle{yes})
	astTestMust(t, e)
	astTestPin(t, m, base)
	astTestMust(t, m.DelParameter(&sp))
	stored, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{base, yes, no})
	astTestMust(t, e)
	astTestPin(t, m, stored)
	not, e := m.MkNotDecl()
	astTestMust(t, e)
	mapped, e := m.MkArrayMap(not, ast.SliceArguments([]ast.ASTHandle{stored}))
	astTestMust(t, e)
	astTestPin(t, m, mapped)
	out := astTestPin(t, m, ast.ASTHandle{})
	for _, index := range []ast.ASTHandle{yes, no} {
		selected, e := m.MkFamilyApp(fid, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{mapped, index})
		astTestMust(t, e)
		astTestPin(t, m, selected)
		astTestMust(t, ev.Reset(nil))
		astTestMust(t, ev.Evaluate(selected, out))
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if got != index {
			t.Fatal("mapped selection", index, got)
		}
	}
	astTestMust(t, ev.Reset(nil))
	astTestMust(t, ev.Evaluate(mapped, out))
	value, e := m.RefGet(out)
	astTestMust(t, e)
	op, e := m.GetDeclKind(value)
	astTestMust(t, e)
	if op != ast.ASTArrayOpStore {
		t.Fatal("map traversal did not produce store")
	}
	result, e := m.GetArg(value, 2)
	astTestMust(t, e)
	if result != yes {
		t.Fatal("mapped value was not Boolean-reduced")
	}
}
