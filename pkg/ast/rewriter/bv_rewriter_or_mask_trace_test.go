// Source: src/ast/rewriter/bv_rewriter.cpp:1675-1710; src/util/trace.h:40-46.
package rewriter

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestBVRewriterORMaskSourceTrace(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	bindBVTraceGlobals(t, m)
	p, err := NewBVRewriter(m, c, g, nil)
	astTestMust(t, err)
	defer func() { astTestMust(t, p.Close()) }()
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mask_bug")
	astTestMust(t, p.BindTrace(tags))
	xs := bvMergeVars(t, m, 2)
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	rewrite := func(args ...ast.ASTHandle) RewriteStatus {
		t.Helper()
		status, e := p.RewriteOR(ast.SliceArguments(args), out)
		astTestMust(t, e)
		return status
	}
	if status := rewrite(xs[0], xs[1]); status != RewriteFailed {
		t.Fatal("nonmask status", status)
	}
	if status := rewrite(xs[0], bvAddNum(t, m, 255, 8)); status != RewriteDone {
		t.Fatal("all-one status", status)
	}
	if status := rewrite(xs[0], bvAddNum(t, m, 9, 8)); status != Rewrite2 {
		t.Fatal("mask status", status)
	}
	if !tags.ShouldTrace("mask_bug") {
		if output.Len() != 0 {
			t.Fatal("erased source trace", output.String())
		}
		return
	}
	want := "-------- [mask_bug] mk_bv_or src/ast/rewriter/bv_rewriter.cpp:1707 ---------\n" +
		"(assert (distinct (bvor (_ bv9 8)\n(:var 0))\n(concat ((_ extract 7 4) (:var 0)) #b1 ((_ extract 2 1) (:var 0)) #b1)))\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source OR mask record: got %q, want %q", got, want)
	}
	tags.Disable("mask_bug")
	if status := rewrite(xs[0], bvAddNum(t, m, 9, 8)); status != Rewrite2 || output.String() != want {
		t.Fatalf("disabled OR mask trace: status=%v records=%q", status, output.String())
	}
}
