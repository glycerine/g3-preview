package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:2543-2564; src/util/trace.h:40-46.

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestBVRewriterEqualityReductionSourceTrace(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	bindBVTraceGlobals(t, m)
	local, err := c.NewValue()
	astTestMust(t, err)
	defer func() { astTestMust(t, c.Del(&local)) }()
	p, err := NewBVRewriter(m, c, g, &local)
	astTestMust(t, err)
	defer func() { astTestMust(t, p.Close()) }()
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mk_mul_eq")
	tags.Enable("eq_remove_trailing")
	astTestMust(t, p.BindTrace(tags))
	xs := bvMergeVars(t, m, 2)
	three := bvAddNum(t, m, 3, 8)
	six := bvAddNum(t, m, 6, 8)
	product := bvMergeMul(t, m, three, xs[0])
	other := bvMergeMul(t, m, three, xs[1])
	sum := bvPropagationOp(t, m, p, 4, bvMergeMul(t, m, bvAddNum(t, m, 2, 8), xs[0]), bvAddNum(t, m, 4, 8))
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	status, err := p.RewriteEquality(product, six, out)
	astTestMust(t, err)
	if status != Rewrite1 {
		t.Fatal("first multiplication equality", status)
	}
	status, err = p.RewriteEquality(sum, other, out)
	astTestMust(t, err)
	if status != Rewrite2 {
		t.Fatal("reverse multiplication equality", status)
	}
	key := "bv_trailing"
	astTestMust(t, c.SetBoolName(&local, &key, true))
	astTestMust(t, p.Update(&local))
	two := bvAddNum(t, m, 2, 8)
	lhs := bvMergeMul(t, m, two, xs[0])
	low, err := m.MkBitVectorExtract(6, 0, xs[0])
	astTestMust(t, err)
	astTestPin(t, m, low)
	rhs := bvPropagationConcat(t, m, low, bvAddNum(t, m, 0, 1))
	status, err = p.RewriteEquality(lhs, rhs, out)
	astTestMust(t, err)
	if status == RewriteFailed {
		t.Fatal("trailing equality did not reduce")
	}
	if !tags.ShouldTrace("mk_mul_eq") {
		if output.Len() != 0 {
			t.Fatal("erased source trace", output.String())
		}
		return
	}
	record := func(tag, line, body string) string {
		return "-------- [" + tag + "] mk_eq_core src/ast/rewriter/bv_rewriter.cpp:" + line + " ---------\n" + body + "------------------------------------------------\n"
	}
	want := record("mk_mul_eq", "2555", "(bvmul #x03 (:var 0))\n=\n#x06\n----->\n(= (:var 0) #x02)\n") +
		record("mk_mul_eq", "2561", "(bvadd (bvmul #x02 (:var 0)) #x04)\n=\n(bvmul #x03 (:var 1))\n----->\n(= (:var 1) (bvmul #xab (bvadd (bvmul #x02 (:var 0)) #x04)))\n") +
		record("eq_remove_trailing", "2548", "(bvmul #x02 (:var 0))\n=\n(concat ((_ extract 6 0) (:var 0)) #b0)\n----->\ntrue\n")
	if got := output.String(); got != want {
		t.Fatalf("source equality records: got %q, want %q", got, want)
	}
	tags.Disable("mk_mul_eq")
	tags.Disable("eq_remove_trailing")
	status, err = p.RewriteEquality(product, six, out)
	astTestMust(t, err)
	if status != Rewrite1 || output.String() != want {
		t.Fatalf("disabled equality trace: status=%v records=%q", status, output.String())
	}
}
