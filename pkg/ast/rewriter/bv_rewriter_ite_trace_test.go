package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:2657-2660; src/util/trace.h:40-46.

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestBVRewriterITECoreSourceTrace(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	bindBVTraceGlobals(t, m)
	p, err := NewBVRewriter(m, c, g, nil)
	astTestMust(t, err)
	defer func() { astTestMust(t, p.Close()) }()
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("bv_ite")
	astTestMust(t, p.BindTrace(tags))
	x := bvMergeVars(t, m, 1)[0]
	one := bvAddNum(t, m, 1, 8)
	two := bvAddNum(t, m, 2, 8)
	three := bvAddNum(t, m, 3, 8)
	condition, err := m.MkEq(x, one)
	astTestMust(t, err)
	astTestPin(t, m, condition)
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	status, err := p.RewriteIte(condition, two, two, out)
	astTestMust(t, err)
	if status != Rewrite1 {
		t.Fatal("equal branches", status)
	}
	status, err = p.RewriteIte(condition, two, three, out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatal("distinct branches", status)
	}
	if !tags.ShouldTrace("bv_ite") {
		if output.Len() != 0 {
			t.Fatal("erased source trace", output.String())
		}
		return
	}
	record := func(body string) string {
		return "-------- [bv_ite] mk_ite_core src/ast/rewriter/bv_rewriter.cpp:2658 ---------\n" + body + "------------------------------------------------\n"
	}
	want := record("mk_ite_core:\n(= (:var 0) #x01)?\n#x02\n:#x02\n") +
		record("mk_ite_core:\n(= (:var 0) #x01)?\n#x02\n:#x03\n")
	if got := output.String(); got != want {
		t.Fatalf("source ITE records: got %q, want %q", got, want)
	}
	tags.Disable("bv_ite")
	status, err = p.RewriteIte(condition, two, two, out)
	astTestMust(t, err)
	if status != Rewrite1 || output.String() != want {
		t.Fatalf("disabled ITE trace: status=%v records=%q", status, output.String())
	}
}
