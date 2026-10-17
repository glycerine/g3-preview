package ast

import (
	"strings"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestASTFPAExtremaDeclarationsAndNumericBranches(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	makeValue := func(n float64) ASTHandle {
		var value unumeral.ArbitraryFloat
		astTestMust(t, m.floatingPointPlugin.numeric.SetDouble(&value, 8, 24, n))
		ref, e := m.floatingPointPlugin.mkNumeral(&value, nil)
		astTestMust(t, e)
		astTestMust(t, m.floatingPointPlugin.numeric.Del(&value))
		return m.external(ref)
	}
	three, five := makeValue(3), makeValue(5)
	for _, tc := range []struct {
		kind int32
		name string
		want ASTHandle
	}{{FPAOpMin, "fp.min", three}, {FPAOpMax, "fp.max", five}} {
		decl, e := m.MkFamilyFuncDecl(family, tc.kind, nil, []ASTHandle{sort, sort}, ASTHandle{})
		astTestMust(t, e)
		r, e := m.internal(decl)
		astTestMust(t, e)
		name, e := m.Symbols.String(m.declAt(r.slot).name)
		astTestMust(t, e)
		sortRef, e := m.internal(sort)
		astTestMust(t, e)
		if name != tc.name || m.declAt(r.slot).arity != 2 || m.declAt(r.slot).rangeSort != sortRef {
			t.Fatal("extremum declaration", tc.kind, name)
		}
		app, e := m.MkFamilyApp(family, tc.kind, nil, []ASTHandle{three, five})
		astTestMust(t, e)
		gotSort, e := m.GetSort(app)
		astTestMust(t, e)
		if gotSort != sort {
			t.Fatal("extremum application sort", tc.kind)
		}
		for _, domain := range [][]ASTHandle{nil, {sort}, {sort, sort, sort}, {sort, boolSort}} {
			_, e := m.MkFamilyFuncDecl(family, tc.kind, nil, domain, ASTHandle{})
			if e == nil || !strings.Contains(e.Error(), "floating point") && !strings.Contains(e.Error(), "FloatingPoint") {
				t.Fatal("invalid extremum declaration", tc.kind, domain, e)
			}
		}
		out, found, e := m.RewriteFPAExtremumNumerals(three, five, tc.kind)
		astTestMust(t, e)
		if !found || out != tc.want || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("finite extremum or scoped cleanup", tc.kind, found, out)
		}
	}
	parameter, err := m.NewParameterAST(sort)
	astTestMust(t, err)
	positiveZero, err := m.MkFamilyApp(family, astFPAPlusZero, []ASTParameter{parameter}, nil)
	astTestMust(t, err)
	negativeZero, err := m.MkFamilyApp(family, astFPAMinusZero, []ASTParameter{parameter}, nil)
	astTestMust(t, err)
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	for _, kind := range []int32{FPAOpMin, FPAOpMax} {
		for _, operands := range [][2]ASTHandle{{positiveZero, negativeZero}, {negativeZero, positiveZero}, {variable, three}, {three, variable}} {
			out, found, e := m.RewriteFPAExtremumNumerals(operands[0], operands[1], kind)
			astTestMust(t, e)
			if found || out != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
				t.Fatal("unreduced extremum or scoped cleanup", kind, operands, found, out)
			}
		}
	}
}
