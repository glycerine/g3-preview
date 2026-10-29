package qe

// Sources: src/muz/spacer/spacer_legacy_mbp.cpp;
// src/muz/spacer/spacer_util.cpp qe_project legacy path.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
)

func TestLegacyProjectionCombinesBooleanArrayAndArithmetic(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p, err := NewLegacyProjection(m, f.r.config)
	fmMust(t, err)
	defer func() { fmMust(t, p.Close()) }()
	b := f.constant("legacy-combined-bool", f.boolean)
	x := f.constant("legacy-combined-int", f.integer)
	arraySort := f.own(m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{f.integer}), f.integer))
	a := f.constant("legacy-combined-array-a", arraySort)
	c := f.constant("legacy-combined-array-c", arraySort)
	zero := f.own(m.ArithmeticIntegerNumeralText("0"))
	one := f.own(m.ArithmeticIntegerNumeralText("1"))
	three := f.own(m.ArithmeticIntegerNumeralText("3"))
	truth := f.own(m.MkTrue())
	base := f.own(m.MkConstArray(arraySort, zero))
	arrayEquality := f.own(m.MkEq(a, c))
	lower := f.own(m.MkArithmeticGE(x, zero))
	upper := f.own(m.MkArithmeticLE(x, three))
	formula := f.own(m.MkAndArray([]ast.ASTHandle{b, arrayEquality, lower, upper}))
	md, err := model.NewModel(m)
	fmMust(t, err)
	defer func() { fmMust(t, md.Close()) }()
	for variable, value := range map[ast.ASTHandle]ast.ASTHandle{b: truth, x: one, a: base, c: base} {
		decl, err := m.GetDecl(variable)
		fmMust(t, err)
		fmMust(t, md.Core().RegisterConstant(decl, value))
	}
	result, err := p.Project(md.Core(), []ast.ASTHandle{b, a, x}, formula, false, true)
	fmMust(t, err)
	defer func() { fmMust(t, result.Close()) }()
	remaining, err := result.Variables()
	fmMust(t, err)
	if len(remaining) != 0 {
		t.Fatal("combined legacy projection retained variables", len(remaining))
	}
	handle, err := result.Formula()
	fmMust(t, err)
	for _, variable := range []ast.ASTHandle{b, a, x} {
		contains, err := ast.NewContainsApp(m, variable)
		fmMust(t, err)
		found, err := contains.Check(handle)
		fmMust(t, err)
		fmMust(t, contains.Close())
		if found {
			t.Fatal("combined legacy projection retains input variable")
		}
	}
}
