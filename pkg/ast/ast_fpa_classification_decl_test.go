package ast

import "testing"

func TestASTFloatingPointClassificationDeclarationsAndApplications(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	floatSort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	parameter, err := m.NewParameterAST(floatSort)
	astTestMust(t, err)
	zero, err := m.MkFamilyApp(family, astFPAPlusZero, []ASTParameter{parameter}, nil)
	astTestMust(t, err)
	for _, tc := range []struct {
		kind int32
		name string
	}{
		{astFPAIsZero, "fp.isZero"},
		{astFPAIsNegative, "fp.isNegative"},
		{astFPAIsPositive, "fp.isPositive"},
		{astFPAIsNaN, "fp.isNaN"},
		{astFPAIsInf, "fp.isInfinite"},
		{astFPAIsNormal, "fp.isNormal"},
		{astFPAIsSubnormal, "fp.isSubnormal"},
	} {
		decl, err := m.MkFamilyFuncDecl(family, tc.kind, nil, []ASTHandle{floatSort}, ASTHandle{})
		astTestMust(t, err)
		r, err := m.internal(decl)
		astTestMust(t, err)
		gotFamily, gotKind, err := m.declIdentity(r)
		astTestMust(t, err)
		if gotFamily != family || gotKind != tc.kind || m.declAt(r.slot).arity != 1 || m.declAt(r.slot).rangeSort != m.boolSort {
			t.Fatalf("%s declaration mismatch", tc.name)
		}
		name, err := m.Symbols.String(m.declAt(r.slot).name)
		astTestMust(t, err)
		if name != tc.name {
			t.Fatalf("kind %d name %q, want %q", tc.kind, name, tc.name)
		}
		app, err := m.MkFamilyApp(family, tc.kind, nil, []ASTHandle{zero})
		astTestMust(t, err)
		gotSort, err := m.GetSort(app)
		astTestMust(t, err)
		if gotSort != m.external(m.boolSort) {
			t.Fatalf("%s application has wrong range", tc.name)
		}
	}
	if _, err := m.MkFamilyFuncDecl(family, astFPAIsNaN, nil, nil, ASTHandle{}); err == nil {
		t.Fatal("zero-arity classification accepted")
	}
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	if _, err := m.MkFamilyFuncDecl(family, astFPAIsNaN, nil, []ASTHandle{boolSort}, ASTHandle{}); err == nil {
		t.Fatal("non-floating-point classification domain accepted")
	}
}
