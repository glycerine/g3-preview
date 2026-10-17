package ast

import (
	"math"
	"strings"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestASTFPAToFPIEEEBVExactFieldsAndRefusals(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
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
	decl, err := m.MkFamilyFuncDecl(family, FPAOpToFP, ps, []ASTHandle{bvSort}, ASTHandle{})
	astTestMust(t, err)
	astTestPin(t, m, decl)
	d, err := m.internal(decl)
	astTestMust(t, err)
	name, err := m.Symbols.String(m.declAt(d.slot).name)
	astTestMust(t, err)
	if name != "to_fp" || m.declAt(d.slot).arity != 1 {
		t.Fatal("one-BV to_fp declaration", name)
	}
	makeFloat := func(bits uint32) ASTHandle {
		var v unumeral.ArbitraryFloat
		astTestMust(t, m.floatingPointPlugin.numeric.SetFloat(&v, 8, 24, math.Float32frombits(bits)))
		r, e := m.floatingPointPlugin.mkNumeral(&v, nil)
		astTestMust(t, e)
		astTestMust(t, m.floatingPointPlugin.numeric.Del(&v))
		return m.external(r)
	}
	for _, bits := range []uint32{
		0x00000000, 0x80000000, 0x3fc00000, 0xbf800000,
		0x7f800000, 0xff800000, 0x7fc00001, 0x00000001,
	} {
		bv, e := m.BitVectorNumeralUint64(uint64(bits), 32)
		astTestMust(t, e)
		got, found, e := m.RewriteFPAToFPIEEEBVNumeral(decl, bv)
		astTestMust(t, e)
		if !found || got != makeFloat(bits) || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatalf("IEEE field conversion or scoped cleanup: bits=%08x found=%v got=%v", bits, found, got)
		}
		if bits != 0x7fc00001 {
			roundTrip, yes, again, e := m.RewriteFPAToIEEEBVNumeral(got, false)
			astTestMust(t, e)
			if !yes || again || roundTrip != bv {
				t.Fatalf("IEEE round trip: bits=%08x yes=%v again=%v", bits, yes, again)
			}
		}
	}
	doubleBVSort, err := m.BitVectorSort(64)
	astTestMust(t, err)
	doubleE, err := m.NewParameterInt(11)
	astTestMust(t, err)
	doubleS, err := m.NewParameterInt(53)
	astTestMust(t, err)
	defer func() {
		astTestMust(t, m.DelParameter(&doubleS))
		astTestMust(t, m.DelParameter(&doubleE))
	}()
	doubleDecl, err := m.MkFamilyFuncDecl(family, FPAOpToFP, []ASTParameter{doubleE, doubleS}, []ASTHandle{doubleBVSort}, ASTHandle{})
	astTestMust(t, err)
	astTestPin(t, m, doubleDecl)
	for _, bits := range []uint64{0x3ff0000000000000, 0x8000000000000000, 0x7ff0000000000000, 0x0000000000000001} {
		bv, e := m.BitVectorNumeralUint64(bits, 64)
		astTestMust(t, e)
		got, found, e := m.RewriteFPAToFPIEEEBVNumeral(doubleDecl, bv)
		astTestMust(t, e)
		var wantValue unumeral.ArbitraryFloat
		astTestMust(t, m.floatingPointPlugin.numeric.SetDouble(&wantValue, 11, 53, math.Float64frombits(bits)))
		want, e := m.floatingPointPlugin.mkNumeral(&wantValue, nil)
		astTestMust(t, e)
		astTestMust(t, m.floatingPointPlugin.numeric.Del(&wantValue))
		if !found || got != m.external(want) || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatalf("64-bit IEEE field conversion: bits=%016x found=%v got=%v", bits, found, got)
		}
		roundTrip, yes, again, e := m.RewriteFPAToIEEEBVNumeral(got, false)
		astTestMust(t, e)
		if !yes || again || roundTrip != bv {
			t.Fatalf("64-bit IEEE round trip: bits=%016x yes=%v again=%v", bits, yes, again)
		}
	}
	variable, err := m.MkVar(0, bvSort)
	astTestMust(t, err)
	got, found, err := m.RewriteFPAToFPIEEEBVNumeral(decl, variable)
	astTestMust(t, err)
	if found || got != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("symbolic BV should refuse", found, got)
	}
	wrongSort, err := m.BitVectorSort(31)
	astTestMust(t, err)
	_, err = m.MkFamilyFuncDecl(family, FPAOpToFP, ps, []ASTHandle{wrongSort}, ASTHandle{})
	if err == nil || !strings.Contains(err.Error(), "ebits+sbits") {
		t.Fatal("one-BV width mismatch", err)
	}
	_, err = m.MkFamilyFuncDecl(family, FPAOpToFP, nil, []ASTHandle{bvSort}, ASTHandle{})
	if err == nil || !strings.Contains(err.Error(), "number of parameters") {
		t.Fatal("one-BV parameter count", err)
	}
}
