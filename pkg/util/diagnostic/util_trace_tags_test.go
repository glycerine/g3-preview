package diagnostic

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestTraceTagsSourceControlAndFinalize(t *testing.T) {
	tags := NewTraceTags()
	if tags.IsEnabled("alpha") {
		t.Fatal("cold tags")
	}
	tags.Enable("alpha\x00ignored")
	tags.Enable("alpha")
	if !tags.IsEnabled("alpha") || tags.IsEnabled("beta") || tags.IsEnabled("alpha-x") {
		t.Fatal("C-string insertion and duplicate")
	}
	tags.EnableAll(true)
	tags.Disable("alpha")
	if !tags.IsEnabled("alpha") || !tags.IsEnabled("beta") {
		t.Fatal("all flag precedes tag lookup")
	}
	tags.EnableAll(false)
	if tags.IsEnabled("alpha") || tags.IsEnabled("beta") {
		t.Fatal("disabled tag and all flag")
	}
	tags.Enable("gamma")
	tags.EnableAll(true)
	tags.Finalize()
	if !tags.IsEnabled("gamma") || tags.tags != nil {
		t.Fatal("finalize preserves all flag but releases table")
	}
	tags.EnableAll(false)
	if tags.IsEnabled("gamma") {
		t.Fatal("finalized table retained tag")
	}
	tags.Disable("missing")
	if tags.tags == nil || len(tags.tags) != 0 {
		t.Fatal("source disable initializes empty table")
	}
}

func TestTraceEmissionCompileGateAndSourceRecord(t *testing.T) {
	var output bytes.Buffer
	tags := NewTraceTagsWithWriter(&output)
	tags.Enable("gparams")
	if !traceCompiled && tags.ShouldTrace("gparams") {
		t.Fatal("erased TRACE_CODE evaluated its tag")
	}
	if e := tags.Emit("gparams", "set", "src/util/gparams.cpp", 563, "setting [x] <- '7'\n"); e != nil {
		t.Fatal(e)
	}
	if !traceCompiled {
		if output.Len() != 0 {
			t.Fatal("erased TRACE_CODE wrote", output.String())
		}
		return
	}
	want := "-------- [gparams] set src/util/gparams.cpp:563 ---------\nsetting [x] <- '7'\n------------------------------------------------\n"
	if output.String() != want {
		t.Fatal("source TRACE record", output.String())
	}
	flushed := &flushedWarningWriter{}
	other := NewTraceTagsWithWriter(flushed)
	other.Enable("gparams")
	if e := other.Emit("gparams", "set", "src/util/gparams.cpp", 563, "x\n"); e != nil || flushed.flushes != 1 {
		t.Fatal("TRACE must flush its completed record", e, flushed.flushes)
	}
	tags.Disable("gparams")
	if e := tags.Emit("gparams", "set", "src/util/gparams.cpp", 563, "disabled\n"); e != nil || output.String() != want {
		t.Fatal("disabled tag emitted", e, output.String())
	}
}

func TestTraceShortAndConditionalGateOrder(t *testing.T) {
	var output bytes.Buffer
	tags := NewTraceTagsWithWriter(&output)
	conditions, bodies := 0, 0
	condition := func() bool { conditions++; return true }
	body := func(w io.Writer) error { bodies++; return writeTracePart(w, "body\n") }
	if e := tags.EmitShort("cooperate", body); e != nil {
		t.Fatal(e)
	}
	orderedBody := func(w io.Writer) error {
		if !strings.HasSuffix(output.String(), "-------- [cooperate] checkpoint src/util/cooperate.cpp:57 ---------\n") {
			t.Fatal("CTRACE body ran before header", output.String())
		}
		return body(w)
	}
	if e := tags.EmitConditional("cooperate", "checkpoint", "src/util/cooperate.cpp", 57, condition, orderedBody); e != nil {
		t.Fatal(e)
	}
	if conditions != 0 || bodies != 0 {
		t.Fatal("disabled tag evaluated CODE", conditions, bodies)
	}
	tags.Enable("cooperate")
	falseCondition := func() bool { conditions++; return false }
	if e := tags.EmitConditional("cooperate", "checkpoint", "src/util/cooperate.cpp", 57, falseCondition, body); e != nil {
		t.Fatal(e)
	}
	if e := tags.EmitShort("cooperate", body); e != nil {
		t.Fatal(e)
	}
	if e := tags.EmitConditional("cooperate", "checkpoint", "src/util/cooperate.cpp", 57, condition, orderedBody); e != nil {
		t.Fatal(e)
	}
	if !traceCompiled {
		if conditions != 0 || bodies != 0 || output.Len() != 0 {
			t.Fatal("erased trace evaluated source code", conditions, bodies, output.String())
		}
		return
	}
	want := "body\n-------- [cooperate] checkpoint src/util/cooperate.cpp:57 ---------\nbody\n------------------------------------------------\n"
	if conditions != 2 || bodies != 2 || output.String() != want {
		t.Fatal("STRACE/CTRACE gate or output", conditions, bodies, output.String())
	}
	flushed := &flushedWarningWriter{}
	short := NewTraceTagsWithWriter(flushed)
	short.Enable("cooperate")
	if e := short.EmitShort("cooperate", func(w io.Writer) error { return writeTracePart(w, "x") }); e != nil || flushed.flushes != 1 {
		t.Fatal("STRACE flush", e, flushed.flushes)
	}
}
