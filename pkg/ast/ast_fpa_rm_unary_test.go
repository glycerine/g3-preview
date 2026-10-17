package ast

import (
	"strings"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestASTFPARMUnaryDeclarationsModesAndNumericCleanup(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	rmSort, err := m.MkFamilySort(family, 1, nil)
	astTestMust(t, err)
	floatSort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	makeValue := func(n float64) ASTHandle {
		var value unumeral.ArbitraryFloat
		astTestMust(t, m.floatingPointPlugin.numeric.SetDouble(&value, 8, 24, n))
		ref, e := m.floatingPointPlugin.mkNumeral(&value, nil)
		astTestMust(t, e)
		astTestMust(t, m.floatingPointPlugin.numeric.Del(&value))
		return m.external(ref)
	}
	four, two, half, three := makeValue(4), makeValue(2), makeValue(2.5), makeValue(3)
	modes := []struct {
		kind int32
		mode unumeral.ArbitraryFloatRoundingMode
		want ASTHandle
	}{
		{astFPARMNearestEven, unumeral.ArbitraryFloatNearestEven, two},
		{astFPARMNearestAway, unumeral.ArbitraryFloatNearestAway, three},
		{astFPARMTowardPositive, unumeral.ArbitraryFloatTowardPositive, three},
		{astFPARMTowardNegative, unumeral.ArbitraryFloatTowardNegative, two},
		{astFPARMTowardZero, unumeral.ArbitraryFloatTowardZero, two},
	}
	for _, tc := range modes {
		rm, e := m.MkFamilyApp(family, tc.kind, nil, nil)
		astTestMust(t, e)
		found, mode, e := m.ReadFPARoundingMode(rm)
		astTestMust(t, e)
		if !found || mode != tc.mode {
			t.Fatal("rounding mode source mapping", tc.kind, found, mode)
		}
		out, found, e := m.RewriteFPARMUnaryNumeral(rm, half, FPAOpRoundToIntegral)
		astTestMust(t, e)
		if !found || out != tc.want || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("roundToIntegral mode or scoped cleanup", tc.kind, found, out)
		}
		out, found, e = m.RewriteFPARMUnaryNumeral(rm, four, FPAOpSqrt)
		astTestMust(t, e)
		if !found || out != two || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("sqrt mode or scoped cleanup", tc.kind, found, out)
		}
	}
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	for _, tc := range []struct {
		kind int32
		name string
	}{{FPAOpSqrt, "fp.sqrt"}, {FPAOpRoundToIntegral, "fp.roundToIntegral"}} {
		decl, e := m.MkFamilyFuncDecl(family, tc.kind, nil, []ASTHandle{rmSort, floatSort}, ASTHandle{})
		astTestMust(t, e)
		r, e := m.internal(decl)
		astTestMust(t, e)
		name, e := m.Symbols.String(m.declAt(r.slot).name)
		astTestMust(t, e)
		floatRef, e := m.internal(floatSort)
		astTestMust(t, e)
		if name != tc.name || m.declAt(r.slot).arity != 2 || m.declAt(r.slot).rangeSort != floatRef {
			t.Fatal("rounding unary declaration", tc.kind, name)
		}
		for _, domain := range [][]ASTHandle{nil, {rmSort}, {rmSort, floatSort, floatSort}, {boolSort, floatSort}, {rmSort, boolSort}} {
			_, e := m.MkFamilyFuncDecl(family, tc.kind, nil, domain, ASTHandle{})
			if e == nil || !strings.Contains(e.Error(), "floating point") && !strings.Contains(e.Error(), "RoundingMode") && !strings.Contains(e.Error(), "FloatingPoint") {
				t.Fatal("rounding unary invalid domain", tc.kind, domain, e)
			}
		}
		rm, e := m.MkFamilyApp(family, astFPARMNearestEven, nil, nil)
		astTestMust(t, e)
		app, e := m.MkFamilyApp(family, tc.kind, nil, []ASTHandle{rm, four})
		astTestMust(t, e)
		gotSort, e := m.GetSort(app)
		astTestMust(t, e)
		if gotSort != floatSort {
			t.Fatal("rounding unary application sort", tc.kind)
		}
		variableRM, e := m.MkVar(0, rmSort)
		astTestMust(t, e)
		out, found, e := m.RewriteFPARMUnaryNumeral(variableRM, four, tc.kind)
		astTestMust(t, e)
		if found || out != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("symbolic rounding mode refusal", tc.kind, found, out)
		}
		variableFloat, e := m.MkVar(0, floatSort)
		astTestMust(t, e)
		out, found, e = m.RewriteFPARMUnaryNumeral(rm, variableFloat, tc.kind)
		astTestMust(t, e)
		if found || out != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("symbolic floating point refusal", tc.kind, found, out)
		}
	}
}
