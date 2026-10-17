package ast

import "testing"

func TestASTFPASMTEqualityTwoScopedNumeralsAndShortCircuit(t *testing.T) {
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
	value := func(kind int32) ASTHandle {
		h, e := m.MkFamilyApp(family, kind, []ASTParameter{parameter}, nil)
		astTestMust(t, e)
		return h
	}
	nan, pzero, nzero := value(astFPANaN), value(astFPAPlusZero), value(astFPAMinusZero)
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	for _, tc := range []struct {
		left, right  ASTHandle
		found, equal bool
	}{
		{nan, nan, true, true},
		{pzero, nzero, true, false},
		{pzero, pzero, true, true},
		{nan, pzero, true, false},
		{variable, nan, false, false},
		{nan, variable, false, false},
	} {
		found, equal, err := m.EqualFPANumerals(tc.left, tc.right)
		astTestMust(t, err)
		if found != tc.found || equal != tc.equal || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatalf("SMT equality: found=%v equal=%v active=%v", found, equal, m.floatingPointPlugin.numeralProbeFrames)
		}
	}
}
