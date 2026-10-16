package ufbv

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestUFBVReschedulingSourceTraceRecord(t *testing.T) {
	m := cacheFixture(t)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	x, e := m.MkVar(0, b)
	cacheMust(t, e)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{b},
		Names: []symbol.Symbol{symbol.Numerical(921)}, Body: x,
	})
	cacheMust(t, e)
	index, e := newRewriteRuleIndex(m)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, index.close()) })
	back, e := newRewriteBackIndex(m)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, back.close()) })
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	index.trace = tags
	cacheMust(t, back.traceRescheduling(index, q))
	if output.Len() != 0 {
		t.Fatal("disabled rescheduling trace emitted", output.String())
	}
	tags.Enable("demodulator")
	cacheMust(t, back.traceRescheduling(index, q))
	if !tags.ShouldTrace("demodulator") {
		if output.Len() != 0 {
			t.Fatal("erased rescheduling trace emitted", output.String())
		}
		return
	}
	want := "-------- [demodulator] reschedule_demodulators src/tactic/ufbv/ufbv_rewriter.cpp:637 ---------\n" +
		"Rescheduling: \n(forall ((x!1 Bool)) x!1)\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source rescheduling trace:\ngot  %q\nwant %q", got, want)
	}
	refused := &refusingUFBVTraceWriter{}
	refusedTags := diagnostic.NewTraceTagsWithWriter(refused)
	refusedTags.Enable("demodulator")
	index.trace = refusedTags
	cacheMust(t, back.traceRescheduling(index, q))
	if refusedTags.ShouldTrace("demodulator") && refused.calls != 1 {
		t.Fatal("rescheduling trace stream refusal did not retain bad state", refused.calls)
	}
}
