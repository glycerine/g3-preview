package ast

import "testing"

func TestASTFPAClassificationOneScopedNumeralPerCall(t *testing.T) {
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
	nan, err := m.MkFamilyApp(family, astFPANaN, []ASTParameter{parameter}, nil)
	astTestMust(t, err)
	for _, tc := range []struct {
		kind FPAClassificationKind
		want bool
	}{
		{FPAIsNaN, true}, {FPAIsInfinite, false}, {FPAIsZero, false},
		{FPAIsNormal, false}, {FPAIsSubnormal, false}, {FPAIsNegative, false}, {FPAIsPositive, false},
	} {
		found, value, err := m.ClassifyFPANumeral(nan, tc.kind)
		astTestMust(t, err)
		if !found || value != tc.want || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatalf("classification %d: found=%v value=%v active probe=%v", tc.kind, found, value, m.floatingPointPlugin.numeralProbeFrames)
		}
	}
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	found, value, err := m.ClassifyFPANumeral(variable, FPAIsNaN)
	astTestMust(t, err)
	if found || value || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("symbolic operand retained a scoped numeral", found, value)
	}
}
