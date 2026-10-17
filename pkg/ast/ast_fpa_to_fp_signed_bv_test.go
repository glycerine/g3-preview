package ast

import (
	"strings"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestASTFPAToFPSignedBVSourceNormalizationAndRounding(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	rmSort, err := m.MkFamilySort(family, 1, nil)
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
	declarations := map[uint32]ASTHandle{}
	for _, bits := range []uint32{1, 8, 32, 65} {
		bvSort, e := m.BitVectorSort(bits)
		astTestMust(t, e)
		decl, e := m.MkFamilyFuncDecl(family, FPAOpToFP, ps, []ASTHandle{rmSort, bvSort}, ASTHandle{})
		astTestMust(t, e)
		astTestPin(t, m, decl)
		declarations[bits] = decl
		d, e := m.internal(decl)
		astTestMust(t, e)
		name, e := m.Symbols.String(m.declAt(d.slot).name)
		astTestMust(t, e)
		if name != "to_fp" || m.declAt(d.slot).arity != 2 {
			t.Fatal("signed BV declaration", bits, name)
		}
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
		bits  uint32
		input uint64
		rm    ASTHandle
		want  float64
	}{
		{1, 0, rne, 0}, {1, 1, rne, -1},
		{8, 0, rne, 0}, {8, 127, rne, 127},
		{8, 128, rne, -128}, {8, 255, rne, -1},
		{32, 0x7fffffff, rne, 2147483648},
		{32, 0x80000000, rne, -2147483648},
		{32, 0x80000001, rne, -2147483648},
		{32, 0x80000001, rtp, -2147483520},
		{32, 0xffffffff, rne, -1},
	} {
		bv, e := m.BitVectorNumeralUint64(tc.input, tc.bits)
		astTestMust(t, e)
		got, found, e := m.RewriteFPAToFPSignedBVNumeral(declarations[tc.bits], tc.rm, bv)
		astTestMust(t, e)
		if !found || got != makeFloat(tc.want) || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("signed BV conversion or scoped cleanup", tc, found, got)
		}
	}
	highBit, err := m.numerals.PowerOfTwo(64)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.numerals.Del(&highBit)) }()
	wideBV, err := m.BitVectorNumeral(&highBit, 65)
	astTestMust(t, err)
	wide, found, err := m.RewriteFPAToFPSignedBVNumeral(declarations[65], rne, wideBV)
	astTestMust(t, err)
	if !found || wide != makeFloat(-18446744073709551616) || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("65-bit signed high-bit conversion", found, wide)
	}
	bvSort, err := m.BitVectorSort(8)
	astTestMust(t, err)
	variable, err := m.MkVar(0, bvSort)
	astTestMust(t, err)
	got, found, err := m.RewriteFPAToFPSignedBVNumeral(declarations[8], rne, variable)
	astTestMust(t, err)
	if found || got != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("symbolic signed BV should refuse", found, got)
	}
	symbolicRM, err := m.MkVar(0, rmSort)
	astTestMust(t, err)
	zero, err := m.BitVectorNumeralUnsigned(0, 8)
	astTestMust(t, err)
	got, found, err = m.RewriteFPAToFPSignedBVNumeral(declarations[8], symbolicRM, zero)
	astTestMust(t, err)
	if found || got != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("symbolic rounding mode should refuse", found, got)
	}
	_, err = m.MkFamilyFuncDecl(family, FPAOpToFP, nil, []ASTHandle{rmSort, bvSort}, ASTHandle{})
	if err == nil || !strings.Contains(err.Error(), "number of parameters") {
		t.Fatal("source parameter validation", err)
	}
}
