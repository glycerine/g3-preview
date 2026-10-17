package ast

import (
	"math"
	"strings"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestASTFPAToBVSourceBoundsRoundingAndUnspecified(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	fpSort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	rmSort, err := m.MkFamilySort(family, 1, nil)
	astTestMust(t, err)
	width, err := m.NewParameterInt(4)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelParameter(&width)) }()
	declarations := [2]ASTHandle{}
	for i, kind := range [...]int32{FPAOpToUBV, FPAOpToSBV} {
		declarations[i], err = m.MkFamilyFuncDecl(family, kind, []ASTParameter{width}, []ASTHandle{rmSort, fpSort}, ASTHandle{})
		astTestMust(t, err)
		d, e := m.internal(declarations[i])
		astTestMust(t, e)
		gotWidth, e := m.bitVectorSortSize(m.declAt(d.slot).rangeSort, nil)
		astTestMust(t, e)
		if gotWidth != 4 {
			t.Fatal("conversion width", gotWidth)
		}
	}
	rne, err := m.MkFamilyApp(family, astFPARMNearestEven, nil, nil)
	astTestMust(t, err)
	rtz, err := m.MkFamilyApp(family, astFPARMTowardZero, nil, nil)
	astTestMust(t, err)
	makeValue := func(n float64) ASTHandle {
		var v unumeral.ArbitraryFloat
		astTestMust(t, m.floatingPointPlugin.numeric.SetDouble(&v, 8, 24, n))
		ref, e := m.floatingPointPlugin.mkNumeral(&v, nil)
		astTestMust(t, e)
		astTestMust(t, m.floatingPointPlugin.numeric.Del(&v))
		return m.external(ref)
	}
	for _, tc := range []struct {
		signed bool
		input  float64
		rm     ASTHandle
		want   uint64
		found  bool
	}{
		{false, 15, rne, 15, true}, {false, 15.5, rne, 0, false},
		{false, -1, rne, 0, false}, {false, 3.9, rtz, 3, true},
		{true, -8, rne, 8, true}, {true, 7, rne, 7, true},
		{true, 7.5, rne, 0, false}, {true, -8.5, rne, 8, true},
		{true, -9, rne, 0, false},
	} {
		decl := declarations[0]
		if tc.signed {
			decl = declarations[1]
		}
		value := makeValue(tc.input)
		got, found, e := m.RewriteFPAToBVNumeral(decl, tc.rm, value, false)
		astTestMust(t, e)
		if found != tc.found || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("numeric conversion or cleanup", tc, found)
		}
		if found {
			want, e := m.BitVectorNumeralUint64(tc.want, 4)
			astTestMust(t, e)
			if got != want {
				t.Fatal("numeric bits", tc, got, want)
			}
		} else if got != (ASTHandle{}) {
			t.Fatal("refused conversion published output", tc)
		}
	}
	zero, err := m.BitVectorNumeralUnsigned(0, 4)
	astTestMust(t, err)
	for _, n := range []float64{math.NaN(), math.Inf(1), 16} {
		value := makeValue(n)
		got, found, e := m.RewriteFPAToBVNumeral(declarations[0], rne, value, true)
		astTestMust(t, e)
		if !found || got != zero {
			t.Fatal("hardware unspecified", n, got, found)
		}
	}
	variable, err := m.MkVar(0, fpSort)
	astTestMust(t, err)
	got, found, err := m.RewriteFPAToBVNumeral(declarations[0], rne, variable, true)
	astTestMust(t, err)
	if found || got != (ASTHandle{}) {
		t.Fatal("symbolic argument should refuse")
	}
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	for _, tc := range []struct {
		ps     []ASTParameter
		domain []ASTHandle
		part   string
	}{
		{nil, []ASTHandle{rmSort, fpSort}, "number of parameters"},
		{[]ASTParameter{width}, nil, "number of arguments"},
		{[]ASTParameter{width}, []ASTHandle{boolSort, fpSort}, "RoundingMode"},
		{[]ASTParameter{width}, []ASTHandle{rmSort, boolSort}, "FloatingPoint"},
	} {
		_, e := m.MkFamilyFuncDecl(family, FPAOpToUBV, tc.ps, tc.domain, ASTHandle{})
		if e == nil || !strings.Contains(e.Error(), tc.part) {
			t.Fatal("source declaration validation", tc.part, e)
		}
	}
}
