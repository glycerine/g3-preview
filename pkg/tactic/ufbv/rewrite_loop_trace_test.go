package ufbv

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestUFBVTraversalSourceEntryAndResultTraces(t *testing.T) {
	m := cacheFixture(t)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	x, e := m.MkVar(0, b)
	cacheMust(t, e)
	f, e := m.MkFuncDecl(symbol.Numerical(902), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	a, e := m.MkConst(symbol.Numerical(903), b)
	cacheMust(t, e)
	cacheMust(t, m.IncRef(a))
	t.Cleanup(func() { cacheMust(t, m.DecRef(a)) })
	lhs, e := m.MkApp(f, []ast.ASTHandle{x})
	cacheMust(t, e)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{b},
		Names: []symbol.Symbol{symbol.Numerical(904)}, Body: x,
	})
	cacheMust(t, e)
	index, e := newRewriteRuleIndex(m)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, index.close()) })
	cacheMust(t, index.insert(q, lhs, x))
	loop, e := newRewriteTraversal(m)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, loop.close()) })
	boolean := ufbvBooleanFixture(t, m)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	index.trace = tags
	out, e := loop.rewrite(index, boolean, a)
	cacheMust(t, e)
	cacheMust(t, m.DelRef(&out))
	if output.Len() != 0 {
		t.Fatal("disabled traversal trace emitted", output.String())
	}
	tags.Enable("demodulator")
	out, e = loop.rewrite(index, boolean, a)
	cacheMust(t, e)
	cacheMust(t, m.DelRef(&out))
	if !tags.ShouldTrace("demodulator") {
		if output.Len() != 0 {
			t.Fatal("erased traversal trace emitted", output.String())
		}
		return
	}
	want := "-------- [demodulator] rewrite src/tactic/ufbv/ufbv_rewriter.cpp:359 ---------\n" +
		"rewrite: k!903\n------------------------------------------------\n" +
		"-------- [demodulator] rewrite src/tactic/ufbv/ufbv_rewriter.cpp:462 ---------\n" +
		"rewrite result: k!903\n------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source traversal records:\ngot  %q\nwant %q", got, want)
	}
	refused := &refusingUFBVTraceWriter{}
	refusedTags := diagnostic.NewTraceTagsWithWriter(refused)
	refusedTags.Enable("demodulator")
	index.trace = refusedTags
	out, e = loop.rewrite(index, boolean, a)
	cacheMust(t, e)
	cacheMust(t, m.DelRef(&out))
	if refusedTags.ShouldTrace("demodulator") && refused.calls != 1 {
		t.Fatal("traversal stream refusal did not retain bad state", refused.calls)
	}
	var prefix bytes.Buffer
	prefixTags := diagnostic.NewTraceTagsWithWriter(&prefix)
	prefixTags.Enable("demodulator")
	index.trace = prefixTags
	if e := loop.traceRewrite(index, 359, "rewrite: ", ast.ASTHandle{}); e == nil {
		t.Fatal("invalid operand passed source trace formatter")
	}
	if prefixTags.ShouldTrace("demodulator") && !strings.HasSuffix(prefix.String(), "rewrite: ") {
		t.Fatal("source heading must precede failed operand formatting", prefix.String())
	}
}
