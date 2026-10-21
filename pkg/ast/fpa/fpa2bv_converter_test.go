package fpa_test

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/fpa"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestConverterVariablesAndIEEEBitVectorConversions(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	variable, err := c.MkVar(7, sort)
	must(t, err)
	sign, exponent, significand, err := c.SplitFP(variable)
	must(t, err)
	if index, e := m.GetVarIndex(sign); e != nil || index != 7 {
		t.Fatal("sign variable", index, e)
	}
	if index, e := m.GetVarIndex(significand); e != nil || index != 8 {
		t.Fatal("significand variable", index, e)
	}
	if index, e := m.GetVarIndex(exponent); e != nil || index != 9 {
		t.Fatal("exponent variable", index, e)
	}

	packedSort, err := m.BitVectorSort(32)
	must(t, err)
	packed, err := m.MkVar(10, packedSort)
	must(t, err)
	converted, err := c.FromIEEEBV(sort, packed)
	must(t, err)
	sign, exponent, significand, err = c.SplitFP(converted)
	must(t, err)
	wantSign, err := m.MkBitVectorExtract(31, 31, packed)
	must(t, err)
	wantExponent, err := m.MkBitVectorExtract(30, 23, packed)
	must(t, err)
	wantSignificand, err := m.MkBitVectorExtract(22, 0, packed)
	must(t, err)
	if sign != wantSign || exponent != wantExponent || significand != wantSignificand {
		t.Fatal("IEEE split ranges", sign, exponent, significand)
	}

	canonicalNaN, err := c.MkSpecial(sort, fpa.NaN)
	must(t, err)
	one, err := m.BitVectorNumeralUnsigned(1, 1)
	must(t, err)
	top, err := m.BitVectorNumeralAllOnes(8)
	must(t, err)
	payload, err := m.BitVectorNumeralUnsigned(17, 23)
	must(t, err)
	alternateNaN, err := c.MkFP(one, top, payload)
	must(t, err)
	got, err := c.ToIEEEBV(alternateNaN)
	must(t, err)
	want, err := c.JoinFP(canonicalNaN)
	must(t, err)
	if got != want {
		t.Fatal("high-unspecified NaN canonicalization", got, want)
	}
	must(t, c.SetHighUnspecified(false))
	if _, err = c.ToIEEEBV(alternateNaN); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("low-unspecified UF branch must remain explicit", err)
	}
}

func TestConverterFloatFormatConversionSpecialCasesAndSymbolicShape(t *testing.T) {
	m, c := fixture(t)
	sourceSort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	narrowSort, err := m.FloatingPointSort(5, 11)
	must(t, err)
	wideSort, err := m.FloatingPointSort(11, 53)
	must(t, err)
	rne, err := c.MkRoundingMode(ast.FPAOpRMNearestEven)
	must(t, err)

	for _, kind := range []fpa.SpecialValue{
		fpa.NaN, fpa.PositiveZero, fpa.NegativeZero,
		fpa.PositiveInfinity, fpa.NegativeInfinity,
	} {
		input, e := c.MkSpecial(sourceSort, kind)
		must(t, e)
		got, e := c.ToFPFloat(narrowSort, rne, input)
		must(t, e)
		want, e := c.MkSpecial(narrowSort, kind)
		must(t, e)
		if got != want {
			t.Fatal("format conversion special", kind, got, want)
		}
	}

	signSort, err := m.BitVectorSort(1)
	must(t, err)
	exponentSort, err := m.BitVectorSort(8)
	must(t, err)
	significandSort, err := m.BitVectorSort(23)
	must(t, err)
	sign, err := m.MkVar(810, signSort)
	must(t, err)
	exponent, err := m.MkVar(811, exponentSort)
	must(t, err)
	significand, err := m.MkVar(812, significandSort)
	must(t, err)
	input, err := c.MkFP(sign, exponent, significand)
	must(t, err)
	for _, target := range []ast.ASTHandle{narrowSort, wideSort} {
		got, e := c.ToFPFloat(target, rne, input)
		must(t, e)
		gotSort, e := m.GetSort(got)
		must(t, e)
		kind, e := m.GetDeclKind(got)
		must(t, e)
		if gotSort != target || kind != ast.FPAOpFP {
			t.Fatal("symbolic format conversion shape", gotSort, target, kind)
		}
	}
	identity, err := c.ToFPFloat(sourceSort, rne, input)
	must(t, err)
	if identity != input {
		t.Fatal("same-format conversion did not preserve the input", identity, input)
	}
}

func TestConverterSignedUnsignedIntegerConversions(t *testing.T) {
	m, c := fixture(t)
	targetSort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	rne, err := c.MkRoundingMode(ast.FPAOpRMNearestEven)
	must(t, err)
	zero, err := m.BitVectorNumeralUnsigned(0, 32)
	must(t, err)
	positiveZero, err := c.MkSpecial(targetSort, fpa.PositiveZero)
	must(t, err)
	for _, conversion := range []func(ast.ASTHandle, ast.ASTHandle, ast.ASTHandle) (ast.ASTHandle, error){
		c.ToFPSigned, c.ToFPUnsigned,
	} {
		got, e := conversion(targetSort, rne, zero)
		must(t, e)
		if got != positiveZero {
			t.Fatal("integer zero conversion", got, positiveZero)
		}
	}

	one, err := m.BitVectorNumeralUnsigned(1, 32)
	must(t, err)
	for _, conversion := range []func(ast.ASTHandle, ast.ASTHandle, ast.ASTHandle) (ast.ASTHandle, error){
		c.ToFPSigned, c.ToFPUnsigned,
	} {
		got, e := conversion(targetSort, rne, one)
		must(t, e)
		gotSort, e := m.GetSort(got)
		must(t, e)
		kind, e := m.GetDeclKind(got)
		must(t, e)
		if gotSort != targetSort || kind != ast.FPAOpFP {
			t.Fatal("integer one conversion shape", gotSort, kind)
		}
	}

	minusOneBits, err := m.BitVectorNumeralAllOnes(32)
	must(t, err)
	got, err := c.ToFPSigned(targetSort, rne, minusOneBits)
	must(t, err)
	gotSort, err := m.GetSort(got)
	must(t, err)
	if kind, e := m.GetDeclKind(got); e != nil || gotSort != targetSort || kind != ast.FPAOpFP {
		t.Fatal("signed minus-one conversion shape", gotSort, kind, e)
	}

	bvSort, err := m.BitVectorSort(32)
	must(t, err)
	variable, err := m.MkVar(820, bvSort)
	must(t, err)
	for _, conversion := range []func(ast.ASTHandle, ast.ASTHandle, ast.ASTHandle) (ast.ASTHandle, error){
		c.ToFPSigned, c.ToFPUnsigned,
	} {
		got, e := conversion(targetSort, rne, variable)
		must(t, e)
		gotSort, e := m.GetSort(got)
		must(t, e)
		kind, e := m.GetDeclKind(got)
		must(t, e)
		if gotSort != targetSort || kind != ast.FPAOpFP {
			t.Fatal("symbolic integer conversion shape", gotSort, kind)
		}
	}
}

