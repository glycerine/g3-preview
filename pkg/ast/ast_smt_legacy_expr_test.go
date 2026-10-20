package ast

import (
	"strings"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
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

func TestLegacySMTPrinterDeclarationParametersAndPrivateConstants(t *testing.T) {
	m := astTestManager(t)
	printer, err := NewLegacySMTPrinter(m)
	astTestMust(t, err)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	a, err := m.MkConst(astTestSymbol(t, m, "legacy-a"), boolSort)
	astTestMust(t, err)
	astTestPin(t, m, a)
	nested, err := m.MkFuncDecl(astTestSymbol(t, m, "legacy-nested"), nil, boolSort, ASTFuncDeclInfo{})
	astTestMust(t, err)
	astTestPin(t, m, nested)

	integer, err := m.NewParameterInt(-7)
	astTestMust(t, err)
	symbol, err := m.NewParameterString("legacy index")
	astTestMust(t, err)
	rationalValue, err := m.numerals.NewValue()
	astTestMust(t, err)
	astTestMust(t, m.numerals.SetString(&rationalValue, "7/3"))
	rational, err := m.NewParameterRational(&rationalValue)
	astTestMust(t, err)
	astTestMust(t, m.numerals.Del(&rationalValue))
	double, err := m.NewParameterDouble(1.25)
	astTestMust(t, err)
	external, err := m.NewParameterExternal(9)
	astTestMust(t, err)
	sortParameter, err := m.NewParameterAST(boolSort)
	astTestMust(t, err)
	expressionParameter, err := m.NewParameterAST(a)
	astTestMust(t, err)
	declarationParameter, err := m.NewParameterAST(nested)
	astTestMust(t, err)
	parameters := []ASTParameter{integer, symbol, rational, double, external, sortParameter, expressionParameter, declarationParameter}
	t.Cleanup(func() {
		for i := range parameters {
			astTestMust(t, m.DelParameter(&parameters[i]))
		}
	})

	info, err := m.NewFuncDeclInfo(NullFamilyID, 701, parameters, ASTFuncFlags{}, false)
	astTestMust(t, err)
	decl, err := m.MkFuncDeclWithInfo(astTestSymbol(t, m, "legacy-params"), nil, boolSort, info)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelFuncDeclInfo(&info)) })
	constant, err := m.MkApp(decl, nil)
	astTestMust(t, err)
	astTestPin(t, m, constant)

	typedInfo, err := m.NewFuncDeclInfo(NullFamilyID, 702, []ASTParameter{sortParameter}, ASTFuncFlags{}, false)
	astTestMust(t, err)
	typedDecl, err := m.MkFuncDeclWithInfo(astTestSymbol(t, m, "legacy-typed"), nil, boolSort, typedInfo)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelFuncDeclInfo(&typedInfo)) })
	typed, err := m.MkApp(typedDecl, nil)
	astTestMust(t, err)
	astTestPin(t, m, typed)

	privateParameter, err := m.NewParameterInt(4)
	astTestMust(t, err)
	parameters = append(parameters, privateParameter)
	privateInfo, err := m.NewFuncDeclInfo(NullFamilyID, 703, []ASTParameter{privateParameter}, ASTFuncFlags{}, true)
	astTestMust(t, err)
	privateDecl, err := m.MkFuncDeclWithInfo(astTestSymbol(t, m, "legacy-private"), nil, boolSort, privateInfo)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelFuncDeclInfo(&privateInfo)) })
	privateConstant, err := m.MkApp(privateDecl, nil)
	astTestMust(t, err)
	astTestPin(t, m, privateConstant)
	privateAppInfo, err := m.NewFuncDeclInfo(NullFamilyID, 704, []ASTParameter{privateParameter}, ASTFuncFlags{}, true)
	astTestMust(t, err)
	privateAppDecl, err := m.MkFuncDeclWithInfo(astTestSymbol(t, m, "legacy-private-app"), []ASTHandle{boolSort}, boolSort, privateAppInfo)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelFuncDeclInfo(&privateAppInfo)) })
	privateApplication, err := m.MkApp(privateAppDecl, []ASTHandle{a})
	astTestMust(t, err)
	astTestPin(t, m, privateApplication)

	for _, tc := range []struct {
		expr ASTHandle
		want string
	}{
		{constant, "(_ legacy-params -7 legacy index 7/3 1.25 @9 Bool legacy-a legacy-nested )"},
		{typed, "(as legacy-typed Bool)"},
		{privateConstant, "legacy-private"},
		{privateApplication, "((_ legacy-private-app 4) legacy-a)"},
	} {
		var out strings.Builder
		astTestMust(t, printer.DisplayExpr(&out, tc.expr, nil))
		if out.String() != tc.want {
			t.Fatalf("legacy SMT declaration parameters %q, want %q", out.String(), tc.want)
		}
	}
}

func TestLegacySMTPrinterBit2BoolExpansion(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	family, err := m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	printer, err := NewLegacySMTPrinter(m)
	astTestMust(t, err)
	sort, err := m.BitVectorSort(8)
	astTestMust(t, err)
	value, err := m.MkConst(astTestSymbol(t, m, "legacy-bv"), sort)
	astTestMust(t, err)
	astTestPin(t, m, value)
	index, err := m.NewParameterInt(3)
	astTestMust(t, err)
	bit, err := m.MkFamilyApp(family, 55, []ASTParameter{index}, []ASTHandle{value})
	astTestMust(t, err)
	astTestMust(t, m.DelParameter(&index))
	astTestPin(t, m, bit)
	var out strings.Builder
	astTestMust(t, printer.DisplayExpr(&out, bit, nil))
	want := "(= ((_ extract 3 3) legacy-bv) (_ bv1 1))"
	if out.String() != want {
		t.Fatalf("legacy SMT bit2bool expression %q, want %q", out.String(), want)
	}
}

