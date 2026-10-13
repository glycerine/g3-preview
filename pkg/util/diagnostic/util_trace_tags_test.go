package diagnostic

import (
	"bytes"
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