func TestConverterFloatToBitVectorConversions(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	rne, err := c.MkRoundingMode(ast.FPAOpRMNearestEven)
	must(t, err)
	wantZero, err := m.BitVectorNumeralUnsigned(0, 8)
	must(t, err)
	for _, special := range []fpa.SpecialValue{
		fpa.NaN, fpa.PositiveInfinity, fpa.NegativeInfinity,
		fpa.PositiveZero, fpa.NegativeZero,
	} {
		input, e := c.MkSpecial(sort, special)
		must(t, e)
		for _, conversion := range []func(uint32, ast.ASTHandle, ast.ASTHandle) (ast.ASTHandle, error){
			c.ToUBV, c.ToSBV,
		} {
			got, e := conversion(8, rne, input)
			must(t, e)
			if got != wantZero {
				t.Fatal("float-to-BV special", special, got, wantZero)
			}
		}
	}

	signSort, err := m.BitVectorSort(1)
	must(t, err)
	exponentSort, err := m.BitVectorSort(8)
	must(t, err)
	significandSort, err := m.BitVectorSort(23)
	must(t, err)
	sign, err := m.MkVar(830, signSort)
	must(t, err)
	exponent, err := m.MkVar(831, exponentSort)
	must(t, err)
	significand, err := m.MkVar(832, significandSort)
	must(t, err)
	input, err := c.MkFP(sign, exponent, significand)
	must(t, err)
	for _, conversion := range []func(uint32, ast.ASTHandle, ast.ASTHandle) (ast.ASTHandle, error){
		c.ToUBV, c.ToSBV,
	} {
		got, e := conversion(17, rne, input)
		must(t, e)
		if width, e := m.BitVectorSize(got); e != nil || width != 17 {
			t.Fatal("symbolic float-to-BV width", width, e)
		}
	}
	must(t, c.SetHighUnspecified(false))
	if _, err = c.ToUBV(8, rne, input); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("low-unspecified float-to-BV branch must remain explicit", err)
	}
}

func TestConverterFloatToRealConversion(t *testing.T) {
	m, c := fixture(t)
	_, err := m.RegisterArithmeticPlugin()
	must(t, err)
	sort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	for _, special := range []fpa.SpecialValue{
		fpa.NaN, fpa.PositiveInfinity, fpa.NegativeInfinity,
		fpa.PositiveZero, fpa.NegativeZero,
	} {
		input, e := c.MkSpecial(sort, special)
		must(t, e)
		got, e := c.ToReal(input)
		must(t, e)
		text, integer, matched, e := m.ArithmeticNumeralText(got)
		must(t, e)
		if !matched || integer || text != "0" {
			t.Fatal("float-to-real special", special, text, integer, matched)
		}
	}

	signSort, err := m.BitVectorSort(1)
	must(t, err)
	exponentSort, err := m.BitVectorSort(8)
	must(t, err)
	significandSort, err := m.BitVectorSort(23)
	must(t, err)
	sign, err := m.MkVar(840, signSort)
	must(t, err)
	exponent, err := m.MkVar(841, exponentSort)
	must(t, err)
	significand, err := m.MkVar(842, significandSort)
	must(t, err)
	input, err := c.MkFP(sign, exponent, significand)
	must(t, err)
	got, err := c.ToReal(input)
	must(t, err)
	gotSort, err := m.GetSort(got)
	must(t, err)
	zero, err := m.ArithmeticRealNumeralText("0")
	must(t, err)
	realSort, err := m.GetSort(zero)
	must(t, err)
	if gotSort != realSort {
		t.Fatal("symbolic float-to-real sort", gotSort, realSort)
	}
	must(t, c.SetHighUnspecified(false))
	if _, err = c.ToReal(input); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("low-unspecified float-to-real branch must remain explicit", err)
	}
}

