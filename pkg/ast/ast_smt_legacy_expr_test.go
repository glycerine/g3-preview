package ast

import (
	"strings"
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
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

func TestLegacySMTPrinterBitVectorAndLabelLiterals(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	printer, err := NewLegacySMTPrinter(m)
	astTestMust(t, err)
	bv, err := m.BitVectorNumeralUnsigned(13, 8)
	astTestMust(t, err)
	astTestPin(t, m, bv)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	a, err := m.MkConst(astTestSymbol(t, m, "legacy-label-body"), boolSort)
	astTestMust(t, err)
	astTestPin(t, m, a)
	labelName := astTestSymbol(t, m, "legacy label")
	label, err := m.MkLabel(true, []usymbol.Symbol{labelName}, a)
	astTestMust(t, err)
	astTestPin(t, m, label)
	literal, err := m.MkLabelLit([]usymbol.Symbol{labelName})
	astTestMust(t, err)
	astTestPin(t, m, literal)
	for _, tc := range []struct {
		expr ASTHandle
		want string
	}{{bv, "(_ bv13 8)"}, {label, "(! legacy-label-body:lblpos |legacy label|)"}, {literal, "(! true :lblpos |legacy label|)"}} {
		var out strings.Builder
		astTestMust(t, printer.DisplayExpr(&out, tc.expr, nil))
		if out.String() != tc.want {
			t.Fatal("legacy SMT specialized expression", out.String(), tc.want)
		}
	}
}

func TestLegacySMTPrinterQuantifiersPatternsAndNestedVariables(t *testing.T) {
	m := astTestManager(t)
	printer, err := NewLegacySMTPrinter(m)
	astTestMust(t, err)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	bound, err := m.MkVar(0, boolSort)
	astTestMust(t, err)
	astTestPin(t, m, bound)
	function, err := m.MkFuncDecl(astTestSymbol(t, m, "legacy-p"), []ASTHandle{boolSort}, boolSort, ASTFuncDeclInfo{})
	astTestMust(t, err)
	astTestPin(t, m, function)
	application, err := m.MkApp(function, []ASTHandle{bound})
	astTestMust(t, err)
	astTestPin(t, m, application)
	pattern, err := m.MkPattern([]ASTHandle{application})
	astTestMust(t, err)
	astTestPin(t, m, pattern)
	quantified, err := m.MkQuantifier(ASTQuantifierSpec{
		Forall: true, Sorts: []ASTHandle{boolSort}, Names: []usymbol.Symbol{astTestSymbol(t, m, "legacy-x")},
		Body: application, Patterns: []ASTHandle{pattern}, QID: astTestSymbol(t, m, "legacy-qid"),
	})
	astTestMust(t, err)
	astTestPin(t, m, quantified)
	var out strings.Builder
	astTestMust(t, printer.DisplayExpr(&out, quantified, nil))
	want := "(forall ((legacy-x Bool) )(! (legacy-p legacy-x) :pattern ( (legacy-p legacy-x) ) :qid legacy-qid))\n"
	if out.String() != want {
		t.Fatalf("legacy SMT quantified expression %q, want %q", out.String(), want)
	}

	outerVariable, err := m.MkVar(1, boolSort)
	astTestMust(t, err)
	astTestPin(t, m, outerVariable)
	innerVariable, err := m.MkVar(0, boolSort)
	astTestMust(t, err)
	astTestPin(t, m, innerVariable)
	equality, err := m.MkEq(outerVariable, innerVariable)
	astTestMust(t, err)
	astTestPin(t, m, equality)
	inner, err := m.MkQuantifier(ASTQuantifierSpec{
		Forall: false, Sorts: []ASTHandle{boolSort}, Names: []usymbol.Symbol{astTestSymbol(t, m, "legacy-y")}, Body: equality,
	})
	astTestMust(t, err)
	astTestPin(t, m, inner)
	outer, err := m.MkQuantifier(ASTQuantifierSpec{
		Forall: true, Sorts: []ASTHandle{boolSort}, Names: []usymbol.Symbol{astTestSymbol(t, m, "legacy-x")}, Body: inner,
	})
	astTestMust(t, err)
	astTestPin(t, m, outer)
	out.Reset()
	astTestMust(t, printer.DisplayExpr(&out, outer, nil))
	want = "(forall ((legacy-x Bool) )(exists ((legacy-y Bool) )(= legacy-x legacy-y))\n)\n"
	if out.String() != want {
		t.Fatalf("legacy SMT nested quantifier %q, want %q", out.String(), want)
	}

	skolemHack, err := m.MkFuncDecl(astTestSymbol(t, m, "sk_hack"), []ASTHandle{boolSort}, boolSort, ASTFuncDeclInfo{})
	astTestMust(t, err)
	astTestPin(t, m, skolemHack)
	hackApplication, err := m.MkApp(skolemHack, []ASTHandle{bound})
	astTestMust(t, err)
	astTestPin(t, m, hackApplication)
	hackPattern, err := m.MkPattern([]ASTHandle{hackApplication})
	astTestMust(t, err)
	astTestPin(t, m, hackPattern)
	hackQuantifier, err := m.MkQuantifier(ASTQuantifierSpec{
		Forall: true, Sorts: []ASTHandle{boolSort}, Names: []usymbol.Symbol{astTestSymbol(t, m, "legacy-s")},
		Body: bound, Patterns: []ASTHandle{hackPattern},
	})
	astTestMust(t, err)
	astTestPin(t, m, hackQuantifier)
	out.Reset()
	astTestMust(t, printer.DisplayExpr(&out, hackQuantifier, nil))
	want = "(forall ((legacy-s Bool) )(! legacy-s))\n"
	if out.String() != want {
		t.Fatalf("legacy SMT skipped skolem-hack pattern %q, want %q", out.String(), want)
	}
}
