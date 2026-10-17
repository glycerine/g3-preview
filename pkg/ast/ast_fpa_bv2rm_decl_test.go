package ast

import "testing"

func TestASTFPABV2RMDeclarationSortsAndImplicitRange(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	bv3, err := m.BitVectorSort(3)
	astTestMust(t, err)
	rmSort, err := m.MkFamilySort(family, 1, nil)
	astTestMust(t, err)
	decl, err := m.MkFamilyFuncDecl(family, FPAOpBV2RM, nil, []ASTHandle{bv3}, rmSort)
	astTestMust(t, err)
	astTestPin(t, m, decl)
	value, err := m.BitVectorNumeralUnsigned(2, 3)
	astTestMust(t, err)
	app, err := m.MkFamilyApp(family, FPAOpBV2RM, nil, []ASTHandle{value})
	astTestMust(t, err)
	got, err := m.GetSort(app)
	astTestMust(t, err)
	if got != rmSort {
		t.Fatal("bv2rm implicit range", got, rmSort)
	}
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	bv4, err := m.BitVectorSort(4)
	astTestMust(t, err)
	for _, tc := range []struct {
		domain    []ASTHandle
		rangeSort ASTHandle
	}{
		{nil, rmSort},
		{[]ASTHandle{bv4}, rmSort},
		{[]ASTHandle{boolSort}, rmSort},
		{[]ASTHandle{bv3}, boolSort},
	} {
		if _, e := m.MkFamilyFuncDecl(family, FPAOpBV2RM, nil, tc.domain, tc.rangeSort); e == nil {
			t.Fatal("invalid bv2rm declaration accepted", tc)
		}
	}
}