func TestConverterFiniteNumeral(t *testing.T) {
	m, c := fixture(t)
	_, err := m.RegisterArithmeticPlugin()
	must(t, err)
	_, err = m.RegisterBitVectorPlugin()
	must(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	must(t, err)
	sign, err := m.BitVectorNumeralUnsigned(1, 1)
	must(t, err)
	exponent, err := m.BitVectorNumeralUnsigned(127, 8)
	must(t, err)
	significand, err := m.BitVectorNumeralUnsigned(0, 23)
	must(t, err)
	components, err := m.MkFamilyApp(family, ast.FPAOpFP, nil, []ast.ASTHandle{sign, exponent, significand})
	must(t, err)
	decl, err := m.GetDecl(components)
	must(t, err)
	numeral, found, err := m.RewriteFPAFPComponentsNumerals(decl, sign, exponent, significand)
	must(t, err)
	if !found {
		t.Fatal("concrete FP components did not form a numeral")
	}
	converted, found, err := c.MkNumeral(numeral)
	must(t, err)
	if !found {
		t.Fatal("finite FPA numeral was not recognized")
	}
	bits, found, _, err := m.RewriteFPAToIEEEBVNumeral(numeral, true)
	must(t, err)
	if !found {
		t.Fatal("finite FPA numeral lost its IEEE encoding")
	}
	sort, err := m.GetSort(numeral)
	must(t, err)
	want, err := c.FromIEEEBV(sort, bits)
	must(t, err)
	if converted != want {
		t.Fatal("finite numeral conversion", converted, want)
	}
}

func TestConverterRoundingDecisionModes(t *testing.T) {
	m, c := fixture(t)
	bit, err := m.BitVectorSort(1)
	must(t, err)
	sign, err := m.MkVar(100, bit)
	must(t, err)
	last, err := m.MkVar(101, bit)
	must(t, err)
	round, err := m.MkVar(102, bit)
	must(t, err)
	sticky, err := m.MkVar(103, bit)
	must(t, err)

	lastOrSticky, err := m.MkBitVectorOr([]ast.ASTHandle{last, sticky})
	must(t, err)
	notRound, err := m.MkBitVectorNot(round)
	must(t, err)
	notLastOrSticky, err := m.MkBitVectorNot(lastOrSticky)
	must(t, err)
	evenRefusal, err := m.MkBitVectorOr([]ast.ASTHandle{notRound, notLastOrSticky})
	must(t, err)
	incrementEven, err := m.MkBitVectorNot(evenRefusal)
	must(t, err)
	roundOrSticky, err := m.MkBitVectorOr([]ast.ASTHandle{round, sticky})
	must(t, err)
	notRoundOrSticky, err := m.MkBitVectorNot(roundOrSticky)
	must(t, err)
	positiveRefusal, err := m.MkBitVectorOr([]ast.ASTHandle{sign, notRoundOrSticky})
	must(t, err)
	incrementPositive, err := m.MkBitVectorNot(positiveRefusal)
	must(t, err)
	notSign, err := m.MkBitVectorNot(sign)
	must(t, err)
	negativeRefusal, err := m.MkBitVectorOr([]ast.ASTHandle{notSign, notRoundOrSticky})
	must(t, err)
	incrementNegative, err := m.MkBitVectorNot(negativeRefusal)
	must(t, err)
	zero, err := m.BitVectorNumeralUnsigned(0, 1)
	must(t, err)

	for _, tc := range []struct {
		kind int32
		want ast.ASTHandle
	}{
		{ast.FPAOpRMNearestEven, incrementEven},
		{ast.FPAOpRMNearestAway, round},
		{ast.FPAOpRMTowardPositive, incrementPositive},
		{ast.FPAOpRMTowardNegative, incrementNegative},
		{ast.FPAOpRMTowardZero, zero},
	} {
		mode, e := c.MkRoundingMode(tc.kind)
		must(t, e)
		got, e := c.RoundingDecision(mode, sign, last, round, sticky)
		must(t, e)
		if got != tc.want {
			t.Fatal("rounding decision", tc.kind, got, tc.want)
		}
	}
	mode, err := c.MkRoundingMode(ast.FPAOpRMNearestEven)
	must(t, err)
	bad, err := m.BitVectorNumeralUnsigned(0, 2)
	must(t, err)
	if _, err = c.RoundingDecision(mode, bad, last, round, sticky); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("invalid rounding decision width", err)
	}
}

func TestConverterRoundKernelWidthsAndResultShape(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	mode, err := c.MkRoundingMode(ast.FPAOpRMNearestEven)
	must(t, err)
	signSort, err := m.BitVectorSort(1)
	must(t, err)
	significandSort, err := m.BitVectorSort(28)
	must(t, err)
	exponentSort, err := m.BitVectorSort(10)
	must(t, err)
	sign, err := m.MkVar(200, signSort)
	must(t, err)
	significand, err := m.MkVar(201, significandSort)
	must(t, err)
	exponent, err := m.MkVar(202, exponentSort)
	must(t, err)
	result, err := c.Round(sort, mode, sign, significand, exponent)
	must(t, err)
	gotSign, gotExponent, gotSignificand, err := c.SplitFP(result)
	must(t, err)
	for _, field := range []struct {
		value ast.ASTHandle
		width uint32
	}{{gotSign, 1}, {gotExponent, 8}, {gotSignificand, 23}} {
		width, e := m.BitVectorSize(field.value)
		must(t, e)
		if width != field.width {
			t.Fatal("rounded field width", width, field.width)
		}
	}
	badSort, err := m.BitVectorSort(27)
	must(t, err)
	bad, err := m.MkVar(203, badSort)
	must(t, err)
	if _, err = c.Round(sort, mode, sign, bad, exponent); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("invalid extended significand width", err)
	}
}

func TestConverterAddSubtractSpecialCasesAndSymbolicShape(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	rne, err := c.MkRoundingMode(ast.FPAOpRMNearestEven)
	must(t, err)
	rtn, err := c.MkRoundingMode(ast.FPAOpRMTowardNegative)
	must(t, err)
	nan, err := c.MkSpecial(sort, fpa.NaN)
	must(t, err)
	pinf, err := c.MkSpecial(sort, fpa.PositiveInfinity)
	must(t, err)
	ninf, err := c.MkSpecial(sort, fpa.NegativeInfinity)
	must(t, err)
	pzero, err := c.MkSpecial(sort, fpa.PositiveZero)
	must(t, err)
	nzero, err := c.MkSpecial(sort, fpa.NegativeZero)
	must(t, err)
	for _, tc := range []struct {
		mode        ast.ASTHandle
		left, right ast.ASTHandle
		want        ast.ASTHandle
	}{
		{rne, pinf, ninf, nan},
		{rne, pinf, pzero, pinf},
		{rne, pzero, nzero, pzero},
		{rtn, pzero, nzero, nzero},
		{rne, nzero, nzero, nzero},
		{rne, nan, pzero, nan},
	} {
		got, e := c.Add(tc.mode, tc.left, tc.right)
		must(t, e)
		if got != tc.want {
			t.Fatal("special addition", got, tc.want)
		}
	}
	negatedInfinity, err := c.Negate(pinf)
	must(t, err)
	wantSubtraction, err := c.Add(rne, pinf, negatedInfinity)
	must(t, err)
	if got, e := c.Sub(rne, pinf, pinf); e != nil || got != wantSubtraction {
		t.Fatal("subtraction add-of-negation", got, wantSubtraction, e)
	}

	leftDecl := declaration(t, m, "add-left", nil, sort)
	rightDecl := declaration(t, m, "add-right", nil, sort)
	left, err := c.MkConst(leftDecl)
	must(t, err)
	right, err := c.MkConst(rightDecl)
	must(t, err)
	result, err := c.Add(rne, left, right)
	must(t, err)
	if kind, e := m.GetDeclKind(result); e != nil || kind != ast.FPAOpFP {
		t.Fatal("symbolic addition encoding", kind, e)
	}
	resultSort, err := m.GetSort(result)
	must(t, err)
	if resultSort != sort {
		t.Fatal("symbolic addition sort", resultSort, sort)
	}
}

