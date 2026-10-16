// Source: src/ast/rewriter/bv_rewriter.cpp:1058-1085; src/util/trace.h:40-46.
package rewriter

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestBVRewriterUnsignedDivisionToMultiplySourceTrace(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	local, err := c.NewValue()
	astTestMust(t, err)
	defer func() { astTestMust(t, c.Del(&local)) }()
	p, err := NewBVRewriter(m, c, g, &local)
	astTestMust(t, err)
	defer func() { astTestMust(t, p.Close()) }()
	x := bvMergeVars(t, m, 1)[0]
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("udiv2mul")
	astTestMust(t, p.BindTrace(tags))
	three := bvAddNum(t, m, 3, 8)
	_, err = p.RewriteUnsignedDivision(x, three, out)
	astTestMust(t, err)
	if output.Len() != 0 {
		t.Fatal("disabled option reached inverse trace", output.String())
	}
	key := "udiv2mul"
	astTestMust(t, c.SetBoolName(&local, &key, true))
	astTestMust(t, p.Update(&local))
	_, err = p.RewriteUnsignedDivision(x, bvAddNum(t, m, 2, 8), out)
	astTestMust(t, err)
	if output.Len() != 0 {
		t.Fatal("power-of-two shortcut reached inverse trace", output.String())
	}
	status, err := p.RewriteUnsignedDivision(x, three, out)
	astTestMust(t, err)
	if status != Rewrite1 {
		t.Fatal("invertible constant branch", status)
	}
	if !tags.ShouldTrace("udiv2mul") {
		if output.Len() != 0 {
			t.Fatal("erased trace emitted", output.String())
		}
		return
	}
	want := "-------- [udiv2mul] mk_bv_udiv_core src/ast/rewriter/bv_rewriter.cpp:1075 ---------\n" +
		"using udiv2mul\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source inverse-option trace: got %q, want %q", got, want)
	}
}
