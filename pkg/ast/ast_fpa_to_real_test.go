package ast

import (
	"math"
	"strings"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestASTFPAToRealDeclarationFiniteAndUnspecifiedBranches(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	decl, err := m.MkFamilyFuncDecl(family, FPAOpToReal, nil, []ASTHandle{sort}, ASTHandle{})
	astTestMust(t, err)
	r, err := m.internal(decl)
	astTestMust(t, err)
	name, err := m.Symbols.String(m.declAt(r.slot).name)
	astTestMust(t, err)
	if name != "fp.to_real" || m.declAt(r.slot).arity != 1 || m.declAt(r.slot).rangeSort != m.floatingPointPlugin.real {
		t.Fatal("fp.to_real declaration", name)
	}
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	for _, invalid := range [][]ASTHandle{nil, {sort, sort}, {boolSort}} {
		_, err := m.MkFamilyFuncDecl(family, FPAOpToReal, nil, invalid, ASTHandle{})
		if err == nil || !strings.Contains(err.Error(), "fp.to_real") && !strings.Contains(err.Error(), "FloatingPoint") {
			t.Fatal("invalid fp.to_real domain", invalid, err)
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
	q, err := m.numerals.NewValue()
	astTestMust(t, err)
	defer func() { astTestMust(t, m.numerals.Del(&q)) }()
	for _, tc := range []struct {
		input float64
		n     int64
	}{{1.5, 3}, {-1.5, -3}, {0, 0}, {math.Copysign(0, -1), 0}} {
		value := makeValue(tc.input)
		astTestMust(t, m.numerals.SetFraction(&q, tc.n, 2))
		want, e := m.ArithmeticNumeral(&q, false)
		astTestMust(t, e)
		out, found, e := m.RewriteFPAToRealNumeral(value, false)
		astTestMust(t, e)
		if !found || out != want || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("fp.to_real finite conversion or cleanup", tc, found, out, want)
		}
	}
	parameter, err := m.NewParameterAST(sort)
	astTestMust(t, err)
	for _, kind := range []int32{astFPANaN, astFPAPlusInf, astFPAMinusInf} {
		value, e := m.MkFamilyApp(family, kind, []ASTParameter{parameter}, nil)
		astTestMust(t, e)
		out, found, e := m.RewriteFPAToRealNumeral(value, false)
		astTestMust(t, e)
		if found || out != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("default unspecified conversion", kind, found, out)
		}
		out, found, e = m.RewriteFPAToRealNumeral(value, true)
		astTestMust(t, e)
		zero, e := m.GetSomeValue(m.external(m.floatingPointPlugin.real))
		astTestMust(t, e)
		if !found || out != zero || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("hardware unspecified zero conversion", kind, found, out, zero)
		}
	}
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	out, found, err := m.RewriteFPAToRealNumeral(variable, true)
	astTestMust(t, err)
	if found || out != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("symbolic fp.to_real retained", found, out)
	}
}
