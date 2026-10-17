package ast

import (
	"strings"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestASTFPARemainderDeclarationAndApplication(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	parameter, err := m.NewParameterAST(sort)
	astTestMust(t, err)
	zero, err := m.MkFamilyApp(family, astFPAPlusZero, []ASTParameter{parameter}, nil)
	astTestMust(t, err)
	decl, err := m.MkFamilyFuncDecl(family, astFPARem, nil, []ASTHandle{sort, sort}, ASTHandle{})
	astTestMust(t, err)
	r, err := m.internal(decl)
	astTestMust(t, err)
	name, err := m.Symbols.String(m.declAt(r.slot).name)
	astTestMust(t, err)
	sortRef, err := m.internal(sort)
	astTestMust(t, err)
	if name != "fp.rem" || m.declAt(r.slot).arity != 2 || m.declAt(r.slot).rangeSort != sortRef {
		t.Fatal("fp.rem declaration", name)
	}
	app, err := m.MkFamilyApp(family, astFPARem, nil, []ASTHandle{zero, zero})
	astTestMust(t, err)
	gotSort, err := m.GetSort(app)
	astTestMust(t, err)
	if gotSort != sort {
		t.Fatal("fp.rem application sort")
	}
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	for _, domain := range [][]ASTHandle{nil, {sort}, {sort, sort, sort}, {sort, boolSort}} {
		_, err := m.MkFamilyFuncDecl(family, astFPARem, nil, domain, ASTHandle{})
		if err == nil || !strings.Contains(err.Error(), "floating point") && !strings.Contains(err.Error(), "FloatingPoint") {
			t.Fatal("fp.rem invalid domain", domain, err)
		}
	}
}

func TestASTFPARemainderFiniteNumeralAndScopedCleanup(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	_, err = m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	p := m.floatingPointPlugin
	makeValue := func(n float64) ASTHandle {
		var value unumeral.ArbitraryFloat
		astTestMust(t, p.numeric.SetDouble(&value, 8, 24, n))
		ref, e := p.mkNumeral(&value, nil)
		astTestMust(t, e)
		astTestMust(t, p.numeric.Del(&value))
		return m.external(ref)
	}
	seven, two, negativeOne := makeValue(7), makeValue(2), makeValue(-1)
	out, found, err := m.RewriteFPARemainderNumerals(seven, two)
	astTestMust(t, err)
	if !found || out != negativeOne || p.numeralProbeFrames != nil {
		t.Fatal("finite fp.rem or scoped cleanup", found, out, negativeOne)
	}
	sort, err := m.GetSort(seven)
	astTestMust(t, err)
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	out, found, err = m.RewriteFPARemainderNumerals(variable, two)
	astTestMust(t, err)
	if found || out != (ASTHandle{}) || p.numeralProbeFrames != nil {
		t.Fatal("first symbolic fp.rem probe retained", found, out)
	}
	out, found, err = m.RewriteFPARemainderNumerals(seven, variable)
	astTestMust(t, err)
	if found || out != (ASTHandle{}) || p.numeralProbeFrames != nil {
		t.Fatal("second symbolic fp.rem probe retained", found, out)
	}
}
