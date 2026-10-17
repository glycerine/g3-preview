package ast

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestASTFPAFPComponentsNumeralsAndTrace(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	signSort, err := m.BitVectorSort(1)
	astTestMust(t, err)
	expSort, err := m.BitVectorSort(8)
	astTestMust(t, err)
	sigSort, err := m.BitVectorSort(23)
	astTestMust(t, err)
	decl, err := m.MkFamilyFuncDecl(family, FPAOpFP, nil, []ASTHandle{signSort, expSort, sigSort}, ASTHandle{})
	astTestMust(t, err)
	for _, invalid := range [][]ASTHandle{{expSort, expSort, sigSort}, {signSort, expSort}} {
		if _, e := m.MkFamilyFuncDecl(family, FPAOpFP, nil, invalid, ASTHandle{}); e == nil {
			t.Fatalf("invalid fp declaration accepted: %v", invalid)
		}
	}
	sign, err := m.BitVectorNumeralUint64(1, 1)
	astTestMust(t, err)
	exponent, err := m.BitVectorNumeralUint64(127, 8)
	astTestMust(t, err)
	significand, err := m.BitVectorNumeralUint64(0, 23)
	astTestMust(t, err)
	app, err := m.MkFamilyApp(family, FPAOpFP, nil, []ASTHandle{sign, exponent, significand})
	astTestMust(t, err)
	if kind, e := m.GetDeclKind(app); e != nil || kind != FPAOpFP {
		t.Fatalf("derived fp application declaration: kind=%v err=%v", kind, e)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m.traceTags = tags
	value, found, err := m.RewriteFPAFPComponentsNumerals(decl, sign, exponent, significand)
	astTestMust(t, err)
	if !found || m.floatingPointPlugin.numeralProbeFrames != nil || output.Len() != 0 {
		t.Fatalf("disabled concrete FP rewrite: found=%v trace=%q", found, output.String())
	}
	bits, found, _, err := m.RewriteFPAToIEEEBVNumeral(value, false)
	astTestMust(t, err)
	want, err := m.BitVectorNumeralUint64(0xbf800000, 32)
	astTestMust(t, err)
	if !found || bits != want {
		t.Fatalf("negative one IEEE32 bits: found=%v got=%v want=%v", found, bits, want)
	}
	tags.Enable("fp_rewriter")
	_, found, err = m.RewriteFPAFPComponentsNumerals(decl, sign, exponent, significand)
	astTestMust(t, err)
	if !found {
		t.Fatal("enabled trace changed concrete FP rewrite")
	}
	symbolic, err := m.MkVar(0, sigSort)
	astTestMust(t, err)
	_, found, err = m.RewriteFPAFPComponentsNumerals(decl, sign, exponent, symbolic)
	astTestMust(t, err)
	if found || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatal("symbolic significand must refuse and retire scoped MPF")
	}
	if !tags.ShouldTrace("fp_rewriter") {
		if output.Len() != 0 {
			t.Fatalf("erased trace wrote %q", output.String())
		}
		return
	}
	wantTrace := "-------- [fp_rewriter] mk_fp src/ast/rewriter/fpa_rewriter.cpp:700 ---------\n" +
		"simplified (fp ...) to -1.0 0\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != wantTrace {
		t.Fatalf("mk_fp trace: got %q, want %q", got, wantTrace)
	}
	refused := diagnostic.NewTraceTagsWithWriter(astAppTraceRefusingWriter{})
	refused.Enable("fp_rewriter")
	m.traceTags = refused
	_, found, err = m.RewriteFPAFPComponentsNumerals(decl, sign, exponent, significand)
	if err != nil || !found || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatalf("refused trace changed rewrite or scoped cleanup: found=%v err=%v", found, err)
	}
}