func TestConverterMultiplySpecialCasesAndSymbolicShape(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	mode, err := c.MkRoundingMode(ast.FPAOpRMNearestEven)
	must(t, err)
	values := make(map[fpa.SpecialValue]ast.ASTHandle)
	for _, kind := range []fpa.SpecialValue{fpa.NaN, fpa.PositiveInfinity, fpa.NegativeInfinity, fpa.PositiveZero, fpa.NegativeZero} {
		values[kind], err = c.MkSpecial(sort, kind)
		must(t, err)
	}
	for _, tc := range []struct {
		left, right fpa.SpecialValue
		want        fpa.SpecialValue
	}{
		{fpa.PositiveZero, fpa.PositiveInfinity, fpa.NaN},
		{fpa.NegativeZero, fpa.NegativeInfinity, fpa.NaN},
		{fpa.PositiveInfinity, fpa.NegativeInfinity, fpa.NegativeInfinity},
		{fpa.NegativeInfinity, fpa.NegativeInfinity, fpa.PositiveInfinity},
		{fpa.PositiveZero, fpa.NegativeZero, fpa.NegativeZero},
		{fpa.NaN, fpa.PositiveInfinity, fpa.NaN},
	} {
		got, e := c.Mul(mode, values[tc.left], values[tc.right])
		must(t, e)
		if got != values[tc.want] {
			t.Fatal("special multiplication", tc, got, values[tc.want])
		}
	}
	left, err := c.MkConst(declaration(t, m, "mul-left", nil, sort))
	must(t, err)
	right, err := c.MkConst(declaration(t, m, "mul-right", nil, sort))
	must(t, err)
	result, err := c.Mul(mode, left, right)
	must(t, err)
	if kind, e := m.GetDeclKind(result); e != nil || kind != ast.FPAOpFP {
		t.Fatal("symbolic multiplication encoding", kind, e)
	}
	resultSort, err := m.GetSort(result)
	must(t, err)
	if resultSort != sort {
		t.Fatal("symbolic multiplication sort", resultSort, sort)
	}
}

func TestConverterDivideSpecialCasesAndSymbolicShape(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	mode, err := c.MkRoundingMode(ast.FPAOpRMNearestEven)
	must(t, err)
	values := make(map[fpa.SpecialValue]ast.ASTHandle)
	for _, kind := range []fpa.SpecialValue{fpa.NaN, fpa.PositiveInfinity, fpa.NegativeInfinity, fpa.PositiveZero, fpa.NegativeZero} {
		values[kind], err = c.MkSpecial(sort, kind)
		must(t, err)
	}
	for _, tc := range []struct {
		left, right fpa.SpecialValue
		want        fpa.SpecialValue
	}{
		{fpa.PositiveInfinity, fpa.PositiveInfinity, fpa.NaN},
		{fpa.PositiveInfinity, fpa.PositiveZero, fpa.PositiveInfinity},
		{fpa.NegativeInfinity, fpa.PositiveZero, fpa.NegativeInfinity},
		{fpa.PositiveZero, fpa.PositiveZero, fpa.NaN},
		{fpa.NegativeZero, fpa.PositiveInfinity, fpa.NegativeZero},
		{fpa.NaN, fpa.PositiveZero, fpa.NaN},
	} {
		got, e := c.Div(mode, values[tc.left], values[tc.right])
		must(t, e)
		if got != values[tc.want] {
			t.Fatal("special division", tc, got, values[tc.want])
		}
	}
	left, err := c.MkConst(declaration(t, m, "div-left", nil, sort))
	must(t, err)
	right, err := c.MkConst(declaration(t, m, "div-right", nil, sort))
	must(t, err)
	result, err := c.Div(mode, left, right)
	must(t, err)
	if kind, e := m.GetDeclKind(result); e != nil || kind != ast.FPAOpFP {
		t.Fatal("symbolic division encoding", kind, e)
	}
	resultSort, err := m.GetSort(result)
	must(t, err)
	if resultSort != sort {
		t.Fatal("symbolic division sort", resultSort, sort)
	}
}

