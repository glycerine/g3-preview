// Source: src/ast/array_decl_plugin.cpp:543-558;
// src/model/model.cpp:93-109; src/model/model_evaluator.cpp:233-253.
package model_test

import (
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestModelOriginalRecursiveArrayRangeUniverseAndCompletion(t *testing.T) {
	m, md, ev, _, _, _ := evaluatorFixture(t)
	_, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	bs, e := m.MkBoolSort()
	astTestMust(t, e)
	u, e := m.MkUninterpretedSort(astTestSymbol(t, m, "array-range-universe"), nil)
	astTestMust(t, e)
	astTestPin(t, m, u)
	ip, e := m.NewParameterInt(7)
	astTestMust(t, e)
	sp, e := m.NewParameterAST(u)
	astTestMust(t, e)
	value, e := m.MkFamilyApp(3, 0, []ast.ASTParameter{ip, sp}, nil)
	astTestMust(t, e)
	astTestPin(t, m, value)
	astTestMust(t, m.DelParameter(&ip))
	astTestMust(t, m.DelParameter(&sp))
	astTestMust(t, md.RegisterUniverse(u, []ast.ASTHandle{value}))
	// Recursive completion preserves the model range-universe override.
	inner, e := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{u}), u)
	astTestMust(t, e)
	astTestPin(t, m, inner)
	outer, e := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{bs}), inner)
	astTestMust(t, e)
	astTestPin(t, m, outer)
	expectedInner, e := m.MkConstArray(inner, value)
	astTestMust(t, e)
	astTestPin(t, m, expectedInner)
	expected, e := m.MkConstArray(outer, expectedInner)
	astTestMust(t, e)
	astTestPin(t, m, expected)
	got, e := md.GetSomeValue(outer)
	astTestMust(t, e)
	if got != expected {
		t.Fatal("array range must honor model universe", got, expected)
	}
	constant, e := m.MkConst(astTestSymbol(t, m, "completed-array-constant"), outer)
	astTestMust(t, e)
	astTestPin(t, m, constant)
	out := astTestPin(t, m, ast.ASTHandle{})
	astTestMust(t, ev.SetModelCompletion(true))
	astTestMust(t, ev.Evaluate(constant, out))
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if got != expected {
		t.Fatal("array model completion", got, expected)
	}
	d, e := m.GetDecl(constant)
	astTestMust(t, e)
	registered, e := md.Core().ConstantInterpretation(d)
	astTestMust(t, e)
	if registered != expected {
		t.Fatal("array completion was not registered")
	}
}