func TestLegacySMTPrinterDistinctGroupsBySort(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	printer, err := NewLegacySMTPrinter(m)
	astTestMust(t, err)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	bvSort, err := m.BitVectorSort(8)
	astTestMust(t, err)
	intSort := m.external(m.arithmeticPlugin.fields[1])
	constant := func(name string, sort ASTHandle) ASTHandle {
		h, e := m.MkConst(astTestSymbol(t, m, name), sort)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	a := constant("legacy-da", boolSort)
	x := constant("legacy-dx", bvSort)
	i := constant("legacy-di", intSort)
	b := constant("legacy-db", boolSort)
	y := constant("legacy-dy", bvSort)
	makeDistinct := func(args []ASTHandle) ASTHandle {
		domain := make([]ASTHandle, len(args))
		for j := range args {
			domain[j], err = m.GetSort(args[j])
			astTestMust(t, err)
		}
		info, e := m.NewFuncDeclInfo(m.BasicFamilyID(), ASTOpDistinct, nil, ASTFuncFlags{}, false)
		astTestMust(t, e)
		decl, e := m.MkFuncDeclWithInfo(astTestSymbol(t, m, "distinct"), domain, boolSort, info)
		astTestMust(t, e)
		astTestMust(t, m.DelFuncDeclInfo(&info))
		result, e := m.MkApp(decl, args)
		astTestMust(t, e)
		astTestPin(t, m, result)
		return result
	}
	distinct := makeDistinct([]ASTHandle{a, x, i, b, y})
	var out strings.Builder
	astTestMust(t, printer.DisplayExpr(&out, distinct, nil))
	want := "(and (distinct legacy-da legacy-db) (distinct legacy-dx legacy-dy) true)"
	if out.String() != want {
		t.Fatalf("legacy SMT heterogeneous distinct %q, want %q", out.String(), want)
	}

	singletons := makeDistinct([]ASTHandle{a, x, i})
	out.Reset()
	astTestMust(t, printer.DisplayExpr(&out, singletons, nil))
	if out.String() != "(and true)" {
		t.Fatalf("legacy SMT singleton-sort distinct %q", out.String())
	}
}

func TestLegacySMTPrinterImplicationChainSimplification(t *testing.T) {
	m := astTestManager(t)
	printer, err := NewLegacySMTPrinter(m)
	astTestMust(t, err)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	constant := func(name string) ASTHandle {
		h, e := m.MkConst(astTestSymbol(t, m, name), boolSort)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	a, b, c, d := constant("legacy-ia"), constant("legacy-ib"), constant("legacy-ic"), constant("legacy-id")
	inner, err := m.MkImplies(c, d)
	astTestMust(t, err)
	astTestPin(t, m, inner)
	middle, err := m.MkImplies(b, inner)
	astTestMust(t, err)
	astTestPin(t, m, middle)
	outer, err := m.MkImplies(a, middle)
	astTestMust(t, err)
	astTestPin(t, m, outer)
	var out strings.Builder
	astTestMust(t, printer.DisplayExpr(&out, outer, nil))
	if out.String() != "(=> (and legacy-ia legacy-ib legacy-ic) legacy-id)" {
		t.Fatalf("legacy SMT simplified implication chain %q", out.String())
	}
	astTestMust(t, printer.SetSimplifyImplies(false))
	out.Reset()
	astTestMust(t, printer.DisplayExpr(&out, outer, nil))
	if out.String() != "(=> legacy-ia (=> legacy-ib (=> legacy-ic legacy-id)))" {
		t.Fatalf("legacy SMT unsimplified implication chain %q", out.String())
	}
	astTestMust(t, printer.SetSimplifyImplies(true))
	single, err := m.MkImplies(a, b)
	astTestMust(t, err)
	astTestPin(t, m, single)
	out.Reset()
	astTestMust(t, printer.DisplayExpr(&out, single, nil))
	if out.String() != "(=> legacy-ia legacy-ib)" {
		t.Fatalf("legacy SMT single implication %q", out.String())
	}
}

func TestLegacySMTPrinterFloatingPointNumerals(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	printer, err := NewLegacySMTPrinter(m)
	astTestMust(t, err)
	floatSort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	var value unumeral.ArbitraryFloat
	astTestMust(t, m.floatingPointPlugin.numeric.SetDouble(&value, 8, 24, 1.5))
	finiteRef, err := m.floatingPointPlugin.mkNumeral(&value, nil)
	astTestMust(t, err)
	astTestMust(t, m.floatingPointPlugin.numeric.Del(&value))
	finite := m.external(finiteRef)
	astTestPin(t, m, finite)
	nan, err := m.MkFPANaNOfSort(floatSort)
	astTestMust(t, err)
	astTestPin(t, m, nan)
	for _, tc := range []struct {
		expr ASTHandle
		want string
	}{
		{finite, "((_ to_fp 8 24) RTZ 1.5 0)"},
		{nan, "((_ to_fp 8 24) RTZ NaN)"},
	} {
		var out strings.Builder
		astTestMust(t, printer.DisplayExpr(&out, tc.expr, nil))
		if out.String() != tc.want || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatalf("legacy SMT floating-point numeral %q, want %q", out.String(), tc.want)
		}
	}
}