func TestConverterRemainderSpecialCasesAndSymbolicShape(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	values := make(map[fpa.SpecialValue]ast.ASTHandle)
	for _, kind := range []fpa.SpecialValue{fpa.NaN, fpa.PositiveInfinity, fpa.NegativeInfinity, fpa.PositiveZero, fpa.NegativeZero} {
		values[kind], err = c.MkSpecial(sort, kind)
		must(t, err)
	}
	for _, tc := range []struct {
		left, right fpa.SpecialValue
		want        fpa.SpecialValue
	}{
		{fpa.PositiveInfinity, fpa.PositiveInfinity, fpa.NaN},
		{fpa.PositiveZero, fpa.PositiveInfinity, fpa.PositiveZero},
		{fpa.NegativeZero, fpa.PositiveInfinity, fpa.NegativeZero},
		{fpa.PositiveInfinity, fpa.PositiveZero, fpa.NaN},
		{fpa.PositiveZero, fpa.PositiveZero, fpa.NaN},
		{fpa.NaN, fpa.PositiveInfinity, fpa.NaN},
	} {
		got, e := c.Rem(values[tc.left], values[tc.right])
		if e != nil {
			t.Fatalf("special remainder %v: %v", tc, e)
		}
		if got != values[tc.want] {
			t.Fatal("special remainder", tc, got, values[tc.want])
		}
	}
	left, err := c.MkConst(declaration(t, m, "rem-left", nil, sort))
	must(t, err)
	right, err := c.MkConst(declaration(t, m, "rem-right", nil, sort))
	must(t, err)
	result, err := c.Rem(left, right)
	must(t, err)
	if kind, e := m.GetDeclKind(result); e != nil || kind != ast.FPAOpFP {
		t.Fatal("symbolic remainder encoding", kind, e)
	}
	resultSort, err := m.GetSort(result)
	must(t, err)
	if resultSort != sort {
		t.Fatal("symbolic remainder sort", resultSort, sort)
	}
}

func TestConverterRoundToIntegralSpecialCasesAndSymbolicShape(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	mode, err := c.MkRoundingMode(ast.FPAOpRMNearestEven)
	must(t, err)
	for _, kind := range []fpa.SpecialValue{fpa.NaN, fpa.PositiveInfinity, fpa.NegativeInfinity, fpa.PositiveZero, fpa.NegativeZero} {
		value, e := c.MkSpecial(sort, kind)
		must(t, e)
		got, e := c.RoundToIntegral(mode, value)
		must(t, e)
		if got != value {
			t.Fatal("round-to-integral special", kind, got, value)
		}
	}
	value, err := c.MkConst(declaration(t, m, "integral-value", nil, sort))
	must(t, err)
	result, err := c.RoundToIntegral(mode, value)
	must(t, err)
	if kind, e := m.GetDeclKind(result); e != nil || kind != ast.FPAOpFP {
		t.Fatal("symbolic round-to-integral encoding", kind, e)
	}
	resultSort, err := m.GetSort(result)
	must(t, err)
	if resultSort != sort {
		t.Fatal("symbolic round-to-integral sort", resultSort, sort)
	}
}

func TestConverterSquareRootSpecialCasesAndSymbolicShape(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	mode, err := c.MkRoundingMode(ast.FPAOpRMNearestEven)
	must(t, err)
	nan, err := c.MkSpecial(sort, fpa.NaN)
	must(t, err)
	positiveInfinity, err := c.MkSpecial(sort, fpa.PositiveInfinity)
	must(t, err)
	negativeInfinity, err := c.MkSpecial(sort, fpa.NegativeInfinity)
	must(t, err)
	positiveZero, err := c.MkSpecial(sort, fpa.PositiveZero)
	must(t, err)
	negativeZero, err := c.MkSpecial(sort, fpa.NegativeZero)
	must(t, err)
	for _, tc := range []struct {
		value, want ast.ASTHandle
	}{
		{nan, nan},
		{positiveInfinity, positiveInfinity},
		{negativeInfinity, nan},
		{positiveZero, positiveZero},
		{negativeZero, negativeZero},
	} {
		got, e := c.Sqrt(mode, tc.value)
		must(t, e)
		if got != tc.want {
			t.Fatal("square-root special", got, tc.want)
		}
	}
	value, err := c.MkConst(declaration(t, m, "sqrt-value", nil, sort))
	must(t, err)
	result, err := c.Sqrt(mode, value)
	must(t, err)
	if kind, e := m.GetDeclKind(result); e != nil || kind != ast.FPAOpFP {
		t.Fatal("symbolic square-root encoding", kind, e)
	}
	resultSort, err := m.GetSort(result)
	must(t, err)
	if resultSort != sort {
		t.Fatal("symbolic square-root sort", resultSort, sort)
	}
}

func TestConverterFMASpecialCasesAndSymbolicShape(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	rne, err := c.MkRoundingMode(ast.FPAOpRMNearestEven)
	must(t, err)
	rtn, err := c.MkRoundingMode(ast.FPAOpRMTowardNegative)
	must(t, err)
	values := make(map[fpa.SpecialValue]ast.ASTHandle)
	for _, kind := range []fpa.SpecialValue{fpa.NaN, fpa.PositiveInfinity, fpa.NegativeInfinity, fpa.PositiveZero, fpa.NegativeZero} {
		values[kind], err = c.MkSpecial(sort, kind)
		must(t, err)
	}
	for _, tc := range []struct {
		mode    ast.ASTHandle
		x, y, z fpa.SpecialValue
		want    fpa.SpecialValue
	}{
		{rne, fpa.NaN, fpa.PositiveInfinity, fpa.PositiveZero, fpa.NaN},
		{rne, fpa.PositiveInfinity, fpa.PositiveZero, fpa.PositiveZero, fpa.NaN},
		{rne, fpa.PositiveInfinity, fpa.PositiveInfinity, fpa.NegativeInfinity, fpa.NaN},
		{rne, fpa.PositiveInfinity, fpa.NegativeInfinity, fpa.NegativeInfinity, fpa.NegativeInfinity},
		{rne, fpa.PositiveZero, fpa.PositiveZero, fpa.NegativeZero, fpa.PositiveZero},
		{rtn, fpa.PositiveZero, fpa.PositiveZero, fpa.NegativeZero, fpa.NegativeZero},
		{rne, fpa.PositiveZero, fpa.PositiveZero, fpa.PositiveInfinity, fpa.PositiveInfinity},
	} {
		got, e := c.FMA(tc.mode, values[tc.x], values[tc.y], values[tc.z])
		must(t, e)
		if got != values[tc.want] {
			t.Fatal("FMA special", tc, got, values[tc.want])
		}
	}
	x, err := c.MkConst(declaration(t, m, "fma-x", nil, sort))
	must(t, err)
	y, err := c.MkConst(declaration(t, m, "fma-y", nil, sort))
	must(t, err)
	z, err := c.MkConst(declaration(t, m, "fma-z", nil, sort))
	must(t, err)
	result, err := c.FMA(rne, x, y, z)
	must(t, err)
	if kind, e := m.GetDeclKind(result); e != nil || kind != ast.FPAOpFP {
		t.Fatal("symbolic FMA encoding", kind, e)
	}
	resultSort, err := m.GetSort(result)
	must(t, err)
	if resultSort != sort {
		t.Fatal("symbolic FMA sort", resultSort, sort)
	}
}

