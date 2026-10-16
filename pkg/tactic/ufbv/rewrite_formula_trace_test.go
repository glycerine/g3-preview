package ufbv

import (
	"bytes"
	"io"
	"strings"
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

func TestUFBVFoundDemodulatorSourceLongAndShortRecords(t *testing.T) {
	m := cacheFixture(t)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	aName, e := m.Symbols.Intern("lhs_const")
	cacheMust(t, e)
	bName, e := m.Symbols.Intern("rhs_const")
	cacheMust(t, e)
	a, e := m.MkConst(aName, b)
	cacheMust(t, e)
	c, e := m.MkConst(bName, b)
	cacheMust(t, e)
	for _, h := range []ast.ASTHandle{a, c} {
		cacheMust(t, m.IncRef(h))
		t.Cleanup(func() { cacheMust(t, m.DecRef(h)) })
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("demodulator")
	tags.Enable("demodulator_s")
	r, e := newFormulaProcessor(m, ufbvBooleanFixture(t, m))
	cacheMust(t, e)
	r.trace = tags
	t.Cleanup(func() { cacheMust(t, r.close()) })
	cacheMust(t, r.traceFoundDemodulator(a, c))
	if !tags.ShouldTrace("demodulator") {
		if output.Len() != 0 {
			t.Fatal("erased found-demodulator trace emitted", output.String())
		}
		return
	}
	want := "-------- [demodulator] operator() src/tactic/ufbv/ufbv_rewriter.cpp:702 ---------\n" +
		"Found demodulator: \nlhs_const\n ---> \nrhs_const\n" +
		"------------------------------------------------\n" +
		"-------- [demodulator_s] operator() src/tactic/ufbv/ufbv_rewriter.cpp:706 ---------\n" +
		"Found demodulator: \nlhs_const[1] ---> rhs_const[1]\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source found-demodulator records:\ngot  %q\nwant %q", got, want)
	}
	at := strings.Index(want, "-------- [demodulator_s]")
	longRecord, shortRecord := want[:at], want[at:]
	output.Reset()
	tags.Disable("demodulator")
	cacheMust(t, r.traceFoundDemodulator(a, c))
	if got := output.String(); got != shortRecord {
		t.Fatal("short tag must emit independently", got, shortRecord)
	}
	output.Reset()
	tags.Enable("demodulator")
	tags.Disable("demodulator_s")
	cacheMust(t, r.traceFoundDemodulator(a, c))
	if got := output.String(); got != longRecord {
		t.Fatal("long tag must emit independently", got, longRecord)
	}
	variable, e := m.MkVar(0, b)
	cacheMust(t, e)
	cacheMust(t, m.IncRef(variable))
	t.Cleanup(func() { cacheMust(t, m.DecRef(variable)) })
	output.Reset()
	tags.Disable("demodulator")
	tags.Enable("demodulator_s")
	cacheMust(t, r.traceFoundDemodulator(a, variable))
	variableText, e := r.traceFormulaText(variable)
	cacheMust(t, e)
	want = "-------- [demodulator_s] operator() src/tactic/ufbv/ufbv_rewriter.cpp:706 ---------\n" +
		"Found demodulator: \nlhs_const[1] ---> " + variableText + "\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatal("short trace non-application RHS branch", got, want)
	}
}
