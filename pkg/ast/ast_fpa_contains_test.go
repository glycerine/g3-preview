// Source: src/ast/fpa_decl_plugin.cpp:980-1037.
package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTContainsFloatingPointTraversesEverySourceNodeShape(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)

	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	floating, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	rounding, err := m.MkFamilySort(family, 1, nil)
	astTestMust(t, err)

	check := func(name string, h ASTHandle, want bool) {
		t.Helper()
		got, e := m.ContainsFloatingPoint(h)
		astTestMust(t, e)
		if got != want {
			t.Fatalf("%s: got %v want %v", name, got, want)
		}
	}
	check("plain sort", boolean, false)
	check("floating sort", floating, true)
	check("rounding sort", rounding, true)

	variable, err := m.MkVar(0, floating)
	astTestMust(t, err)
	check("variable sort", variable, true)

	parameter, err := m.NewParameterAST(floating)
	astTestMust(t, err)
	parameterizedInfo, err := m.NewSortInfo(NullFamilyID, 77, []ASTParameter{parameter}, InfiniteSortSize(), false)
	astTestMust(t, err)
	parameterizedSort, err := m.MkSort(astTestSymbol(t, m, "Parameterized"), parameterizedInfo)
	astTestMust(t, err)
	check("sort AST parameter", parameterizedSort, true)

	plainDecl, err := m.MkFuncDecl(astTestSymbol(t, m, "plain"), []ASTHandle{floating}, boolean, ASTFuncDeclInfo{})
	astTestMust(t, err)
	check("declaration domain", plainDecl, true)
	rangeDecl, err := m.MkFuncDecl(astTestSymbol(t, m, "range"), nil, floating, ASTFuncDeclInfo{})
	astTestMust(t, err)
	check("declaration range", rangeDecl, true)

	declInfo, err := m.NewFuncDeclInfo(NullFamilyID, 78, []ASTParameter{parameter}, ASTFuncFlags{}, false)
	astTestMust(t, err)
	parameterizedDecl, err := m.MkFuncDeclWithInfo(astTestSymbol(t, m, "parameterized"), nil, boolean, declInfo)
	astTestMust(t, err)
	check("declaration AST parameter", parameterizedDecl, true)
	app, err := m.MkApp(parameterizedDecl, nil)
	astTestMust(t, err)
	check("application declaration", app, true)

	truth, err := m.MkTrue()
	astTestMust(t, err)
	quantifier, err := m.MkQuantifier(ASTQuantifierSpec{
		Forall: true,
		Sorts:  []ASTHandle{floating},
		Names:  []usymbol.Symbol{astTestSymbol(t, m, "x")},
		Body:   truth,
	})
	astTestMust(t, err)
	check("quantifier declaration sort", quantifier, true)
}