func fixture(t *testing.T) (*ast.ASTManager, *fpa.Converter) {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	s, err := usymbol.NewManager(usymbol.Config{Budget: b})
	must(t, err)
	n, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	must(t, err)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: n, Budget: b})
	must(t, err)
	c, err := fpa.NewConverter(m)
	must(t, err)
	t.Cleanup(func() {
		must(t, c.Close())
		must(t, m.Close())
		must(t, n.Close())
		must(t, s.Close())
		if b.Used() != 0 {
			t.Fatalf("retained %d bytes", b.Used())
		}
	})
	return m, c
}

func declaration(t *testing.T, m *ast.ASTManager, name string, domain []ast.ASTHandle, result ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	symbol, err := m.Symbols.Intern(name)
	must(t, err)
	decl, err := m.MkFuncDecl(symbol, domain, result, ast.ASTFuncDeclInfo{})
	must(t, err)
	return decl
}

func TestConverterSortAccessSpecialsSplitAndJoin(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(40, 65)
	must(t, err)
	ebits, sbits, matched, err := m.FloatingPointSortShape(sort)
	must(t, err)
	if !matched || ebits != 40 || sbits != 65 {
		t.Fatal("floating-point sort shape", matched, ebits, sbits)
	}
	rmSort, err := m.RoundingModeSort()
	must(t, err)
	if yes, e := m.IsRoundingModeSort(rmSort); e != nil || !yes {
		t.Fatal("rounding-mode sort", yes, e)
	}

	nan, err := c.MkSpecial(sort, fpa.NaN)
	must(t, err)
	sign, exponent, significand, err := c.SplitFP(nan)
	must(t, err)
	zero1, err := m.BitVectorNumeralUnsigned(0, 1)
	must(t, err)
	top40, err := m.BitVectorNumeralAllOnes(40)
	must(t, err)
	one64, err := m.BitVectorNumeralUnsigned(1, 64)
	must(t, err)
	if sign != zero1 || exponent != top40 || significand != one64 {
		t.Fatal("canonical NaN components", sign, exponent, significand)
	}
	joined, err := c.JoinFP(nan)
	must(t, err)
	want, err := m.MkBitVectorConcat([]ast.ASTHandle{zero1, top40, one64})
	must(t, err)
	if joined != want {
		t.Fatal("joined IEEE field order", joined, want)
	}
}

func TestConverterNaNEqualityDistinctAndComponentITE(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	first, err := c.MkSpecial(sort, fpa.NaN)
	must(t, err)
	one, err := m.BitVectorNumeralUnsigned(1, 1)
	must(t, err)
	top, err := m.BitVectorNumeralAllOnes(8)
	must(t, err)
	payload, err := m.BitVectorNumeralUnsigned(7, 23)
	must(t, err)
	second, err := c.MkFP(one, top, payload)
	must(t, err)
	equal, err := c.MkEq(first, second)
	must(t, err)
	if !m.IsTrue(equal) {
		t.Fatal("all NaN payloads must denote the single SMT NaN", equal)
	}
	distinct, err := c.MkDistinct([]ast.ASTHandle{first, second})
	must(t, err)
	if !m.IsFalse(distinct) {
		t.Fatal("distinct NaN NaN must be false", distinct)
	}

	positiveZero, err := c.MkSpecial(sort, fpa.PositiveZero)
	must(t, err)
	negativeZero, err := c.MkSpecial(sort, fpa.NegativeZero)
	must(t, err)
	equal, err = c.MkEq(positiveZero, negativeZero)
	must(t, err)
	if !m.IsFalse(equal) {
		t.Fatal("SMT equality preserves the zero sign", equal)
	}
	truth, err := m.MkTrue()
	must(t, err)
	choice, err := c.MkITE(truth, positiveZero, negativeZero)
	must(t, err)
	if choice != positiveZero {
		t.Fatal("component ITE true branch", choice, positiveZero)
	}
}

func TestConverterRoundingModeCodesAndEquality(t *testing.T) {
	m, c := fixture(t)
	kinds := []int32{
		ast.FPAOpRMNearestEven,
		ast.FPAOpRMNearestAway,
		ast.FPAOpRMTowardPositive,
		ast.FPAOpRMTowardNegative,
		ast.FPAOpRMTowardZero,
	}
	for code, kind := range kinds {
		rm, err := c.MkRoundingMode(kind)
		must(t, err)
		bits, err := m.GetArg(rm, 0)
		must(t, err)
		want, err := m.BitVectorNumeralUnsigned(uint32(code), 3)
		must(t, err)
		if bits != want {
			t.Fatalf("rounding mode %d: got %v want %v", kind, bits, want)
		}
		equal, err := c.MkEq(rm, rm)
		must(t, err)
		if !m.IsTrue(equal) {
			t.Fatal("rounding mode reflexivity", kind, equal)
		}
	}
}

