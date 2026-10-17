package ast

import "testing"

func TestASTFPAToFPThreeBVFieldsAndRefusal(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	signSort, err := m.BitVectorSort(2)
	astTestMust(t, err)
	expSort, err := m.BitVectorSort(8)
	astTestMust(t, err)
	sigSort, err := m.BitVectorSort(23)
	astTestMust(t, err)
	ignored, err := m.NewParameterInt(99)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelParameter(&ignored)) }()
	decl, err := m.MkFamilyFuncDecl(family, FPAOpToFP, []ASTParameter{ignored}, []ASTHandle{signSort, expSort, sigSort}, ASTHandle{})
	astTestMust(t, err)
	astTestPin(t, m, decl)
	d, err := m.internal(decl)
	astTestMust(t, err)
	info, err := m.funcInfo(m.declAt(d.slot).info)
	astTestMust(t, err)
	first, err := m.infoParameter(info, 0)
	astTestMust(t, err)
	second, err := m.infoParameter(info, 1)
	astTestMust(t, err)
	e, err := m.parameterOfKind(first, ASTParameterInt)
	astTestMust(t, err)
	s, err := m.parameterOfKind(second, ASTParameterInt)
	astTestMust(t, err)
	if e.integer != 8 || s.integer != 24 {
		t.Fatalf("derived FP format = (%d, %d)", e.integer, s.integer)
	}
	for _, tc := range []struct {
		bits uint32
		sign uint64
	}{
		{0x00000000, 0}, {0x80000000, 1}, {0x3fc00000, 0},
		{0xbf800000, 1}, {0x7f800000, 0}, {0xff800000, 1},
		{0x7fc00001, 0}, {0x00000001, 0},
		// The source interprets the sign as r1.is_one(), even for a 2-bit BV.
		{0x3f800000, 2},
	} {
		sign, e := m.BitVectorNumeralUint64(tc.sign, 2)
		astTestMust(t, e)
		exp, e := m.BitVectorNumeralUint64(uint64((tc.bits>>23)&0xff), 8)
		astTestMust(t, e)
		sig, e := m.BitVectorNumeralUint64(uint64(tc.bits&0x7fffff), 23)
		astTestMust(t, e)
		got, found, e := m.RewriteFPAToFPThreeBVNumerals(decl, sign, exp, sig)
		astTestMust(t, e)
		if !found || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatalf("three-BV numeral refused or leaked: %08x", tc.bits)
		}
		if tc.bits == 0x7fc00001 {
			continue
		}
		roundTrip, yes, again, e := m.RewriteFPAToIEEEBVNumeral(got, false)
		astTestMust(t, e)
		if !yes || again {
			t.Fatalf("reverse IEEE rewrite refused: %08x", tc.bits)
		}
		want, e := m.BitVectorNumeralUint64(uint64(tc.bits), 32)
		astTestMust(t, e)
		if roundTrip != want {
			t.Fatalf("three-BV round trip: %08x", tc.bits)
		}
	}
	sign, err := m.BitVectorNumeralUint64(0, 2)
	astTestMust(t, err)
	exp, err := m.BitVectorNumeralUint64(127, 8)
	astTestMust(t, err)
	symbolic, err := m.MkVar(0, sigSort)
	astTestMust(t, err)
	got, found, err := m.RewriteFPAToFPThreeBVNumerals(decl, sign, exp, symbolic)
	astTestMust(t, err)
	if found || got != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("symbolic significand should refuse", found, got)
	}
}
