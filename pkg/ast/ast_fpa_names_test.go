package ast

import "testing"

func TestASTFloatingPointOriginalBuiltinNames(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	var ops, sorts ASTBuiltinNames
	astTestMust(t, m.GetFloatingPointOpNames(&ops, astTestSymbol(t, m, "unrestricted")))
	astTestMust(t, m.GetFloatingPointSortNames(&sorts, astTestSymbol(t, m, "unrestricted")))
	if len(ops.Items()) != 47 || len(sorts.Items()) != 6 {
		t.Fatal("builtin name counts", len(ops.Items()), len(sorts.Items()))
	}
	wantOps := []struct {
		index int
		name  string
		kind  int32
	}{{0, "+oo", astFPAPlusInf}, {10, "RNE", astFPARMNearestEven}, {15, "fp.abs", astFPAAbs}, {44, "to_fp_unsigned", astFPAToFPUnsigned}, {45, "to_ieee_bv", astFPAToIEEEBV}, {46, "fp.to_ieee_bv", astFPAToIEEEBV}}
	for _, want := range wantOps {
		got := ops.Items()[want.index]
		text, e := m.Symbols.Display(got.Name)
		astTestMust(t, e)
		if text != want.name || got.Kind != want.kind {
			t.Fatal("operation name", want.index, text, got.Kind, want)
		}
	}
	for i, want := range []string{"FloatingPoint", "RoundingMode", "Float16", "Float32", "Float64", "Float128"} {
		got := sorts.Items()[i]
		text, e := m.Symbols.Display(got.Name)
		astTestMust(t, e)
		if text != want || got.Kind != int32(i) {
			t.Fatal("sort name", i, text, got.Kind, want)
		}
	}
}