func TestConverterClassificationsUnarySignsAndComparisons(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	values := make(map[fpa.SpecialValue]ast.ASTHandle)
	for _, value := range []fpa.SpecialValue{fpa.PositiveInfinity, fpa.NegativeInfinity, fpa.NaN, fpa.PositiveZero, fpa.NegativeZero} {
		values[value], err = c.MkSpecial(sort, value)
		must(t, err)
	}
	zeroSign, err := m.BitVectorNumeralUnsigned(0, 1)
	must(t, err)
	zeroExp, err := m.BitVectorNumeralUnsigned(0, 8)
	must(t, err)
	oneExp, err := m.BitVectorNumeralUnsigned(1, 8)
	must(t, err)
	zeroSig, err := m.BitVectorNumeralUnsigned(0, 23)
	must(t, err)
	oneSig, err := m.BitVectorNumeralUnsigned(1, 23)
	must(t, err)
	subnormal, err := c.MkFP(zeroSign, zeroExp, oneSig)
	must(t, err)
	normal, err := c.MkFP(zeroSign, oneExp, zeroSig)
	must(t, err)

	classify := func(value ast.ASTHandle, kind ast.FPAClassificationKind, want bool) {
		t.Helper()
		got, e := c.Classify(value, kind)
		must(t, e)
		if m.IsTrue(got) != want || m.IsFalse(got) == want {
			t.Fatalf("classification %d: got %v want %v", kind, got, want)
		}
	}
	classify(values[fpa.NaN], ast.FPAIsNaN, true)
	classify(values[fpa.NaN], ast.FPAIsPositive, false)
	classify(values[fpa.PositiveInfinity], ast.FPAIsInfinite, true)
	classify(values[fpa.NegativeInfinity], ast.FPAIsNegative, true)
	classify(values[fpa.PositiveZero], ast.FPAIsZero, true)
	classify(values[fpa.NegativeZero], ast.FPAIsNegative, true)
	classify(subnormal, ast.FPAIsSubnormal, true)
	classify(subnormal, ast.FPAIsNormal, false)
	classify(normal, ast.FPAIsNormal, true)

	abs, err := c.Abs(values[fpa.NegativeZero])
	must(t, err)
	if abs != values[fpa.PositiveZero] {
		t.Fatal("absolute value clears sign", abs, values[fpa.PositiveZero])
	}
	negatedNaN, err := c.Negate(values[fpa.NaN])
	must(t, err)
	if negatedNaN != values[fpa.NaN] {
		t.Fatal("negation preserves NaN sign", negatedNaN, values[fpa.NaN])
	}

	compare := func(kind int32, left, right ast.ASTHandle, want bool) {
		t.Helper()
		got, e := c.Compare(kind, left, right)
		must(t, e)
		if m.IsTrue(got) != want || m.IsFalse(got) == want {
			t.Fatalf("comparison %d: got %v want %v", kind, got, want)
		}
	}
	compare(ast.FPAOpEq, values[fpa.PositiveZero], values[fpa.NegativeZero], true)
	compare(ast.FPAOpEq, values[fpa.NaN], values[fpa.NaN], false)
	compare(ast.FPAOpLt, values[fpa.NegativeInfinity], values[fpa.PositiveInfinity], true)
	compare(ast.FPAOpLt, values[fpa.PositiveInfinity], values[fpa.NegativeInfinity], false)
	compare(ast.FPAOpLt, values[fpa.NegativeZero], values[fpa.PositiveZero], false)
	compare(ast.FPAOpLt, values[fpa.NaN], values[fpa.PositiveInfinity], false)
	compare(ast.FPAOpLe, values[fpa.NegativeZero], values[fpa.PositiveZero], true)
	compare(ast.FPAOpGt, values[fpa.PositiveInfinity], values[fpa.NegativeInfinity], true)
	compare(ast.FPAOpGe, values[fpa.NegativeZero], values[fpa.PositiveZero], true)
}

func TestConverterBiasLeadingZerosAndUnpack(t *testing.T) {
	m, c := fixture(t)
	zeroBit, err := m.BitVectorNumeralUnsigned(0, 1)
	must(t, err)
	oneBit, err := m.BitVectorNumeralUnsigned(1, 1)
	must(t, err)
	leading, err := c.LeadingZeros(zeroBit, 8)
	must(t, err)
	wantOne, err := m.BitVectorNumeralUnsigned(1, 8)
	must(t, err)
	if leading != wantOne {
		t.Fatal("one-bit zero leading count", leading, wantOne)
	}
	leading, err = c.LeadingZeros(oneBit, 8)
	must(t, err)
	wantZero, err := m.BitVectorNumeralUnsigned(0, 8)
	must(t, err)
	if leading != wantZero {
		t.Fatal("one-bit one leading count", leading, wantZero)
	}
	exponent, err := m.MkVar(0, func() ast.ASTHandle {
		s, e := m.BitVectorSort(8)
		must(t, e)
		return s
	}())
	must(t, err)
	biased, err := c.Bias(exponent)
	must(t, err)
	unbiased, err := c.Unbias(exponent)
	must(t, err)
	if width, e := m.BitVectorSize(biased); e != nil || width != 8 {
		t.Fatal("biased exponent width", width, e)
	}
	if width, e := m.BitVectorSize(unbiased); e != nil || width != 8 {
		t.Fatal("unbiased exponent width", width, e)
	}

	zeroSig, err := m.BitVectorNumeralUnsigned(0, 23)
	must(t, err)
	oneSig, err := m.BitVectorNumeralUnsigned(1, 23)
	must(t, err)
	oneExp, err := m.BitVectorNumeralUnsigned(1, 8)
	must(t, err)
	zeroExp, err := m.BitVectorNumeralUnsigned(0, 8)
	must(t, err)
	normal, err := c.MkFP(zeroBit, oneExp, zeroSig)
	must(t, err)
	normalParts, err := c.Unpack(normal, false)
	must(t, err)
	wantNormalSig, err := m.MkBitVectorConcatPair(oneBit, zeroSig)
	must(t, err)
	wantNormalExp, err := c.Unbias(oneExp)
	must(t, err)
	if normalParts.Sign != zeroBit || normalParts.Significand != wantNormalSig || normalParts.Exponent != wantNormalExp || normalParts.LeadingZeros != zeroExp {
		t.Fatal("normal unpack", normalParts)
	}
	subnormal, err := c.MkFP(zeroBit, zeroExp, oneSig)
	must(t, err)
	subnormalParts, err := c.Unpack(subnormal, false)
	must(t, err)
	wantSubnormalSig, err := m.MkBitVectorZeroExtend(1, oneSig)
	must(t, err)
	if subnormalParts.Significand != wantSubnormalSig || subnormalParts.Exponent != wantNormalExp || subnormalParts.LeadingZeros != zeroExp {
		t.Fatal("subnormal unpack", subnormalParts)
	}
	normalized, err := c.Unpack(subnormal, true)
	must(t, err)
	if width, e := m.BitVectorSize(normalized.Significand); e != nil || width != 24 {
		t.Fatal("normalized significand width", width, e)
	}
	if width, e := m.BitVectorSize(normalized.LeadingZeros); e != nil || width != 8 {
		t.Fatal("normalization shift width", width, e)
	}
}

