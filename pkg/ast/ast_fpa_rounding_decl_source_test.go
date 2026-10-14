package ast

import (
	"errors"
	"testing"
)

func TestASTFloatingPointRoundingDeclarationsAndApplications(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	family, e := m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	for _, tc := range []struct {
		kind int32
		name string
	}{
		{astFPARMNearestEven, "roundNearestTiesToEven"},
		{astFPARMNearestAway, "roundNearestTiesToAway"},
		{astFPARMTowardPositive, "roundTowardPositive"},
		{astFPARMTowardNegative, "roundTowardNegative"},
		{astFPARMTowardZero, "roundTowardZero"},
	} {
		for _, makeApp := range []bool{false, true} {
			var h ASTHandle
			if makeApp {
				h, e = m.MkFamilyApp(family, tc.kind, nil, nil)
			} else {
				h, e = m.MkFamilyFuncDecl(family, tc.kind, nil, nil, ASTHandle{})
			}
			astTestMust(t, e)
			r, err := m.internal(h)
			astTestMust(t, err)
			if makeApp {
				if m.nodeAt(r.slot).kind != ASTApp {
					t.Fatal("rounding expression is not an application", tc.kind)
				}
				r = m.appAt(r.slot).decl
			}
			gotFamily, gotKind, err := m.declIdentity(r)
			astTestMust(t, err)
			d := m.declAt(r.slot)
			name, err := m.Symbols.String(d.name)
			astTestMust(t, err)
			info, err := m.sortInfo(m.sortAt(d.rangeSort.slot).info)
			astTestMust(t, err)
			if gotFamily != family || gotKind != tc.kind || name != tc.name || d.arity != 0 || info.family != family || info.kind != 1 {
				t.Fatalf("kind %d app %t: got family %d kind %d name %q arity %d sort %d/%d", tc.kind, makeApp, gotFamily, gotKind, name, d.arity, info.family, info.kind)
			}
		}
	}
	astTestMust(t, m.Close())
}

func TestASTFloatingPointRoundingValidationOrder(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	family, e := m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p, e := m.NewParameterInt(7)
	astTestMust(t, e)
	boolSort := m.external(m.boolSort)
	for _, tc := range []struct {
		ps      []ASTParameter
		domain  []ASTHandle
		message string
	}{
		{[]ASTParameter{p}, []ASTHandle{boolSort}, "rounding mode constant does not have parameters"},
		{nil, []ASTHandle{boolSort}, "rounding mode is a constant"},
	} {
		decl, err := m.MkFamilyFuncDecl(family, astFPARMNearestEven, tc.ps, tc.domain, ASTHandle{})
		var source ASTException
		if decl != (ASTHandle{}) || !errors.As(err, &source) || source.Error() != tc.message {
			t.Fatal("source rounding validation order", decl, err, tc.message)
		}
	}
	astTestMust(t, m.Close())
}
