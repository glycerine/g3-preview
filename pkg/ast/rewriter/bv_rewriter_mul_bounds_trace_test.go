package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:2762-2784; src/util/trace.h:40-46.

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestBVRewriterSignedMultiplyUnderflowSourceTrace(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, err := NewBVRewriter(m, c, g, nil)
	astTestMust(t, err)
	defer func() { astTestMust(t, p.Close()) }()
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("bv_rewriter_bvsmul_no_underflow")
	astTestMust(t, p.BindTrace(tags))
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	rewrite := func(a, b uint32, mode BVMultiplyBound, want bool) {
		t.Helper()
		args := ast.SliceArguments([]ast.ASTHandle{bvAddNum(t, m, a, 8), bvAddNum(t, m, b, 8)})
		status, e := p.RewriteMultiplyBound(args, out, mode)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if status != RewriteDone || m.IsTrue(got) != want {
			t.Fatalf("multiply bound %d * %d: status=%v result=%v", a, b, status, got)
		}
	}
	rewrite(128, 2, BVSignedNoUnderflow, false)
	rewrite(255, 2, BVSignedNoUnderflow, true)
	rewrite(1, 2, BVSignedNoUnderflow, true)
	rewrite(128, 2, BVUnsignedNoOverflow, false)
	if !tags.ShouldTrace("bv_rewriter_bvsmul_no_underflow") {
		if output.Len() != 0 {
			t.Fatal("erased source trace", output.String())
		}
		return
	}
	record := func(body string) string {
		return "-------- [bv_rewriter_bvsmul_no_underflow] mk_bvsmul_no_underflow src/ast/rewriter/bv_rewriter.cpp:2781 ---------\n" + body + "------------------------------------------------\n"
	}
	want := record("a0:-128 a1:2 mr:-256 neg_lim:-128\n") + record("a0:-1 a1:2 mr:-2 neg_lim:-128\n")
	if got := output.String(); got != want {
		t.Fatalf("source underflow records: got %q, want %q", got, want)
	}
	tags.Disable("bv_rewriter_bvsmul_no_underflow")
	rewrite(128, 2, BVSignedNoUnderflow, false)
	if got := output.String(); got != want {
		t.Fatalf("disabled underflow trace: got %q, want %q", got, want)
	}
}
