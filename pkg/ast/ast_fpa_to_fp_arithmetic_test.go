package ast

import (
	"strings"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestASTFPAToFPArithmeticRealIntRoundingAndRefusals(t *testing.T) {
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
	declarations := [2]ASTHandle{}
	for i, sort := range [...]ASTHandle{realSort, intSort} {
		declarations[i], err = m.MkFamilyFuncDecl(family, FPAOpToFP, ps, []ASTHandle{rmSort, sort}, ASTHandle{})
		astTestMust(t, err)
		astTestPin(t, m, declarations[i])
	}
	rne, err := m.MkFamilyApp(family, astFPARMNearestEven, nil, nil)
	astTestMust(t, err)
	astTestPin(t, m, rne)
	rtp, err := m.MkFamilyApp(family, astFPARMTowardPositive, nil, nil)
	astTestMust(t, err)
	astTestPin(t, m, rtp)
	makeFloat := func(n float64) ASTHandle {
		var v unumeral.ArbitraryFloat
		astTestMust(t, m.floatingPointPlugin.numeric.SetDouble(&v, 8, 24, n))
		r, e := m.floatingPointPlugin.mkNumeral(&v, nil)
		astTestMust(t, e)
		astTestMust(t, m.floatingPointPlugin.numeric.Del(&v))
		return m.external(r)
	}
	q, err := m.numerals.NewValue()
	astTestMust(t, err)
	defer func() { astTestMust(t, m.numerals.Del(&q)) }()
	for _, tc := range []struct {
		integer bool
		value   string
		rm      ASTHandle
		want    float64
	}{
		{false, "16777217/16777216", rne, 1},
		{false, "16777217/16777216", rtp, 1.0000001192092896},
		{false, "-3/2", rne, -1.5},
		{true, "16777217", rne, 16777216},
		{true, "16777217", rtp, 16777218},
		{true, "18446744073709551617", rne, 18446744073709551616},
	} {
		astTestMust(t, m.numerals.SetString(&q, tc.value))
		arg, e := m.ArithmeticNumeral(&q, tc.integer)
		astTestMust(t, e)
		decl := declarations[0]
		if tc.integer {
			decl = declarations[1]
		}
		got, found, e := m.RewriteFPAToFPArithmeticNumeral(decl, tc.rm, arg)
		astTestMust(t, e)
		if !found || got != makeFloat(tc.want) || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("arithmetic conversion or scoped cleanup", tc, found, got)
		}
	}
	for i, sort := range [...]ASTHandle{realSort, intSort} {
		variable, e := m.MkVar(0, sort)
		astTestMust(t, e)
		got, found, e := m.RewriteFPAToFPArithmeticNumeral(declarations[i], rne, variable)
		astTestMust(t, e)
		if found || got != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("symbolic arithmetic input should refuse", i, found, got)
		}
	}
	symbolicRM, err := m.MkVar(0, rmSort)
	astTestMust(t, err)
	astTestMust(t, m.numerals.SetInt64(&q, 0))
	zero, err := m.ArithmeticNumeral(&q, false)
	astTestMust(t, err)
	got, found, err := m.RewriteFPAToFPArithmeticNumeral(declarations[0], symbolicRM, zero)
	astTestMust(t, err)
	if found || got != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("symbolic rounding mode should refuse", found, got)
	}
	for _, declSort := range [...]ASTHandle{realSort, intSort} {
		_, e := m.MkFamilyFuncDecl(family, FPAOpToFP, nil, []ASTHandle{rmSort, declSort}, ASTHandle{})
		if e == nil || !strings.Contains(e.Error(), "expecting two integer parameters") {
			t.Fatal("arithmetic declaration parameters", e)
		}
	}
}
