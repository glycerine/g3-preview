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
	astTestMust(t, ev.Cleanup(nil))
	check("FPA rewriter after cleanup", ieeeEq, no)
}
