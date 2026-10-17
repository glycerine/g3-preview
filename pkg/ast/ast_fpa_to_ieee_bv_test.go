package ast

import (
	"math"
	"strings"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestASTFPAToIEEEBVDeclarationFiniteAndNaNBranches(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	decl, err := m.MkFamilyFuncDecl(family, FPAOpToIEEEBV, nil, []ASTHandle{sort}, ASTHandle{})
	astTestMust(t, err)
	r, err := m.internal(decl)
	astTestMust(t, err)
	name, err := m.Symbols.String(m.declAt(r.slot).name)
	astTestMust(t, err)
	width, err := m.bitVectorSortSize(m.declAt(r.slot).rangeSort, nil)
	astTestMust(t, err)
	if name != "fp.to_ieee_bv" || m.declAt(r.slot).arity != 1 || width != 32 {
		t.Fatal("fp.to_ieee_bv declaration", name, width)
	}
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	for _, invalid := range [][]ASTHandle{nil, {sort, sort}, {boolSort}} {
		_, err := m.MkFamilyFuncDecl(family, FPAOpToIEEEBV, nil, invalid, ASTHandle{})
		if err == nil || !strings.Contains(err.Error(), "fp.to_ieee_bv") && !strings.Contains(err.Error(), "FloatingPoint") {
			t.Fatal("invalid fp.to_ieee_bv declaration", invalid, err)
		}
	}
	makeValue := func(n float64) ASTHandle {
		var value unumeral.ArbitraryFloat
		astTestMust(t, m.floatingPointPlugin.numeric.SetDouble(&value, 8, 24, n))
		ref, e := m.floatingPointPlugin.mkNumeral(&value, nil)
		astTestMust(t, e)
		astTestMust(t, m.floatingPointPlugin.numeric.Del(&value))
		return m.external(ref)
	}
	for _, tc := range []struct {
		input float64
		bits  uint64
	}{{1.5, 0x3fc00000}, {-1.5, 0xbfc00000}, {math.Copysign(0, -1), 0x80000000}, {math.Inf(1), 0x7f800000}} {
		value := makeValue(tc.input)
		want, e := m.BitVectorNumeralUint64(tc.bits, 32)
		astTestMust(t, e)
		out, found, rewriteOne, e := m.RewriteFPAToIEEEBVNumeral(value, false)
		astTestMust(t, e)
		if !found || rewriteOne || out != want || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("finite IEEE BV or scoped cleanup", tc, found, rewriteOne, out, want)
		}
	}
	parameter, err := m.NewParameterAST(sort)
	astTestMust(t, err)
	nan, err := m.MkFamilyApp(family, astFPANaN, []ASTParameter{parameter}, nil)
	astTestMust(t, err)
	out, found, rewriteOne, err := m.RewriteFPAToIEEEBVNumeral(nan, false)
	astTestMust(t, err)
	if found || rewriteOne || out != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("default NaN conversion", found, rewriteOne, out)
	}
	out, found, rewriteOne, err = m.RewriteFPAToIEEEBVNumeral(nan, true)
	astTestMust(t, err)
	if !found || !rewriteOne || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("hardware NaN conversion", found, rewriteOne, out)
	}
	kind, err := m.GetDeclKind(out)
	astTestMust(t, err)
	if kind != 37 {
		t.Fatal("hardware NaN must remain a concat rewrite", kind)
	}
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	out, found, rewriteOne, err = m.RewriteFPAToIEEEBVNumeral(variable, true)
	astTestMust(t, err)
	if found || rewriteOne || out != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("symbolic IEEE BV retained", found, rewriteOne, out)
	}
}
