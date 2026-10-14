package ast

import (
	"errors"
	"testing"
)

func TestASTFloatingPointSpecialDeclSortSelectionAndScopedValue(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	family, e := m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	sort, e := m.MkFamilySort(family, 3, nil)
	astTestMust(t, e)
	sortParameter, e := m.NewParameterAST(sort)
	astTestMust(t, e)
	ebits, e := m.NewParameterInt(11)
	astTestMust(t, e)
	sbits, e := m.NewParameterInt(53)
	astTestMust(t, e)
	invalidSortParameter, e := m.NewParameterAST(m.external(m.boolSort))
	astTestMust(t, e)
	for _, tc := range []struct {
		name         string
		ps           []ASTParameter
		rangeSort    ASTHandle
		wantE, wantS int32
	}{
		{"sort parameter", []ASTParameter{sortParameter}, ASTHandle{}, 8, 24},
		{"integer widths before range", []ASTParameter{ebits, sbits}, sort, 11, 53},
		{"range after invalid sort parameter", []ASTParameter{invalidSortParameter}, sort, 8, 24},
	} {
		for _, kind := range []int32{astFPAPlusInf, astFPAMinusInf, astFPANaN, astFPAPlusZero, astFPAMinusZero} {
			// mk_float_const_decl ignores arity and domain in the source.
			decl, err := m.MkFamilyFuncDecl(family, kind, tc.ps, []ASTHandle{m.external(m.boolSort)}, tc.rangeSort)
			astTestMust(t, err)
			r, err := m.internal(decl)
			astTestMust(t, err)
			gotFamily, gotKind, err := m.declIdentity(r)
			astTestMust(t, err)
			if gotFamily != family || gotKind != kind || m.declAt(r.slot).arity != 0 || m.floatingPointPlugin.numeralProbeFrames != nil {
				t.Fatalf("%s kind %d: declaration or scoped numeral mismatch", tc.name, kind)
			}
			rangeRef := m.declAt(r.slot).rangeSort
			info, err := m.sortInfo(m.sortAt(rangeRef.slot).info)
			astTestMust(t, err)
			first, err := m.infoParameter(info, 0)
			astTestMust(t, err)
			second, err := m.infoParameter(info, 1)
			astTestMust(t, err)
			x, err := m.parameterOfKind(first, ASTParameterInt)
			astTestMust(t, err)
			y, err := m.parameterOfKind(second, ASTParameterInt)
			astTestMust(t, err)
			if x.integer != tc.wantE || y.integer != tc.wantS {
				t.Fatalf("%s kind %d: widths %d/%d", tc.name, kind, x.integer, y.integer)
			}
		}
	}
	if decl, err := m.MkFamilyFuncDecl(family, astFPANaN, nil, nil, ASTHandle{}); decl != (ASTHandle{}) {
		t.Fatal("missing sort published declaration", decl)
	} else {
		var source ASTException
		if !errors.As(err, &source) || source.Error() != "sort of floating point constant was not specified" {
			t.Fatal("source missing-sort exception", err)
		}
	}
	astTestMust(t, m.Close())
}

func TestASTFloatingPointSpecialDeclFailedScalarRetainedForCleanup(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	family, e := m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	ebits, e := m.NewParameterInt(11)
	astTestMust(t, e)
	sbits, e := m.NewParameterInt(5000)
	astTestMust(t, e)
	m.BudgetField.FailAfter(0)
	decl, err := m.MkFamilyFuncDecl(family, astFPANaN, []ASTParameter{ebits, sbits}, nil, ASTHandle{})
	if err == nil || decl != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames == nil {
		t.Fatal("failed source NaN scope was published or discarded", decl, err)
	}
	m.BudgetField.FailAfter(-1)
	astTestMust(t, m.Close())
}

func TestASTFloatingPointSpecialAppReachesDeclarationAndClassification(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	family, e := m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	ebits, e := m.NewParameterInt(8)
	astTestMust(t, e)
	sbits, e := m.NewParameterInt(24)
	astTestMust(t, e)
	for _, tc := range []struct {
		kind  int32
		class astFloatClass
	}{
		{astFPANaN, astFloatNaN},
		{astFPAPlusInf, astFloatPInf},
		{astFPAMinusInf, astFloatNInf},
		{astFPAPlusZero, astFloatPZero},
		{astFPAMinusZero, astFloatNZero},
	} {
		app, err := m.MkFamilyApp(family, tc.kind, []ASTParameter{ebits, sbits}, nil)
		astTestMust(t, err)
		r, err := m.internal(app)
		astTestMust(t, err)
		if m.nodeAt(r.slot).kind != ASTApp {
			t.Fatal("family application did not publish an app", tc.kind)
		}
		decl := m.appAt(r.slot).decl
		gotFamily, gotKind, err := m.declIdentity(decl)
		astTestMust(t, err)
		matched, err := m.floatingPointPlugin.isNumeralClass(r, tc.class)
		astTestMust(t, err)
		if gotFamily != family || gotKind != tc.kind || !matched || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("family app missed special numeral or leaked local scope", tc.kind, gotFamily, gotKind, matched)
		}
	}
	astTestMust(t, m.Close())
}
