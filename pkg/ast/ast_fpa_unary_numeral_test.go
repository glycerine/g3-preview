package ast

import "testing"

func TestASTFPAUnaryScopedNumeralRetiresBeforeReturn(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	parameter, err := m.NewParameterAST(sort)
	astTestMust(t, err)
	positive, err := m.MkFamilyApp(family, astFPAPlusZero, []ASTParameter{parameter}, nil)
	astTestMust(t, err)
	negative, err := m.MkFamilyApp(family, astFPAMinusZero, []ASTParameter{parameter}, nil)
	astTestMust(t, err)
	for _, tc := range []struct {
		kind int32
		want ASTHandle
	}{{FPAOpNeg, positive}, {FPAOpAbs, positive}} {
		out, found, err := m.RewriteFPAUnaryNumeral(negative, tc.kind)
		astTestMust(t, err)
		if !found || out != tc.want || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("FPA unary local survived successful source return", tc.kind, found, out)
		}
	}
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	out, found, err := m.RewriteFPAUnaryNumeral(variable, FPAOpAbs)
	astTestMust(t, err)
	if found || out != (ASTHandle{}) || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("symbolic unary numeral probe retained", found, out)
	}
}
