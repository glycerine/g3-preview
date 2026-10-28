package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:770-780; src/util/trace.h:40-46.

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestBVRewriterExtractPropagationSourceTrace(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	bindBVTraceGlobals(t, m)
	local, err := c.NewValue()
	astTestMust(t, err)
	defer func() { astTestMust(t, c.Del(&local)) }()
	key := "bv_extract_prop"
	astTestMust(t, c.SetBoolName(&local, &key, true))
	p, err := NewBVRewriter(m, c, g, &local)
	astTestMust(t, err)
	defer func() { astTestMust(t, p.Close()) }()
	x := bvPropagationVar(t, m, 0, 4)
	y := bvPropagationVar(t, m, 1, 4)
	left := bvPropagationConcat(t, m, bvAddNum(t, m, 15, 4), x)
	right := bvPropagationConcat(t, m, bvAddNum(t, m, 15, 4), y)
	arg := bvPropagationOp(t, m, p, 4, left, right)
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("extract_prop")
	astTestMust(t, p.BindTrace(tags))
	status, err := p.RewriteExtract(5, 1, arg, out)
	astTestMust(t, err)
	if status != Rewrite2 {
		t.Fatal("shortened-add source branch", status)
	}
	if !tags.ShouldTrace("extract_prop") {
		if output.Len() != 0 {
			t.Fatal("erased source trace", output.String())
		}
		return
	}
	want := "-------- [extract_prop] mk_extract_core src/ast/rewriter/bv_rewriter.cpp:776 ---------\n" +
		"(bvadd (concat #xf (:var 0)) (concat #xf (:var 1)))\n[5,1]\n2---->\n" +
		"((_ extract 5 1) (bvadd (concat #b11 (:var 0)) (concat #b11 (:var 1))))\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source propagation record: got %q, want %q", got, want)
	}
	tags.Disable("extract_prop")
	status, err = p.RewriteExtract(5, 1, arg, out)
	astTestMust(t, err)
	if status != Rewrite2 || output.String() != want {
		t.Fatalf("disabled propagation trace: status=%v records=%q", status, output.String())
	}
}
