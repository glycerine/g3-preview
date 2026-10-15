package ast

import "testing"

func TestASTSMT2FunctionDeclarationParameterClassification(t *testing.T) {
	m := astTestManager(t)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	intSort, err := m.MkSort(astTestSymbol(t, m, "other-sort"), ASTSortInfo{})
	astTestMust(t, err)
	name := astTestSymbol(t, m, "indexed")
	referenced, err := m.MkFuncDecl(astTestSymbol(t, m, "reference"), nil, boolSort, ASTFuncDeclInfo{})
	astTestMust(t, err)

	integer, err := m.NewParameterInt(-2)
	astTestMust(t, err)
	r, err := m.numerals.NewValue()
	astTestMust(t, err)
	astTestMust(t, m.numerals.SetString(&r, "7/3"))
	rational, err := m.NewParameterRational(&r)
	astTestMust(t, err)
	astTestMust(t, m.numerals.Del(&r))
	function, err := m.NewParameterAST(referenced)
	astTestMust(t, err)
	sortParam, err := m.NewParameterAST(boolSort)
	astTestMust(t, err)
	symbol, err := m.NewParameterSymbol(astTestSymbol(t, m, "bad"))
	astTestMust(t, err)

	makeDecl := func(family FamilyID, params []ASTParameter, rangeSort ASTHandle) ASTHandle {
		info, e := m.NewFuncDeclInfo(family, 741, params, ASTFuncFlags{}, false)
		astTestMust(t, e)
		decl, e := m.MkFuncDeclWithInfo(name, nil, rangeSort, info)
		astTestMust(t, e)
		astTestMust(t, m.DelFuncDeclInfo(&info))
		return decl
	}
	for _, tc := range []struct {
		name        string
		decl        ASTHandle
		indexed, as bool
	}{
		{"integer-rational-function", makeDecl(0, []ASTParameter{integer, rational, function}, boolSort), true, false},
		{"sort-as-range", makeDecl(0, []ASTParameter{sortParam}, boolSort), false, true},
		{"sort-other-range", makeDecl(0, []ASTParameter{sortParam}, intSort), false, false},
		{"unsupported-symbol", makeDecl(0, []ASTParameter{integer, symbol}, boolSort), false, false},
		{"null-family", makeDecl(NullFamilyID, []ASTParameter{integer}, boolSort), false, false},
		{"no-parameters", makeDecl(0, nil, boolSort), false, false},
		{"plain", referenced, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			indexed, e := m.SMT2IsIndexedFdecl(tc.decl)
			astTestMust(t, e)
			as, e := m.SMT2IsSortParam(tc.decl)
			astTestMust(t, e)
			if indexed != tc.indexed || as != tc.as {
				t.Fatalf("indexed=%v, as=%v; want %v, %v", indexed, as, tc.indexed, tc.as)
			}
		})
	}
	for _, p := range []*ASTParameter{&integer, &rational, &function, &sortParam, &symbol} {
		astTestMust(t, m.DelParameter(p))
	}
	if _, err := m.SMT2IsIndexedFdecl(boolSort); err == nil {
		t.Fatal("sort cannot be classified as a function declaration")
	}
}
