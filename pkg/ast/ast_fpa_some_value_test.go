package ast

// Source: src/ast/fpa_decl_plugin.cpp:856-869.

import "testing"

func TestASTFloatingPointSomeValueSourceDefaults(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	floatSort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	rmSort, err := m.MkFamilySort(family, 1, nil)
	astTestMust(t, err)
	nan, err := m.GetSomeValue(floatSort)
	astTestMust(t, err)
	wantNaN, err := m.MkFPANaNOfSort(floatSort)
	astTestMust(t, err)
	if nan != wantNaN {
		t.Fatal("floating-point default must be canonical NaN", nan, wantNaN)
	}
	rm, err := m.GetSomeValue(rmSort)
	astTestMust(t, err)
	wantRM, err := m.MkFamilyApp(family, astFPARMTowardZero, nil, nil)
	astTestMust(t, err)
	if rm != wantRM {
		t.Fatal("rounding-mode default must be toward zero", rm, wantRM)
	}
	ebits, err := m.NewParameterInt(5)
	astTestMust(t, err)
	sbits, err := m.NewParameterInt(11)
	astTestMust(t, err)
	customSort, err := m.MkFamilySort(family, 0, []ASTParameter{ebits, sbits})
	astTestMust(t, err)
	customNaN, err := m.GetSomeValue(customSort)
	astTestMust(t, err)
	wantCustom, err := m.MkFPANaNOfSort(customSort)
	astTestMust(t, err)
	if customNaN != wantCustom || customNaN == nan {
		t.Fatal("custom floating-point format default", customNaN, wantCustom, nan)
	}
	if m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("get_some_value retained a completed MPF local")
	}
}

func TestASTFloatingPointSomeValueRefusedMPFConstruction(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	baseline := m.Budget().Used()
	m.Budget().FailAfter(0)
	value, err := m.GetSomeValue(sort)
	m.Budget().FailAfter(-1)
	if err == nil || !value.IsNull() {
		t.Fatal("refused MPF construction published a value", value, err)
	}
	frame := m.floatingPointPlugin.numeralProbeFrames
	if frame == nil || frame.bodyCause == nil || !frame.sourceDone || !frame.consumed || m.Budget().Used() != baseline {
		t.Fatalf("refused MPF construction lost its stopped record or local cleanup: frame=%+v budget=%d baseline=%d", frame, m.Budget().Used(), baseline)
	}
	value, err = m.GetSomeValue(sort)
	astTestMust(t, err)
	if value.IsNull() || m.floatingPointPlugin.numeralProbeFrames != frame {
		t.Fatal("later FPA value did not recover while failed invocation remained owned")
	}
}
