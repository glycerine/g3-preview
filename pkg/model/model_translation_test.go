package model_test

// Source: src/model/func_interp.cpp:328-341; src/model/model.cpp:152-182.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
)

func TestModelTranslationConstantsFunctionsUniversesAndOrder(t *testing.T) {
	from := astTestManager(t)
	to, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: from.Symbols, Numerals: from.NumeralService(), Budget: from.Budget()})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, to.Close()) })
	translation, err := ast.NewASTTranslation(from, to, true)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, translation.Close()) })

	source, err := model.NewModel(from)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, source.Close()) })
	yes, err := from.MkTrue()
	astTestMust(t, err)
	no, err := from.MkFalse()
	astTestMust(t, err)
	constant := goalTestAtom(t, from, "translated-constant")
	constantDecl, err := from.GetDecl(constant)
	astTestMust(t, err)
	astTestMust(t, source.Core().RegisterConstant(constantDecl, yes))
	f := modelTestFunctionDecl(t, from, "translated-function")
	fi, err := model.NewFunctionInterpretation(from, 1)
	astTestMust(t, err)
	astTestMust(t, fi.InsertEntry([]ast.ASTHandle{no}, yes))
	variable, err := from.MkVar(0, mustBoolSort(t, from))
	astTestMust(t, err)
	astTestMust(t, fi.SetElse(variable))
	astTestMust(t, source.Core().RegisterFunction(f, fi))
	sort, err := from.MkUninterpretedSort(astTestSymbol(t, from, "translated-sort"), nil)
	astTestMust(t, err)
	value, err := source.GetSomeValue(sort)
	astTestMust(t, err)
	astTestMust(t, source.RegisterUniverse(sort, []ast.ASTHandle{value}))

	translated, err := source.Translate(translation)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, translated.Close()) })
	translatedConstant, err := translation.Translate(constantDecl)
	astTestMust(t, err)
	translatedYes, err := translation.Translate(yes)
	astTestMust(t, err)
	got, err := translated.Core().ConstantInterpretation(translatedConstant)
	astTestMust(t, err)
	if got != translatedYes {
		t.Fatal("translated constant", got, translatedYes)
	}
	translatedF, err := translation.Translate(f)
	astTestMust(t, err)
	gotFunction, err := translated.Core().FunctionInterpretation(translatedF)
	astTestMust(t, err)
	if gotFunction == nil || gotFunction.NumEntries() != 1 || gotFunction.Arity() != 1 {
		t.Fatal("translated function graph")
	}
	entry, err := gotFunction.Entry(0)
	astTestMust(t, err)
	translatedNo, err := translation.Translate(no)
	astTestMust(t, err)
	arg, err := entry.Argument(0)
	astTestMust(t, err)
	entryResult, err := entry.Result()
	astTestMust(t, err)
	fallback, err := gotFunction.Else()
	astTestMust(t, err)
	translatedVariable, err := translation.Translate(variable)
	astTestMust(t, err)
	if arg != translatedNo || entryResult != translatedYes || fallback != translatedVariable {
		t.Fatal("translated entry or fallback", arg, entryResult, fallback)
	}
	translatedSort, err := translation.Translate(sort)
	astTestMust(t, err)
	translatedValue, err := translation.Translate(value)
	astTestMust(t, err)
	universe, err := translated.Universe(translatedSort)
	astTestMust(t, err)
	if len(universe) != 1 || universe[0] != translatedValue {
		t.Fatal("translated universe", universe)
	}
	first, err := translated.Core().Declaration(0)
	astTestMust(t, err)
	second, err := translated.Core().Declaration(1)
	astTestMust(t, err)
	if first != translatedConstant || second != translatedF {
		t.Fatal("translated declaration order", first, second)
	}
}

func TestModelCompatibilityEvaluateCompletion(t *testing.T) {
	m, md, _, _, _, config := evaluatorFixture(t)
	missing := goalTestAtom(t, m, "model-eval-missing")
	out := astTestPin(t, m, ast.ASTHandle{})
	astTestMust(t, md.Evaluate(missing, out, true, config))
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if !m.IsFalse(got) {
		t.Fatal("model::eval completion", got)
	}
}
