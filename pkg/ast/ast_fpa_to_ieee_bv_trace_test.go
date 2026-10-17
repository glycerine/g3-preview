package ast

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestASTFPAToIEEEBVSourceTrace(t *testing.T) {
	c, globals, _, budget := dynAckSourceFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	previous, err := uparams.BindProcessGlobalParamsService(globals)
	astTestMust(t, err)
	t.Cleanup(func() { _, e := uparams.BindProcessGlobalParamsService(previous); astTestMust(t, e) })
	_, err = m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	var value numeral.ArbitraryFloat
	astTestMust(t, m.floatingPointPlugin.numeric.SetDouble(&value, 8, 24, 1.5))
	ref, err := m.floatingPointPlugin.mkNumeral(&value, nil)
	astTestMust(t, err)
	astTestMust(t, m.floatingPointPlugin.numeric.Del(&value))
	constant := m.external(ref)
	if _, found, _, e := m.RewriteFPAToIEEEBVNumeral(constant, false); e != nil || !found {
		t.Fatalf("disabled trace conversion: found=%v err=%v", found, e)
	}
	if output.Len() != 0 {
		t.Fatalf("disabled trace wrote %q", output.String())
	}
	tags.Enable("fp_rewriter")
	if _, found, _, e := m.RewriteFPAToIEEEBVNumeral(constant, false); e != nil || !found {
		t.Fatalf("numeral trace conversion: found=%v err=%v", found, e)
	}
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	if _, found, _, e := m.RewriteFPAToIEEEBVNumeral(variable, false); e != nil || found {
		t.Fatalf("symbolic trace conversion: found=%v err=%v", found, e)
	}
	if !tags.ShouldTrace("fp_rewriter") {
		if output.Len() != 0 {
			t.Fatalf("erased trace wrote %q", output.String())
		}
		return
	}
	want := "-------- [fp_rewriter] mk_to_ieee_bv src/ast/rewriter/fpa_rewriter.cpp:765 ---------\n" +
		"to_ieee_bv of (fp #b0 #x7f #b10000000000000000000000)\n" +
		"------------------------------------------------\n" +
		"-------- [fp_rewriter] mk_to_ieee_bv src/ast/rewriter/fpa_rewriter.cpp:769 ---------\n" +
		"to_ieee_bv numeral: 1.5 0\n" +
		"------------------------------------------------\n" +
		"-------- [fp_rewriter] mk_to_ieee_bv src/ast/rewriter/fpa_rewriter.cpp:765 ---------\n" +
		"to_ieee_bv of (:var 0)\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source traces and guard order: got %q, want %q", got, want)
	}
	refused := diagnostic.NewTraceTagsWithWriter(astAppTraceRefusingWriter{})
	refused.Enable("fp_rewriter")
	m.traceTags = refused
	if _, found, _, e := m.RewriteFPAToIEEEBVNumeral(constant, false); e != nil || !found || m.floatingPointPlugin.numeralProbeFrames != nil {
		t.Fatalf("refused trace changed conversion or scoped cleanup: found=%v err=%v", found, e)
	}
}
