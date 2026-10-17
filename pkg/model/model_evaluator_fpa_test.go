// Source: src/model/model_evaluator.cpp:149-174;
// src/ast/rewriter/fpa_rewriter.cpp:547-667.
package model_test

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestModelEvaluatorFPAEqualityAndApplicationReduction(t *testing.T) {
	m, _, ev, _, _, _ := evaluatorFixture(t)
	prior, err := m.ExistingFloatingPointFamilyID()
	astTestMust(t, err)
	if prior != ast.NullFamilyID {
		t.Fatal("floating-point plugin unexpectedly registered")
	}
	plainDecl := modelTestFunctionDecl(t, m, "eval-before-fpa")
	plainArg, err := m.MkTrue()
	astTestMust(t, err)
	plainApp, err := m.MkApp(plainDecl, []ast.ASTHandle{plainArg})
	astTestMust(t, err)
	astTestPin(t, m, plainApp)
	out := astTestPin(t, m, ast.ASTHandle{})
	astTestMust(t, ev.Evaluate(plainApp, out))
	plainResult, err := m.RefGet(out)
	astTestMust(t, err)
	if plainResult != plainApp {
		t.Fatal("unregistered FPA family intercepted uninterpreted application")
	}
	_, err = m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	gotFamily, err := m.ExistingFloatingPointFamilyID()
	astTestMust(t, err)
	if gotFamily != family {
		t.Fatal("registered floating-point family", gotFamily, family)
	}
	sort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	astTestPin(t, m, sort)
	nan, err := m.MkFPANaNOfSort(sort)
	astTestMust(t, err)
	astTestPin(t, m, nan)
	positive, err := m.MkFPAInfinityOfSort(sort, false)
	astTestMust(t, err)
	astTestPin(t, m, positive)
	negative, err := m.MkFPAInfinityOfSort(sort, true)
	astTestMust(t, err)
	astTestPin(t, m, negative)
	positiveZero, err := m.MkFPAZeroOfSort(sort, false)
	astTestMust(t, err)
	astTestPin(t, m, positiveZero)
	negativeZero, err := m.MkFPAZeroOfSort(sort, true)
	astTestMust(t, err)
	astTestPin(t, m, negativeZero)
	yes, err := m.MkTrue()
	astTestMust(t, err)
	no, err := m.MkFalse()
	astTestMust(t, err)
	check := func(name string, term, want ast.ASTHandle) {
		t.Helper()
		astTestPin(t, m, term)
		astTestMust(t, ev.Evaluate(term, out))
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if got != want {
			t.Fatalf("%s: got %v, want %v", name, got, want)
		}
	}
	smtEq, err := m.MkEq(nan, nan)
	astTestMust(t, err)
	check("SMT NaN equality", smtEq, yes)
	ieeeEq, err := m.MkFamilyApp(family, ast.FPAOpEq, nil, []ast.ASTHandle{nan, nan})
	astTestMust(t, err)
	check("IEEE NaN equality", ieeeEq, no)
	neg, err := m.MkFamilyApp(family, ast.FPAOpNeg, nil, []ast.ASTHandle{positive})
	astTestMust(t, err)
	check("negated infinity", neg, negative)
	unequal, err := m.MkEq(positive, negative)
	astTestMust(t, err)
	check("SMT unequal infinities", unequal, no)
	signedZeroEq, err := m.MkEq(positiveZero, negativeZero)
	astTestMust(t, err)
	check("SMT signed-zero inequality", signedZeroEq, no)
	ieeeZeroEq, err := m.MkFamilyApp(family, ast.FPAOpEq, nil, []ast.ASTHandle{positiveZero, negativeZero})
	astTestMust(t, err)
	check("IEEE signed-zero equality", ieeeZeroEq, yes)
	astTestMust(t, ev.Cleanup(nil))
	check("FPA rewriter after cleanup", ieeeEq, no)
}

func TestModelEvaluatorFPACompletionUsesPluginValues(t *testing.T) {
	m, _, ev, _, _, _ := evaluatorFixture(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	floatSort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	rmSort, err := m.MkFamilySort(family, 1, nil)
	astTestMust(t, err)
	floatConst, err := m.MkConst(astTestSymbol(t, m, "eval-fpa-completion"), floatSort)
	astTestMust(t, err)
	astTestPin(t, m, floatConst)
	rmConst, err := m.MkConst(astTestSymbol(t, m, "eval-rm-completion"), rmSort)
	astTestMust(t, err)
	astTestPin(t, m, rmConst)
	wantFloat, err := m.MkFPANaNOfSort(floatSort)
	astTestMust(t, err)
	wantRM, err := m.MkFamilyApp(family, ast.FPAOpRMTowardZero, nil, nil)
	astTestMust(t, err)
	astTestMust(t, ev.SetModelCompletion(true))
	out := astTestPin(t, m, ast.ASTHandle{})
	for _, tc := range []struct {
		name  string
		value ast.ASTHandle
		want  ast.ASTHandle
	}{
		{"FloatingPoint", floatConst, wantFloat},
		{"RoundingMode", rmConst, wantRM},
	} {
		astTestMust(t, ev.Evaluate(tc.value, out))
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if got != tc.want {
			t.Fatalf("%s completion: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
