package ast

import (
	"strings"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestASTFPAFMADeclarationNumericAndScopedBranches(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	rmSort, err := m.MkFamilySort(family, 1, nil)
	astTestMust(t, err)
	floatSort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	domain := []ASTHandle{rmSort, floatSort, floatSort, floatSort}
	decl, err := m.MkFamilyFuncDecl(family, FPAOpFMA, nil, domain, ASTHandle{})
	astTestMust(t, err)
	r, err := m.internal(decl)
	astTestMust(t, err)
	name, err := m.Symbols.String(m.declAt(r.slot).name)
	astTestMust(t, err)
	floatRef, err := m.internal(floatSort)
	astTestMust(t, err)
	if name != "fp.fma" || m.declAt(r.slot).arity != 4 || m.declAt(r.slot).rangeSort != floatRef {
		t.Fatal("fma declaration", name)
	}
	for _, invalid := range [][]ASTHandle{nil, {rmSort, floatSort}, {rmSort, floatSort, floatSort}, {rmSort, floatSort, floatSort, floatSort, floatSort}, {boolSort, floatSort, floatSort, floatSort}, {rmSort, boolSort, boolSort, boolSort}, {rmSort, floatSort, floatSort, boolSort}} {
		_, err := m.MkFamilyFuncDecl(family, FPAOpFMA, nil, invalid, ASTHandle{})
		if err == nil || !strings.Contains(err.Error(), "fused_ma") && !strings.Contains(err.Error(), "RoundingMode") && !strings.Contains(err.Error(), "FloatingPoint") {
			t.Fatal("invalid fma declaration", invalid, err)
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
	two, three, four, ten := makeValue(2), makeValue(3), makeValue(4), makeValue(10)
	rm, err := m.MkFamilyApp(family, astFPARMNearestEven, nil, nil)
	astTestMust(t, err)
	app, err := m.MkFamilyApp(family, FPAOpFMA, nil, []ASTHandle{rm, two, three, four})
	astTestMust(t, err)
	gotSort, err := m.GetSort(app)
	astTestMust(t, err)
	if gotSort != floatSort {
		t.Fatal("fma application range")
	}
	out, found, err := m.RewriteFPAFMANumerals(rm, two, three, four)
	astTestMust(t, err)
	if !found || out != ten || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("finite fma or scoped cleanup", found, out)
	}
	variableFloat, err := m.MkVar(0, floatSort)
	astTestMust(t, err)
	for _, values := range [][3]ASTHandle{{variableFloat, three, four}, {two, variableFloat, four}, {two, three, variableFloat}} {
		out, found, err = m.RewriteFPAFMANumerals(rm, values[0], values[1], values[2])
		astTestMust(t, err)
		if found || out != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("symbolic fma operand retained", values, found, out)
		}
	}
	variableRM, err := m.MkVar(0, rmSort)
	astTestMust(t, err)
	out, found, err = m.RewriteFPAFMANumerals(variableRM, two, three, four)
	astTestMust(t, err)
	if found || out != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("symbolic fma mode retained", found, out)
	}
}
