// Source: src/ast/bv_decl_plugin.cpp:746-751;
// src/model/model.cpp:93-109; src/model/model_evaluator.cpp:233-253.
package model_test

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestModelOriginalBitVectorZeroCompletion(t *testing.T) {
	m, md, ev, _, _, _ := evaluatorFixture(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(65)
	astTestMust(t, e)
	constant, e := m.MkConst(astTestSymbol(t, m, "completed-bv-constant"), s)
	astTestMust(t, e)
	astTestPin(t, m, constant)
	out := astTestPin(t, m, ast.ASTHandle{})
	astTestMust(t, ev.SetModelCompletion(true))
	astTestMust(t, ev.Evaluate(constant, out))
	got, e := m.RefGet(out)
	astTestMust(t, e)
	expected, e := m.GetSomeValue(s)
	astTestMust(t, e)
	valueKind, e := m.IsValue(got)
	astTestMust(t, e)
	if got != expected || !valueKind {
		t.Fatal("completed bit-vector must use the plugin numeral", got, expected, valueKind)
	}
	d, e := m.GetDecl(constant)
	astTestMust(t, e)
	registered, e := md.Core().ConstantInterpretation(d)
	astTestMust(t, e)
	if registered != got {
		t.Fatal("numeric completion was not registered")
	}
	_, e = m.RegisterArrayPlugin()
	astTestMust(t, e)
	bs, e := m.MkBoolSort()
	astTestMust(t, e)
	as, e := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{bs}), s)
	astTestMust(t, e)
	want, e := m.MkConstArray(as, got)
	astTestMust(t, e)
	astTestPin(t, m, want)
	value, e := md.GetSomeValue(as)
	astTestMust(t, e)
	if value != want {
		t.Fatal("recursive array numeric range completion")
	}
}
