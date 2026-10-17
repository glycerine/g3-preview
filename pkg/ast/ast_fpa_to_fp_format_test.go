package ast

import (
	"math"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestASTFPAToFPFormatRoundingSpecialsAndRefusals(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	rmSort, err := m.MkFamilySort(family, 1, nil)
	astTestMust(t, err)
	inputSort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	eParam, err := m.NewParameterInt(5)
	astTestMust(t, err)
	sParam, err := m.NewParameterInt(11)
	astTestMust(t, err)
	defer func() {
		astTestMust(t, m.DelParameter(&sParam))
		astTestMust(t, m.DelParameter(&eParam))
	}()
	decl, err := m.MkFamilyFuncDecl(family, FPAOpToFP, []ASTParameter{eParam, sParam}, []ASTHandle{rmSort, inputSort}, ASTHandle{})
	astTestMust(t, err)
	astTestPin(t, m, decl)
	d, err := m.internal(decl)
	astTestMust(t, err)
	rangeSort := m.declAt(d.slot).rangeSort
	rangeInfo, err := m.sortInfo(m.sortAt(rangeSort.slot).info)
	astTestMust(t, err)
	e, err := m.infoParameter(rangeInfo, 0)
	astTestMust(t, err)
	s, err := m.infoParameter(rangeInfo, 1)
	astTestMust(t, err)
	eWidth, err := m.parameterOfKind(e, ASTParameterInt)
	astTestMust(t, err)
	sWidth, err := m.parameterOfKind(s, ASTParameterInt)
	astTestMust(t, err)
	if eWidth.integer != 5 || sWidth.integer != 11 {
		t.Fatal("to_fp target format", eWidth.integer, sWidth.integer)
	}
	rne, err := m.MkFamilyApp(family, astFPARMNearestEven, nil, nil)
	astTestMust(t, err)
	astTestPin(t, m, rne)
	rtp, err := m.MkFamilyApp(family, astFPARMTowardPositive, nil, nil)
	astTestMust(t, err)
	astTestPin(t, m, rtp)
	makeFloat := func(ebits, sbits uint32, n float64) ASTHandle {
		var v unumeral.ArbitraryFloat
		astTestMust(t, m.floatingPointPlugin.numeric.SetDouble(&v, ebits, sbits, n))
		r, e := m.floatingPointPlugin.mkNumeral(&v, nil)
		astTestMust(t, e)
		astTestMust(t, m.floatingPointPlugin.numeric.Del(&v))
		return m.external(r)
	}
	for _, tc := range []struct {
		input float64
		rm    ASTHandle
		want  float64
	}{
		{1.00048828125, rne, 1},
		{1.00048828125, rtp, 1.0009765625},
		{-1.00048828125, rtp, -1},
		{math.Copysign(0, -1), rne, math.Copysign(0, -1)},
		{math.Inf(1), rne, math.Inf(1)},
		{math.Inf(-1), rne, math.Inf(-1)},
		{math.NaN(), rne, math.NaN()},
	} {
		input := makeFloat(8, 24, tc.input)
		got, found, e := m.RewriteFPAToFPFormatNumeral(decl, tc.rm, input)
		astTestMust(t, e)
		if !found || got != makeFloat(5, 11, tc.want) || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("format conversion or scoped cleanup", tc, found, got)
		}
	}
	symbolicValue, err := m.MkVar(0, inputSort)
	astTestMust(t, err)
	got, found, err := m.RewriteFPAToFPFormatNumeral(decl, rne, symbolicValue)
	astTestMust(t, err)
	if found || got != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("symbolic FP should refuse", found, got)
	}
	symbolicRM, err := m.MkVar(0, rmSort)
	astTestMust(t, err)
	zero := makeFloat(8, 24, 0)
	got, found, err = m.RewriteFPAToFPFormatNumeral(decl, symbolicRM, zero)
	astTestMust(t, err)
	if found || got != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("symbolic RM should refuse", found, got)
	}
}
