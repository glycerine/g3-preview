package ast

import (
	"strings"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestASTFPAToFPExponentRationalOrdersAndRefusals(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	rmSort, err := m.MkFamilySort(family, 1, nil)
	astTestMust(t, err)
	realSort := m.external(m.floatingPointPlugin.real)
	intSort := m.external(m.floatingPointPlugin.integer)
	eParam, err := m.NewParameterInt(8)
	astTestMust(t, err)
	sParam, err := m.NewParameterInt(24)
	astTestMust(t, err)
	defer func() {
		astTestMust(t, m.DelParameter(&sParam))
		astTestMust(t, m.DelParameter(&eParam))
	}()
	ps := []ASTParameter{eParam, sParam}
	declRealInt, err := m.MkFamilyFuncDecl(family, FPAOpToFP, ps,
		[]ASTHandle{rmSort, realSort, intSort}, ASTHandle{})
	astTestMust(t, err)
	astTestPin(t, m, declRealInt)
	declIntReal, err := m.MkFamilyFuncDecl(family, FPAOpToFP, ps,
		[]ASTHandle{rmSort, intSort, realSort}, ASTHandle{})
	astTestMust(t, err)
	astTestPin(t, m, declIntReal)
	rne, err := m.MkFamilyApp(family, astFPARMNearestEven, nil, nil)
	astTestMust(t, err)
	astTestPin(t, m, rne)
	rtp, err := m.MkFamilyApp(family, astFPARMTowardPositive, nil, nil)
	astTestMust(t, err)
	astTestPin(t, m, rtp)
	q, err := m.numerals.NewValue()
	astTestMust(t, err)
	defer func() { astTestMust(t, m.numerals.Del(&q)) }()
	makeArithmetic := func(value string, integer bool) ASTHandle {
		astTestMust(t, m.numerals.SetString(&q, value))
		a, e := m.ArithmeticNumeral(&q, integer)
		astTestMust(t, e)
		return a
	}
	makeFloat := func(value float64) ASTHandle {
		var v unumeral.ArbitraryFloat
		astTestMust(t, m.floatingPointPlugin.numeric.SetDouble(&v, 8, 24, value))
		n, e := m.floatingPointPlugin.mkNumeral(&v, nil)
		astTestMust(t, e)
		astTestMust(t, m.floatingPointPlugin.numeric.Del(&v))
		return m.external(n)
	}
	for _, tc := range []struct {
		real, exponent string
		mode           ASTHandle
		want           float64
	}{
		{"3/2", "2", rne, 6},
		{"-3/2", "-1", rne, -0.75},
		{"16777217/16777216", "0", rne, 1},
		{"16777217/16777216", "0", rtp, 1.0000001192092896},
	} {
		real := makeArithmetic(tc.real, false)
		exponent := makeArithmetic(tc.exponent, true)
		for _, order := range []struct {
			decl ASTHandle
			args [2]ASTHandle
		}{
			{declRealInt, [2]ASTHandle{real, exponent}},
			{declIntReal, [2]ASTHandle{exponent, real}},
		} {
			got, found, e := m.RewriteFPAToFPExponentRationalNumerals(order.decl, tc.mode, order.args[0], order.args[1])
			astTestMust(t, e)
			if !found || got != makeFloat(tc.want) || m.floatingPointPlugin.numeralProbeFrames != nil {
				t.Fatal("exponent-rational conversion or scoped cleanup", tc, found, got)
			}
		}
	}
	realVar, err := m.MkVar(0, realSort)
	astTestMust(t, err)
	intVar, err := m.MkVar(0, intSort)
	astTestMust(t, err)
	goodReal := makeArithmetic("3/2", false)
	goodInt := makeArithmetic("2", true)
	for _, args := range [][2]ASTHandle{{realVar, goodInt}, {goodReal, intVar}} {
		got, found, e := m.RewriteFPAToFPExponentRationalNumerals(declRealInt, rne, args[0], args[1])
		astTestMust(t, e)
		if found || got != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("symbolic arithmetic input should refuse", found, got)
		}
	}
	symbolicRM, err := m.MkVar(0, rmSort)
	astTestMust(t, err)
	got, found, err := m.RewriteFPAToFPExponentRationalNumerals(declRealInt, symbolicRM, goodReal, goodInt)
	astTestMust(t, err)
	if found || got != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("symbolic rounding mode should refuse", found, got)
	}
	for _, domain := range [][]ASTHandle{{rmSort, realSort, intSort}, {rmSort, intSort, realSort}} {
		_, e := m.MkFamilyFuncDecl(family, FPAOpToFP, nil, domain, ASTHandle{})
		if e == nil || !strings.Contains(e.Error(), "expecting two integer parameters") {
			t.Fatal("three-argument parameter validation", e)
		}
	}
}
