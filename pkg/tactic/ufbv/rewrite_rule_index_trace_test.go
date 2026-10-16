package ufbv

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestUFBVForwardIndexSourceInsertTrace(t *testing.T) {
	m := cacheFixture(t)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	x, e := m.MkVar(0, b)
	cacheMust(t, e)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{b},
		Names: []symbol.Symbol{symbol.Numerical(901)}, Body: x,
	})
	cacheMust(t, e)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	r, e := newRewriteRuleIndex(m)
	cacheMust(t, e)
	r.trace = tags
	t.Cleanup(func() { cacheMust(t, r.close()) })
	cacheMust(t, r.traceInsert(q))
	if output.Len() != 0 {
		t.Fatal("disabled insertion trace emitted", output.String())
	}
	tags.Enable("demodulator_fwd")
	cacheMust(t, r.traceInsert(q))
	if !tags.ShouldTrace("demodulator_fwd") {
		if output.Len() != 0 {
			t.Fatal("erased insertion trace emitted", output.String())
		}
		return
	}
	want := "-------- [demodulator_fwd] insert_fwd_idx src/tactic/ufbv/ufbv_rewriter.cpp:220 ---------\n" +
		"INSERT: (forall ((x!1 Bool)) x!1)\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source insertion trace:\ngot  %q\nwant %q", got, want)
	}
	refused := &refusingUFBVTraceWriter{}
	refusedTags := diagnostic.NewTraceTagsWithWriter(refused)
	refusedTags.Enable("demodulator_fwd")
	r.trace = refusedTags
	cacheMust(t, r.traceInsert(q))
	if refusedTags.ShouldTrace("demodulator_fwd") && refused.calls != 1 {
		t.Fatal("insertion stream refusal did not retain bad state", refused.calls)
	}
}

func TestUFBVForwardIndexSourceMatchingTraces(t *testing.T) {
	m := cacheFixture(t)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	x, e := m.MkVar(0, b)
	cacheMust(t, e)
	f, e := m.MkFuncDecl(symbol.Numerical(801), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	g, e := m.MkFuncDecl(symbol.Numerical(802), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	a, e := m.MkConst(symbol.Numerical(803), b)
	cacheMust(t, e)
	cacheMust(t, m.IncRef(a))
	t.Cleanup(func() { cacheMust(t, m.DecRef(a)) })
	lhs, e := m.MkApp(f, []ast.ASTHandle{x})
	cacheMust(t, e)
	rhs, e := m.MkApp(g, []ast.ASTHandle{x})
	cacheMust(t, e)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{b},
		Names: []symbol.Symbol{symbol.Numerical(804)}, Body: x,
	})
	cacheMust(t, e)
	r, e := newRewriteRuleIndex(m)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, r.close()) })
	cacheMust(t, r.insert(q, lhs, rhs))
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("demodulator_bug")
	r.trace = tags
	out, matched, e := r.tryRewrite(f, []ast.ASTHandle{a})
	cacheMust(t, e)
	if !matched {
		t.Fatal("source rule did not match")
	}
	cacheMust(t, m.DelRef(&out))
	if !tags.ShouldTrace("demodulator_bug") {
		if output.Len() != 0 {
			t.Fatal("erased matching trace emitted", output.String())
		}
		return
	}
	frame := func(line, body string) string {
		return "-------- [demodulator_bug] rewrite1 src/tactic/ufbv/ufbv_rewriter.cpp:" + line + " ---------\n" + body + "------------------------------------------------\n"
	}
	want := frame("290", "trying to rewrite: k!801 args:\nk!803\n") +
		frame("305", "Matching with demodulator: (forall ((x!1 Bool)) x!1)\n") +
		frame("310", "succeeded...\n(k!802 (:var 0))\n===>\n(k!802 k!803)\n")
	if got := output.String(); got != want {
		t.Fatalf("source matching traces:\ngot  %q\nwant %q", got, want)
	}
	refused := &refusingUFBVTraceWriter{}
	refusedTags := diagnostic.NewTraceTagsWithWriter(refused)
	refusedTags.Enable("demodulator_bug")
	r.trace = refusedTags
	out, matched, e = r.tryRewrite(f, []ast.ASTHandle{a})
	cacheMust(t, e)
	if !matched {
		t.Fatal("source stream refusal stopped rule match")
	}
	cacheMust(t, m.DelRef(&out))
	if refusedTags.ShouldTrace("demodulator_bug") && refused.calls != 1 {
		t.Fatal("matching stream refusal did not retain bad state", refused.calls)
	}
}
