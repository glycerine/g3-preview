package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:2279-2304; src/util/trace.h:40-46.

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestBVRewriterBlastEqualitySourceTrace(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	bindBVTraceGlobals(t, m)
	p, err := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, err)
	defer func() { astTestMust(t, p.Close()) }()
	extract, err := NewMkExtractProc(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, extract.Close()) }()
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("blast_eq_value")
	astTestMust(t, p.BindTrace(tags))
	xs := bvMergeVars(t, m, 2)
	bit := bvAddNum(t, m, 1, 1)
	rhs := bvAddNum(t, m, 3, 8)
	or, err := m.MkBitVectorOr(xs)
	astTestMust(t, err)
	astTestPin(t, m, or)
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	status, err := p.RewriteBlastEqualityValue(bit, ast.ASTHandle{}, out, nil)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatal("width-one guard", status)
	}
	status, err = p.RewriteBlastEqualityValue(xs[0], xs[1], out, nil)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatal("non-numeral guard", status)
	}
	status, err = p.RewriteBlastEqualityValue(or, rhs, out, extract)
	astTestMust(t, err)
	if status != Rewrite3 {
		t.Fatal("blast success", status)
	}
	status, err = p.RewriteBlastEqualityValue(rhs, or, out, extract)
	astTestMust(t, err)
	if status != Rewrite3 {
		t.Fatal("blast after swap", status)
	}
	if !tags.ShouldTrace("blast_eq_value") {
		if output.Len() != 0 {
			t.Fatal("erased source trace", output.String())
		}
		return
	}
	record := func(body string) string {
		return "-------- [blast_eq_value] mk_blast_eq_value src/ast/rewriter/bv_rewriter.cpp:2283 ---------\n" + body + "------------------------------------------------\n"
	}
	want := record("sz: 8\n(:var 0)\n") + record("sz: 8\n(bvor (:var 0) (:var 1))\n") + record("sz: 8\n#x03\n")
	if got := output.String(); got != want {
		t.Fatalf("source blast records: got %q, want %q", got, want)
	}
	tags.Disable("blast_eq_value")
	status, err = p.RewriteBlastEqualityValue(or, rhs, out, extract)
	astTestMust(t, err)
	if status != Rewrite3 || output.String() != want {
		t.Fatalf("disabled blast trace: status=%v records=%q", status, output.String())
	}
	local, err := c.NewValue()
	astTestMust(t, err)
	defer func() { astTestMust(t, c.Del(&local)) }()
	key := "blast_eq_value"
	astTestMust(t, c.SetBoolName(&local, &key, true))
	outer, err := NewBVRewriter(m, c, g, &local)
	astTestMust(t, err)
	defer func() { astTestMust(t, outer.Close()) }()
	tags.Enable("blast_eq_value")
	astTestMust(t, outer.BindTrace(tags))
	status, err = outer.RewriteEquality(or, rhs, out)
	astTestMust(t, err)
	if status != Rewrite3 {
		t.Fatal("outer equality dispatch", status)
	}
	if got := output.String(); got != want+record("sz: 8\n(bvor (:var 0) (:var 1))\n") {
		t.Fatalf("outer equality trace: got %q", got)
	}
}
