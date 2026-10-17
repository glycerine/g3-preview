package ast

import (
	"strings"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestASTFPAToFPUnsignedDeclarationRoundingAndRefusals(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	rmSort, err := m.MkFamilySort(family, 1, nil)
	astTestMust(t, err)
	bvSort, err := m.BitVectorSort(32)
	astTestMust(t, err)
	eParam, err := m.NewParameterInt(8)
	astTestMust(t, err)
	sParam, err := m.NewParameterInt(24)
	astTestMust(t, err)
	defer func() {
		astTestMust(t, m.DelParameter(&sParam))
		astTestMust(t, m.DelParameter(&eParam))
	}()
	ps := []ASTParameter{eParam, sParam}
	domain := []ASTHandle{rmSort, bvSort}
	decl, err := m.MkFamilyFuncDecl(family, FPAOpToFPUnsigned, ps, domain, ASTHandle{})
	astTestMust(t, err)
	astTestPin(t, m, decl)
	d, err := m.internal(decl)
	astTestMust(t, err)
	name, err := m.Symbols.String(m.declAt(d.slot).name)
	astTestMust(t, err)
	if name != "to_fp_unsigned" || m.declAt(d.slot).arity != 2 {
		t.Fatal("source unsigned conversion declaration", name)
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
	for _, tc := range []struct {
		input uint64
		rm    ASTHandle
		want  float64
	}{
		{0, rne, 0}, {255, rne, 255},
		{16777217, rne, 16777216},
		{16777217, rtp, 16777218},
		{4294967295, rne, 4294967296},
	} {
		bv, e := m.BitVectorNumeralUint64(tc.input, 32)
		astTestMust(t, e)
		got, found, e := m.RewriteFPAToFPUnsignedNumeral(decl, tc.rm, bv)
		astTestMust(t, e)
		if !found || got != makeFloat(tc.want) || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("unsigned conversion or cleanup", tc, found, got)
		}
	}
	symbolicBV, err := m.MkVar(0, bvSort)
	astTestMust(t, err)
	got, found, err := m.RewriteFPAToFPUnsignedNumeral(decl, rne, symbolicBV)
	astTestMust(t, err)
	if found || got != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("symbolic BV should refuse", found, got)
	}
	symbolicRM, err := m.MkVar(0, rmSort)
	astTestMust(t, err)
	zero, err := m.BitVectorNumeralUnsigned(0, 32)
	astTestMust(t, err)
	got, found, err = m.RewriteFPAToFPUnsignedNumeral(decl, symbolicRM, zero)
	astTestMust(t, err)
	if found || got != (ASTHandle{}) {
		t.Fatal("symbolic rounding mode should refuse", found, got)
	}
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	wrongParam, err := m.NewParameterAST(boolSort)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelParameter(&wrongParam)) }()
	for _, tc := range []struct {
		ps     []ASTParameter
		domain []ASTHandle
		part   string
	}{
		{nil, nil, "number of arguments"},
		{nil, []ASTHandle{boolSort, bvSort}, "RoundingMode"},
		{nil, []ASTHandle{rmSort, boolSort}, "bit-vector"},
		{nil, domain, "number of parameters"},
		{[]ASTParameter{eParam, wrongParam}, domain, "parameter type"},
	} {
		_, e := m.MkFamilyFuncDecl(family, FPAOpToFPUnsigned, tc.ps, tc.domain, ASTHandle{})
		if e == nil || !strings.Contains(e.Error(), tc.part) {
			t.Fatal("source validation order", tc.part, e)
		}
	}
}
