package ast

import (
	"strings"
	"testing"
)

func TestLegacySMTPrinterBasicAppsVariablesAndArithmeticNumerals(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	printer, err := NewLegacySMTPrinter(m)
	astTestMust(t, err)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	a, err := m.MkConst(astTestSymbol(t, m, "legacy-a"), boolSort)
	astTestMust(t, err)
	astTestPin(t, m, a)
	b, err := m.MkConst(astTestSymbol(t, m, "legacy-b"), boolSort)
	astTestMust(t, err)
	astTestPin(t, m, b)
	and, err := m.MkAnd(a, b)
	astTestMust(t, err)
	astTestPin(t, m, and)
	iff, err := m.MkIff(a, b)
	astTestMust(t, err)
	astTestPin(t, m, iff)
	implies, err := m.MkImplies(a, b)
	astTestMust(t, err)
	astTestPin(t, m, implies)
	ite, err := m.MkIte(a, and, iff)
	astTestMust(t, err)
	astTestPin(t, m, ite)
	integerSort := m.external(m.arithmeticPlugin.fields[1])
	realSort := m.external(m.arithmeticPlugin.fields[0])
	numeral := func(text string, integer bool) ASTHandle {
		value, e := m.numerals.NewValue()
		astTestMust(t, e)
		astTestMust(t, m.numerals.SetString(&value, text))
		h, e := m.ArithmeticNumeral(&value, integer)
		astTestMust(t, e)
		astTestMust(t, m.numerals.Del(&value))
		astTestPin(t, m, h)
		return h
	}
	minusThree, realTwo, half := numeral("-3", true), numeral("2", false), numeral("1/2", false)
	x, err := m.MkVar(0, integerSort)
	astTestMust(t, err)
	astTestPin(t, m, x)
	function, err := m.MkFuncDecl(astTestSymbol(t, m, "legacy function"), []ASTHandle{integerSort}, realSort, ASTFuncDeclInfo{})
	astTestMust(t, err)
	astTestPin(t, m, function)
	application, err := m.MkApp(function, []ASTHandle{x})
	astTestMust(t, err)
	astTestPin(t, m, application)
	for _, tc := range []struct {
		expr  ASTHandle
		names []string
		want  string
	}{{and, nil, "(and legacy-a legacy-b)"}, {iff, nil, "(= legacy-a legacy-b)"}, {implies, nil, "(=> legacy-a legacy-b)"}, {ite, nil, "(ite legacy-a (and legacy-a legacy-b) (= legacy-a legacy-b))"}, {minusThree, nil, "(- 3)"}, {realTwo, nil, "2.0"}, {half, nil, "(/ 1.0 2.0)"}, {application, []string{"bound-x"}, "(|legacy function| bound-x)"}} {
		var out strings.Builder
		astTestMust(t, printer.DisplayExpr(&out, tc.expr, tc.names))
		if out.String() != tc.want {
			t.Fatal("legacy SMT expression", out.String(), tc.want)
		}
	}
}
