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
