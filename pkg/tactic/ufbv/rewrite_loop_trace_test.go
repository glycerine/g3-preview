package ufbv

import (
	"bytes"
	"strconv"
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
	var stackOutput bytes.Buffer
	stackTags := diagnostic.NewTraceTagsWithWriter(&stackOutput)
	stackTags.Enable("demodulator_stack")
	index.trace = stackTags
	out, e = loop.rewrite(index, boolean, a)
	cacheMust(t, e)
	cacheMust(t, m.DelRef(&out))
	if stackTags.ShouldTrace("demodulator_stack") {
		identity := "@" + strconv.FormatUint(a.Origin(), 16) + ":" +
			strconv.FormatUint(uint64(a.Slot()), 16) + ":" +
			strconv.FormatUint(uint64(a.Generation()), 16)
		wantStack := "-------- [demodulator_stack] rewrite src/tactic/ufbv/ufbv_rewriter.cpp:367 ---------\n" +
			"STACK: \n0: " + identity + " = k!903\n" +
			"------------------------------------------------\n"
		if got := stackOutput.String(); got != wantStack {
			t.Fatalf("typed Go stack identity trace:\ngot  %q\nwant %q", got, wantStack)
		}
	} else if stackOutput.Len() != 0 {
		t.Fatal("erased stack trace emitted", stackOutput.String())
	}
	stackRefused := &refusingUFBVTraceWriter{}
	stackRefusedTags := diagnostic.NewTraceTagsWithWriter(stackRefused)
	stackRefusedTags.Enable("demodulator_stack")
	index.trace = stackRefusedTags
	out, e = loop.rewrite(index, boolean, a)
	cacheMust(t, e)
	cacheMust(t, m.DelRef(&out))
	if stackRefusedTags.ShouldTrace("demodulator_stack") && stackRefused.calls != 1 {
		t.Fatal("stack trace stream refusal stopped rewrite", stackRefused.calls)
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

func TestUFBVTraversalSourceRebuiltApplicationTrace(t *testing.T) {
	m := cacheFixture(t)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	f, e := m.MkFuncDecl(symbol.Numerical(910), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	a, e := m.MkConst(symbol.Numerical(911), b)
	cacheMust(t, e)
	c, e := m.MkConst(symbol.Numerical(912), b)
	cacheMust(t, e)
	old, e := m.MkApp(f, []ast.ASTHandle{a})
	cacheMust(t, e)
	rebuilt, e := m.MkApp(f, []ast.ASTHandle{c})
	cacheMust(t, e)
	index, e := newRewriteRuleIndex(m)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, index.close()) })
	loop, e := newRewriteTraversal(m)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, loop.close()) })
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	index.trace = tags
	cacheMust(t, loop.traceRebuilt(index, old, []ast.ASTHandle{c}, rebuilt))
	if output.Len() != 0 {
		t.Fatal("disabled rebuilt trace emitted", output.String())
	}
	tags.Enable("demodulator_bug")
	cacheMust(t, loop.traceRebuilt(index, old, []ast.ASTHandle{c}, rebuilt))
	if !tags.ShouldTrace("demodulator_bug") {
		if output.Len() != 0 {
			t.Fatal("erased rebuilt trace emitted", output.String())
		}
		return
	}
	want := "-------- [demodulator_bug] rewrite src/tactic/ufbv/ufbv_rewriter.cpp:423 ---------\n" +
		"e:\n(k!910 k!911)\nnew_args: \nk!912\n=====>\nna:\n (k!910 k!912)\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source rebuilt application record:\ngot  %q\nwant %q", got, want)
	}
	refused := &refusingUFBVTraceWriter{}
	refusedTags := diagnostic.NewTraceTagsWithWriter(refused)
	refusedTags.Enable("demodulator_bug")
	index.trace = refusedTags
	cacheMust(t, loop.traceRebuilt(index, old, []ast.ASTHandle{c}, rebuilt))
	if refusedTags.ShouldTrace("demodulator_bug") && refused.calls != 1 {
		t.Fatal("rebuilt trace stream refusal did not retain bad state", refused.calls)
	}
}
