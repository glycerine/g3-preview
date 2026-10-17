package ast

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type astFPASignPrefixStopWriter struct{ bytes.Buffer }

func (w *astFPASignPrefixStopWriter) Write(p []byte) (int, error) {
	if strings.HasSuffix(w.Buffer.String(), "sgn: ") {
		return 0, io.ErrClosedPipe
	}
	return w.Buffer.Write(p)
}
func (w *astFPASignPrefixStopWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func TestASTFPAToFPAdjacentSourceTraces(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	rmSort, err := m.MkFamilySort(family, 1, nil)
	astTestMust(t, err)
	fpSort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	eParam, err := m.NewParameterInt(8)
	astTestMust(t, err)
	sParam, err := m.NewParameterInt(24)
	astTestMust(t, err)
	defer func() {
		astTestMust(t, m.DelParameter(&sParam))
		astTestMust(t, m.DelParameter(&eParam))
	}()
	ps := []ASTParameter{eParam, sParam}
	rm, err := m.MkFamilyApp(family, astFPARMNearestEven, nil, nil)
	astTestMust(t, err)
	formatDecl, err := m.MkFamilyFuncDecl(family, FPAOpToFP, ps, []ASTHandle{rmSort, fpSort}, ASTHandle{})
	astTestMust(t, err)
	ieeeSort, err := m.BitVectorSort(32)
	astTestMust(t, err)
	ieeeDecl, err := m.MkFamilyFuncDecl(family, FPAOpToFP, ps, []ASTHandle{ieeeSort}, ASTHandle{})
	astTestMust(t, err)
	signedSort, err := m.BitVectorSort(8)
	astTestMust(t, err)
	signedDecl, err := m.MkFamilyFuncDecl(family, FPAOpToFP, ps, []ASTHandle{rmSort, signedSort}, ASTHandle{})
	astTestMust(t, err)
	realSort := m.external(m.floatingPointPlugin.real)
	intSort := m.external(m.floatingPointPlugin.integer)
	realIntDecl, err := m.MkFamilyFuncDecl(family, FPAOpToFP, ps, []ASTHandle{rmSort, realSort, intSort}, ASTHandle{})
	astTestMust(t, err)
	intRealDecl, err := m.MkFamilyFuncDecl(family, FPAOpToFP, ps, []ASTHandle{rmSort, intSort, realSort}, ASTHandle{})
	astTestMust(t, err)
	signSort, err := m.BitVectorSort(2)
	astTestMust(t, err)
	expSort, err := m.BitVectorSort(8)
	astTestMust(t, err)
	sigSort, err := m.BitVectorSort(23)
	astTestMust(t, err)
	threeBVDecl, err := m.MkFamilyFuncDecl(family, FPAOpToFP, nil, []ASTHandle{signSort, expSort, sigSort}, ASTHandle{})
	astTestMust(t, err)
	var input unumeral.ArbitraryFloat
	astTestMust(t, m.floatingPointPlugin.numeric.SetDouble(&input, 8, 24, 1.5))
	inputRef, err := m.floatingPointPlugin.mkNumeral(&input, nil)
	astTestMust(t, err)
	astTestMust(t, m.floatingPointPlugin.numeric.Del(&input))
	q, err := m.numerals.NewValue()
	astTestMust(t, err)
	defer func() { astTestMust(t, m.numerals.Del(&q)) }()
	astTestMust(t, m.numerals.SetString(&q, "3/2"))
	real, err := m.ArithmeticNumeral(&q, false)
	astTestMust(t, err)
	astTestMust(t, m.numerals.SetString(&q, "2"))
	integer, err := m.ArithmeticNumeral(&q, true)
	astTestMust(t, err)
	sign, err := m.BitVectorNumeralUint64(0, 2)
	astTestMust(t, err)
	exponent, err := m.BitVectorNumeralUint64(127, 8)
	astTestMust(t, err)
	significand, err := m.BitVectorNumeralUint64(0, 23)
	astTestMust(t, err)
	signed, err := m.BitVectorNumeralUint64(255, 8)
	astTestMust(t, err)
	ieee, err := m.BitVectorNumeralUint64(0x3fc00000, 32)
	astTestMust(t, err)
	negativeIEEE, err := m.BitVectorNumeralUint64(0xbf800000, 32)
	astTestMust(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("fp_rewriter")
	m.traceTags = tags
	check := func(found bool, e error) {
		t.Helper()
		if e != nil || !found || m.floatingPointPlugin.numeralProbeFrames != nil {
			t.Fatalf("FPA source trace branch: found=%v err=%v", found, e)
		}
	}
	_, found, e := m.RewriteFPAToFPIEEEBVNumeral(ieeeDecl, ieee)
	check(found, e)
	_, found, e = m.RewriteFPAToFPIEEEBVNumeral(ieeeDecl, negativeIEEE)
	check(found, e)
	_, found, e = m.RewriteFPAToFPFormatNumeral(formatDecl, rm, m.external(inputRef))
	check(found, e)
	_, found, e = m.RewriteFPAToFPSignedBVNumeral(signedDecl, rm, signed)
	check(found, e)
	_, found, e = m.RewriteFPAToFPExponentRationalNumerals(realIntDecl, rm, real, integer)
	check(found, e)
	_, found, e = m.RewriteFPAToFPExponentRationalNumerals(intRealDecl, rm, integer, real)
	check(found, e)
	_, found, e = m.RewriteFPAToFPThreeBVNumerals(threeBVDecl, sign, exponent, significand)
	check(found, e)
	if !tags.ShouldTrace("fp_rewriter") {
		if output.Len() != 0 {
			t.Fatalf("erased FPA traces: %q", output.String())
		}
		return
	}
	want := "-------- [fp_rewriter] mk_to_fp src/ast/rewriter/fpa_rewriter.cpp:143 ---------\n" +
		"sgn: 0\nsig: 4194304\nexp: 0\nv: 1.5 0\n------------------------------------------------\n" +
		"-------- [fp_rewriter] mk_to_fp src/ast/rewriter/fpa_rewriter.cpp:143 ---------\n" +
		"sgn: 1\nsig: 0\nexp: 0\nv: -1.0 0\n------------------------------------------------\n" +
		"-------- [fp_rewriter] mk_to_fp src/ast/rewriter/fpa_rewriter.cpp:168 ---------\n" +
		"v: 1.5 0\n------------------------------------------------\n" +
		"-------- [fp_rewriter] mk_to_fp src/ast/rewriter/fpa_rewriter.cpp:177 ---------\n" +
		"r1: 255\n------------------------------------------------\n" +
		"-------- [fp_rewriter] mk_to_fp src/ast/rewriter/fpa_rewriter.cpp:179 ---------\n" +
		"r1 norm: -1\n------------------------------------------------\n" +
		"-------- [fp_rewriter] mk_to_fp src/ast/rewriter/fpa_rewriter.cpp:195 ---------\n" +
		"r1: 3/2, r2: 2\n------------------------------------------------\n" +
		"-------- [fp_rewriter] mk_to_fp src/ast/rewriter/fpa_rewriter.cpp:209 ---------\n" +
		"r1: 2, r2: 3/2\n------------------------------------------------\n" +
		"-------- [fp_rewriter] mk_to_fp src/ast/rewriter/fpa_rewriter.cpp:226 ---------\n" +
		"v = 1.0 0\n------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("adjacent source traces got %q, want %q", got, want)
	}
	symbolic, err := m.MkVar(0, fpSort)
	astTestMust(t, err)
	_, found, e = m.RewriteFPAToFPFormatNumeral(formatDecl, rm, symbolic)
	if e != nil || found || output.String() != want {
		t.Fatalf("symbolic input reached a TRACE branch: found=%v err=%v trace=%q", found, e, output.String())
	}
	stopped := &astFPASignPrefixStopWriter{}
	stoppedTags := diagnostic.NewTraceTagsWithWriter(stopped)
	stoppedTags.Enable("fp_rewriter")
	m.traceTags = stoppedTags
	_, found, e = m.RewriteFPAToFPIEEEBVNumeral(ieeeDecl, ieee)
	check(found, e)
	wantPrefix := "-------- [fp_rewriter] mk_to_fp src/ast/rewriter/fpa_rewriter.cpp:143 ---------\nsgn: "
	if got := stopped.String(); got != wantPrefix {
		t.Fatalf("stopped source field prefix got %q, want %q", got, wantPrefix)
	}
	refused := diagnostic.NewTraceTagsWithWriter(astAppTraceRefusingWriter{})
	refused.Enable("fp_rewriter")
	m.traceTags = refused
	_, found, e = m.RewriteFPAToFPFormatNumeral(formatDecl, rm, m.external(inputRef))
	check(found, e)
	_, found, e = m.RewriteFPAToFPThreeBVNumerals(threeBVDecl, sign, exponent, significand)
	check(found, e)
}
