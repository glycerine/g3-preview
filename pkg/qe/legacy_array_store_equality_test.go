package qe

// Source: src/muz/spacer/spacer_qe_project.cpp
// array_project_eqs_util store-chain path.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
)

func TestLegacyArrayProjectionSubstitutesStoreEquality(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p, err := newLegacyArrayProjection(m, f.r.config)
	fmMust(t, err)
	defer func() { fmMust(t, p.Close()) }()
	arraySort := f.own(m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{f.integer}), f.integer))
	a := f.constant("legacy-store-a", arraySort)
	b := f.constant("legacy-store-b", arraySort)
	i := f.constant("legacy-store-i", f.integer)
	one := f.own(m.ArithmeticIntegerNumeralText("1"))
	seven := f.own(m.ArithmeticIntegerNumeralText("7"))
	zero := f.own(m.ArithmeticIntegerNumeralText("0"))
	storedA := f.own(m.MkFamilyApp(p.syntax.family, ast.ASTArrayOpStore, nil, []ast.ASTHandle{a, i, seven}))
	storeEquality := f.own(m.MkEq(storedA, b))
	base := f.own(m.MkConstArray(arraySort, zero))
	storedBase := f.own(m.MkFamilyApp(p.syntax.family, ast.ASTArrayOpStore, nil, []ast.ASTHandle{base, i, seven}))
	md, err := model.NewModel(m)
	fmMust(t, err)
	defer func() { fmMust(t, md.Close()) }()
	for variable, value := range map[ast.ASTHandle]ast.ASTHandle{a: base, b: storedBase, i: one} {
		decl, err := m.GetDecl(variable)
		fmMust(t, err)
		fmMust(t, md.Core().RegisterConstant(decl, value))
	}
	result, eliminated, err := p.projectStoreEquality(md.Core(), a, storeEquality)
	fmMust(t, err)
	if !eliminated {
		t.Fatal("store-chain array equality was not eliminated")
	}
	defer func() { fmMust(t, result.Close()) }()
	if len(result.variables) != 1 {
		t.Fatal("store-chain diff constant count", len(result.variables))
	}
	handle, err := m.RefGet(result.formula)
	fmMust(t, err)
	contains, err := ast.NewContainsApp(m, a)
	fmMust(t, err)
	defer func() { fmMust(t, contains.Close()) }()
	remaining, err := contains.Check(handle)
	fmMust(t, err)
	if remaining {
		t.Fatal("store-chain substitution retains projected variable")
	}
}
