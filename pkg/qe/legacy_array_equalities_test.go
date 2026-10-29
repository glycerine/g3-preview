package qe

// Source: src/muz/spacer/spacer_qe_project.cpp
// array_project_eqs_util::find_arr_eqs.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestLegacyArrayProjectionFindsStoreEqualitiesButBlocksSelects(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p, err := newLegacyArrayProjection(m, f.r.config)
	fmMust(t, err)
	defer func() { fmMust(t, p.Close()) }()
	arraySort := f.own(m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{f.integer}), f.integer))
	a := f.constant("legacy-array-a", arraySort)
	b := f.constant("legacy-array-b", arraySort)
	zero := f.own(m.ArithmeticIntegerNumeralText("0"))
	one := f.own(m.ArithmeticIntegerNumeralText("1"))
	stored := f.own(m.MkFamilyApp(p.syntax.family, ast.ASTArrayOpStore, nil, []ast.ASTHandle{a, zero, one}))
	storeEquality := f.own(m.MkEq(stored, b))
	directEquality := f.own(m.MkEq(a, b))
	selected := f.own(m.MkFamilyApp(p.syntax.family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{stored, zero}))
	selectEquality := f.own(m.MkEq(selected, one))
	formula := f.own(m.MkAndArray([]ast.ASTHandle{storeEquality, directEquality, selectEquality}))
	result, err := p.findEqualities(a, formula)
	fmMust(t, err)
	defer func() { fmMust(t, result.Close()) }()
	if len(result.equalities) != 2 {
		t.Fatal("legacy relevant array equality count", len(result.equalities))
	}
	if !result.stores[stored] || !result.stores[storeEquality] || result.stores[selected] || result.stores[selectEquality] {
		t.Fatal("legacy store propagation marks", result.stores[stored], result.stores[storeEquality], result.stores[selected], result.stores[selectEquality])
	}
}
