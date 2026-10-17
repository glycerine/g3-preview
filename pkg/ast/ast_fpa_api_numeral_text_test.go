package ast

import (
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestASTFPANumeralStringAPIScopedMPFAndBranches(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	floatSort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	for _, tc := range []struct {
		kind int32
		want string
	}{
		{astFPARMNearestEven, "roundNearestTiesToEven"},
		{astFPARMNearestAway, "roundNearestTiesToAway"},
		{astFPARMTowardPositive, "roundTowardPositive"},
		{astFPARMTowardNegative, "roundTowardNegative"},
		{astFPARMTowardZero, "roundTowardZero"},
	} {
		h, err := m.MkFamilyApp(family, tc.kind, nil, nil)
		astTestMust(t, err)
		got, matched, err := m.FPANumeralString(h)
		if err != nil || !matched || got != tc.want || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("API rounding-mode text or scoped cleanup", tc.kind, got, matched, err)
		}
	}
	for _, tc := range []struct {
		make func() (ASTHandle, error)
		want string
	}{
		{func() (ASTHandle, error) { return m.MkFPANaNOfSort(floatSort) }, "NaN"},
		{func() (ASTHandle, error) { return m.MkFPAInfinityOfSort(floatSort, false) }, "+oo"},
		{func() (ASTHandle, error) { return m.MkFPAZeroOfSort(floatSort, true) }, "-zero"},
	} {
		h, err := tc.make()
		astTestMust(t, err)
		got, matched, err := m.FPANumeralString(h)
		if err != nil || !matched || got != tc.want || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatal("API FP numeral text or scoped cleanup", got, matched, tc.want, err)
		}
	}
	var finite unumeral.ArbitraryFloat
	astTestMust(t, m.floatingPointPlugin.numeric.SetDouble(&finite, 8, 24, 1.5))
	finiteRef, err := m.floatingPointPlugin.mkNumeral(&finite, nil)
	astTestMust(t, err)
	astTestMust(t, m.floatingPointPlugin.numeric.Del(&finite))
	if got, matched, err := m.FPANumeralString(m.external(finiteRef)); err != nil || !matched || got != "1.5 0" || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("API finite FP decimal formatting or scoped cleanup", got, matched, err)
	}
	symbolic, err := m.MkVar(0, floatSort)
	astTestMust(t, err)
	got, matched, err := m.FPANumeralString(symbolic)
	if err != nil || matched || got != "" || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("API symbolic refusal or scoped cleanup", got, matched, err)
	}
}
