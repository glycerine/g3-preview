package model_test

// Source: src/ast/arith_decl_plugin.cpp:627-630;
// src/model/model.cpp:93-109; src/model/model_evaluator.cpp:233-253.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestModelOriginalArithmeticZeroCompletion(t *testing.T) {
	m, md, ev, _, _, _ := evaluatorFixture(t)
	fid, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	astTestMust(t, ev.SetModelCompletion(true))
	for _, kind := range []int32{0, 1} {
		s, e := m.MkFamilySort(fid, kind, nil)
		astTestMust(t, e)
		constant, e := m.MkConst(astTestSymbol(t, m, "completed-arithmetic-constant"), s)
		astTestMust(t, e)
		astTestPin(t, m, constant)
		out := astTestPin(t, m, ast.ASTHandle{})
		astTestMust(t, ev.Evaluate(constant, out))
		got, e := m.RefGet(out)
		astTestMust(t, e)
		expected, e := m.GetSomeValue(s)
		astTestMust(t, e)
		value, e := m.IsValue(got)
		astTestMust(t, e)
		if got != expected || !value {
			t.Fatal("numeric completion must use source zero numeral", kind, got, expected, value)
		}
		d, e := m.GetDecl(constant)
		astTestMust(t, e)
		registered, e := md.Core().ConstantInterpretation(d)
		astTestMust(t, e)
		if registered != got {
			t.Fatal("numeric completion was not registered", kind)
		}
	}
}
