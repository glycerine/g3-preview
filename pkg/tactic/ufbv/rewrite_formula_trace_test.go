package ufbv

import (
	"bytes"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type refusingUFBVTraceWriter struct{ calls int }

func (w *refusingUFBVTraceWriter) Write([]byte) (int, error) {
	w.calls++
	return 0, io.ErrClosedPipe
}

func TestUFBVFormulaSourceBeforePublishedAndAfterTraces(t *testing.T) {
	m := cacheFixture(t)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	name, e := m.Symbols.Intern("trace_atom")
	cacheMust(t, e)
	a, e := m.MkConst(name, b)
	cacheMust(t, e)
	cacheMust(t, m.IncRef(a))
	t.Cleanup(func() { cacheMust(t, m.DecRef(a)) })
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	processor, e := newFormulaProcessor(m, ufbvBooleanFixture(t, m))
	cacheMust(t, e)
	processor.trace = tags
	t.Cleanup(func() { cacheMust(t, processor.close()) })
	tags.Enable("demodulator")
	result, e := processor.run([]ast.ASTHandle{a})
	cacheMust(t, e)
	if len(result) != 1 || result[0] != a {
		t.Fatal("source unchanged-formula publication", result)
	}
	for _, h := range result {
		cacheMust(t, m.DecRef(h))
	}
	if !tags.ShouldTrace("demodulator") {
		if output.Len() != 0 {
			t.Fatal("compiled-out trace emitted", output.String())
		}
		return
	}
	frame := func(line, body string) string {
		return "-------- [demodulator] operator() src/tactic/ufbv/ufbv_rewriter.cpp:" + line + " ---------\n" + body + "------------------------------------------------\n"
	}
	want := frame("668", "before demodulator:\ntrace_atom\n") +
		frame("736", "trace_atom\n") +
		frame("753", "after demodulator:\ntrace_atom\n")
	if got := output.String(); got != want {
		t.Fatalf("source formula trace order and text:\ngot  %q\nwant %q", got, want)
	}
}

func TestUFBVFormulaTraceWriterRefusalDoesNotStopRewrite(t *testing.T) {
	m := cacheFixture(t)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	name, e := m.Symbols.Intern("refused_trace_atom")
	cacheMust(t, e)
	a, e := m.MkConst(name, b)
	cacheMust(t, e)
	cacheMust(t, m.IncRef(a))
	t.Cleanup(func() { cacheMust(t, m.DecRef(a)) })
	w := &refusingUFBVTraceWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("demodulator")
	r, e := newFormulaProcessor(m, ufbvBooleanFixture(t, m))
	cacheMust(t, e)
	r.trace = tags
	t.Cleanup(func() { cacheMust(t, r.close()) })
	result, e := r.run([]ast.ASTHandle{a})
	cacheMust(t, e)
	if len(result) != 1 || result[0] != a {
		t.Fatal("source rewrite stopped after trace stream refusal", result)
	}
	cacheMust(t, m.DecRef(result[0]))
	if tags.ShouldTrace("demodulator") && w.calls != 1 {
		t.Fatal("failed source trace stream did not retain its bad state", w.calls)
	}
}
