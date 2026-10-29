package qe

// Source: src/muz/spacer/spacer_qe_project.cpp
// array_project_eqs_util no-substitution path.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
)

func TestLegacyArrayProjectionReplacesUnsolvedEqualityPolarity(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p, err := newLegacyArrayProjection(m, f.r.config)
	fmMust(t, err)
	defer func() { fmMust(t, p.Close()) }()
	arraySort := f.own(m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{f.integer}), f.integer))
	a := f.constant("legacy-polarity-a", arraySort)
	b := f.constant("legacy-polarity-b", arraySort)
	i := f.constant("legacy-polarity-i", f.integer)
	zero := f.own(m.ArithmeticIntegerNumeralText("0"))
	one := f.own(m.ArithmeticIntegerNumeralText("1"))
	left := f.own(m.MkConstArray(arraySort, zero))
	right := f.own(m.MkConstArray(arraySort, one))
	arrayEquality := f.own(m.MkEq(a, b))
	trueEquality := f.own(m.MkEq(i, i))
	formula := f.own(m.MkOr(arrayEquality, trueEquality))
	md, err := model.NewModel(m)
	fmMust(t, err)
	defer func() { fmMust(t, md.Close()) }()
	for variable, value := range map[ast.ASTHandle]ast.ASTHandle{a: left, b: right, i: zero} {
		decl, err := m.GetDecl(variable)
		fmMust(t, err)
		fmMust(t, md.Core().RegisterConstant(decl, value))
	}
	result, err := p.replaceEqualityPolarities(md.Core(), a, formula)
	fmMust(t, err)
	defer func() { fmMust(t, result.Close()) }()
	handle, err := m.RefGet(result.formula)
	fmMust(t, err)
	if !m.IsTrue(handle) {
		t.Fatal("model-polarity replacement did not simplify selected disjunction")
	}
}
