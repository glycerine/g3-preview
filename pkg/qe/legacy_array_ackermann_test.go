package qe

// Source: src/muz/spacer/spacer_qe_project.cpp
// array_project_selects_util.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
)

func TestLegacyArrayProjectionAckermannizesEqualModelIndices(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p, err := newLegacyArrayProjection(m, f.r.config)
	fmMust(t, err)
	defer func() { fmMust(t, p.Close()) }()
	arraySort := f.own(m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{f.integer}), f.integer))
	a := f.constant("legacy-ack-array", arraySort)
	i := f.constant("legacy-ack-i", f.integer)
	j := f.constant("legacy-ack-j", f.integer)
	selectI := f.own(m.MkFamilyApp(p.syntax.family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{a, i}))
	selectJ := f.own(m.MkFamilyApp(p.syntax.family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{a, j}))
	formula := f.own(m.MkEq(selectI, selectJ))
	one := f.own(m.ArithmeticIntegerNumeralText("1"))
	seven := f.own(m.ArithmeticIntegerNumeralText("7"))
	base := f.own(m.MkConstArray(arraySort, seven))
	md, err := model.NewModel(m)
	fmMust(t, err)
	defer func() { fmMust(t, md.Close()) }()
	for variable, value := range map[ast.ASTHandle]ast.ASTHandle{i: one, j: one, a: base} {
		decl, err := m.GetDecl(variable)
		fmMust(t, err)
		fmMust(t, md.Core().RegisterConstant(decl, value))
	}
	result, err := p.projectSelects(md.Core(), []ast.ASTHandle{a}, formula)
	fmMust(t, err)
	defer func() { fmMust(t, result.Close()) }()
	if len(result.variables) != 1 {
		t.Fatal("select equivalence class count", len(result.variables))
	}
	constant, err := m.RefGet(result.variables[0])
	fmMust(t, err)
	decl, err := m.GetDecl(constant)
	fmMust(t, err)
	value, err := md.Core().ConstantInterpretation(decl)
	fmMust(t, err)
	text, _, numeral, err := m.ArithmeticNumeralText(value)
	fmMust(t, err)
	if !numeral || text != "7" {
		t.Fatal("select value constant model interpretation", text, numeral)
	}
	handle, err := m.RefGet(result.formula)
	fmMust(t, err)
	contains, err := ast.NewContainsApp(m, a)
	fmMust(t, err)
	defer func() { fmMust(t, contains.Close()) }()
	hasArray, err := contains.Check(handle)
	fmMust(t, err)
	if hasArray {
		t.Fatal("ackermannized formula retains projected array")
	}
}
