package ast

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestASTFPAToFPArithmeticInputSourceTrace(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	rmSort, err := m.MkFamilySort(family, 1, nil)
	astTestMust(t, err)
	eParam, err := m.NewParameterInt(8)
	astTestMust(t, err)
	sParam, err := m.NewParameterInt(24)
	astTestMust(t, err)
	defer func() {
		astTestMust(t, m.DelParameter(&sParam))
		astTestMust(t, m.DelParameter(&eParam))
	}()
	decl, err := m.MkFamilyFuncDecl(family, FPAOpToFP, []ASTParameter{eParam, sParam}, []ASTHandle{rmSort, m.external(m.floatingPointPlugin.real)}, ASTHandle{})
	astTestMust(t, err)
	intDecl, err := m.MkFamilyFuncDecl(family, FPAOpToFP, []ASTParameter{eParam, sParam}, []ASTHandle{rmSort, m.external(m.floatingPointPlugin.integer)}, ASTHandle{})
	astTestMust(t, err)
	rm, err := m.MkFamilyApp(family, astFPARMNearestEven, nil, nil)
	astTestMust(t, err)
	q, err := m.numerals.NewValue()
	astTestMust(t, err)
	defer func() { astTestMust(t, m.numerals.Del(&q)) }()
	astTestMust(t, m.numerals.SetString(&q, "3/2"))
	value, err := m.ArithmeticNumeral(&q, false)
	astTestMust(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m.traceTags = tags
	if _, found, e := m.RewriteFPAToFPArithmeticNumeral(decl, rm, value); e != nil || !found {
		t.Fatalf("disabled trace conversion: found=%v err=%v", found, e)
	}
	if output.Len() != 0 {
		t.Fatalf("disabled source trace: %q", output.String())
	}
	tags.Enable("fp_rewriter")
	if _, found, e := m.RewriteFPAToFPArithmeticNumeral(decl, rm, value); e != nil || !found {
		t.Fatalf("enabled trace conversion: found=%v err=%v", found, e)
	}
	astTestMust(t, m.numerals.SetString(&q, "7"))
	integerValue, err := m.ArithmeticNumeral(&q, true)
	astTestMust(t, err)
	if _, found, e := m.RewriteFPAToFPArithmeticNumeral(intDecl, rm, integerValue); e != nil || !found {
		t.Fatalf("enabled integer trace conversion: found=%v err=%v", found, e)
	}
	if !tags.ShouldTrace("fp_rewriter") {
		if output.Len() != 0 {
			t.Fatalf("erased source trace: %q", output.String())
		}
		return
	}
	want := "-------- [fp_rewriter] mk_to_fp src/ast/rewriter/fpa_rewriter.cpp:159 ---------\n" +
		"r: 3/2\n" +
		"------------------------------------------------\n" +
		"-------- [fp_rewriter] mk_to_fp src/ast/rewriter/fpa_rewriter.cpp:159 ---------\n" +
		"r: 7\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source rational-input trace got %q, want %q", got, want)
	}
	refused := diagnostic.NewTraceTagsWithWriter(astAppTraceRefusingWriter{})
	refused.Enable("fp_rewriter")
	m.traceTags = refused
	if _, found, e := m.RewriteFPAToFPArithmeticNumeral(decl, rm, value); e != nil || !found || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatalf("borrowed trace-stream refusal changed conversion: found=%v err=%v", found, e)
	}
}
