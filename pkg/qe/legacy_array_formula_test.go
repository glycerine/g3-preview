package qe

// Source: src/muz/spacer/spacer_qe_project.cpp array_project.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
)

func TestLegacyArrayProjectionCoordinatesEqualityAndSelects(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p, err := newLegacyArrayProjection(m, f.r.config)
	fmMust(t, err)
	defer func() { fmMust(t, p.Close()) }()
	arraySort := f.own(m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{f.integer}), f.integer))
	a := f.constant("legacy-array-formula-a", arraySort)
	b := f.constant("legacy-array-formula-b", arraySort)
	i := f.constant("legacy-array-formula-i", f.integer)
	one := f.own(m.ArithmeticIntegerNumeralText("1"))
	seven := f.own(m.ArithmeticIntegerNumeralText("7"))
	equality := f.own(m.MkEq(a, b))
	selected := f.own(m.MkFamilyApp(p.syntax.family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{a, i}))
	valueEquality := f.own(m.MkEq(selected, seven))
	formula := f.own(m.MkAnd(equality, valueEquality))
	base := f.own(m.MkConstArray(arraySort, seven))
	md, err := model.NewModel(m)
	fmMust(t, err)
	defer func() { fmMust(t, md.Close()) }()
	for variable, value := range map[ast.ASTHandle]ast.ASTHandle{a: base, b: base, i: one} {
		decl, err := m.GetDecl(variable)
		fmMust(t, err)
		fmMust(t, md.Core().RegisterConstant(decl, value))
	}
	result, err := p.projectFormula(md.Core(), []ast.ASTHandle{a}, formula, false)
	fmMust(t, err)
	defer func() { fmMust(t, result.Close()) }()
	handle, err := m.RefGet(result.formula)
	fmMust(t, err)
	contains, err := ast.NewContainsApp(m, a)
	fmMust(t, err)
	defer func() { fmMust(t, contains.Close()) }()
	remaining, err := contains.Check(handle)
	fmMust(t, err)
	if remaining {
		t.Fatal("legacy array coordinator retains projected variable")
	}
}