func TestConverterSymbolCachesAssertionsAndReset(t *testing.T) {
	m, c := fixture(t)
	floatSort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	rmSort, err := m.RoundingModeSort()
	must(t, err)
	floatConstant := declaration(t, m, "symbolic-float", nil, floatSort)
	rmConstant := declaration(t, m, "symbolic-rm", nil, rmSort)
	for _, decl := range []ast.ASTHandle{floatConstant, rmConstant} {
		must(t, m.IncRef(decl))
		decl := decl
		t.Cleanup(func() { must(t, m.DecRef(decl)) })
	}
	first, err := c.MkConst(floatConstant)
	must(t, err)
	second, err := c.MkConst(floatConstant)
	must(t, err)
	if first != second {
		t.Fatal("floating constant cache", first, second)
	}
	rmFirst, err := c.MkRMConst(rmConstant)
	must(t, err)
	rmSecond, err := c.MkRMConst(rmConstant)
	must(t, err)
	if rmFirst != rmSecond || len(c.ExtraAssertions()) != 1 {
		t.Fatal("rounding constant cache/assertion", rmFirst, rmSecond, len(c.ExtraAssertions()))
	}
	if sorted, e := m.IsWellSorted(c.ExtraAssertions()[0], nil); e != nil || !sorted {
		t.Fatal("rounding range assertion", sorted, e)
	}

	boolSort, err := m.MkBoolSort()
	must(t, err)
	uf := declaration(t, m, "symbolic-fp-uf", []ast.ASTHandle{boolSort}, floatSort)
	must(t, m.IncRef(uf))
	t.Cleanup(func() { must(t, m.DecRef(uf)) })
	truth, err := m.MkTrue()
	must(t, err)
	converted, err := c.MkUF(uf, []ast.ASTHandle{truth})
	must(t, err)
	if _, _, _, e := c.SplitFP(converted); e != nil {
		t.Fatal("converted FP UF", e)
	}
	if len(c.ExtraAssertions()) != 2 {
		t.Fatal("UF connection assertion", len(c.ExtraAssertions()))
	}
	if sorted, e := m.IsWellSorted(c.ExtraAssertions()[1], nil); e != nil || !sorted {
		t.Fatal("UF connection assertion sorting", sorted, e)
	}
	must(t, c.Reset())
	if len(c.ExtraAssertions()) != 0 {
		t.Fatal("reset assertions", len(c.ExtraAssertions()))
	}
	third, err := c.MkConst(floatConstant)
	must(t, err)
	if third == first {
		t.Fatal("reset must discard the fresh constant cache")
	}
}

func TestConverterMinMaxSpecialCasesAndStableUnspecifiedZeros(t *testing.T) {
	m, c := fixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	must(t, err)
	family, err := m.ExistingFloatingPointFamilyID()
	must(t, err)
	minDecl, err := m.MkFamilyFuncDecl(family, ast.FPAOpMin, nil, []ast.ASTHandle{sort, sort}, ast.ASTHandle{})
	must(t, err)
	maxDecl, err := m.MkFamilyFuncDecl(family, ast.FPAOpMax, nil, []ast.ASTHandle{sort, sort}, ast.ASTHandle{})
	must(t, err)
	pinf, err := c.MkSpecial(sort, fpa.PositiveInfinity)
	must(t, err)
	ninf, err := c.MkSpecial(sort, fpa.NegativeInfinity)
	must(t, err)
	nan, err := c.MkSpecial(sort, fpa.NaN)
	must(t, err)
	pzero, err := c.MkSpecial(sort, fpa.PositiveZero)
	must(t, err)
	nzero, err := c.MkSpecial(sort, fpa.NegativeZero)
	must(t, err)
	minimum, err := c.Extremum(minDecl, ast.FPAOpMin, ninf, pinf)
	must(t, err)
	if minimum != ninf {
		t.Fatal("minimum order", minimum, ninf)
	}
	maximum, err := c.Extremum(maxDecl, ast.FPAOpMax, ninf, pinf)
	must(t, err)
	if maximum != pinf {
		t.Fatal("maximum order", maximum, pinf)
	}
	minimum, err = c.Extremum(minDecl, ast.FPAOpMin, nan, pinf)
	must(t, err)
	if minimum != pinf {
		t.Fatal("minimum ignores one NaN", minimum, pinf)
	}
	first, err := c.Extremum(minDecl, ast.FPAOpMin, pzero, nzero)
	must(t, err)
	second, err := c.Extremum(minDecl, ast.FPAOpMin, pzero, nzero)
	must(t, err)
	if first != second {
		t.Fatal("stable unspecified zero result", first, second)
	}
	otherOrder, err := c.Extremum(minDecl, ast.FPAOpMin, nzero, pzero)
	must(t, err)
	firstSign, firstExp, firstSig, err := c.SplitFP(first)
	must(t, err)
	otherSign, otherExp, otherSig, err := c.SplitFP(otherOrder)
	must(t, err)
	zeroExp, err := m.BitVectorNumeralUnsigned(0, 8)
	must(t, err)
	zeroSig, err := m.BitVectorNumeralUnsigned(0, 23)
	must(t, err)
	if firstSign == otherSign || firstExp != zeroExp || otherExp != zeroExp || firstSig != zeroSig || otherSig != zeroSig {
		t.Fatal("opposite-zero unspecified encodings", first, otherOrder)
	}
}
