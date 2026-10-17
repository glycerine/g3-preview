package numeral

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMPNDivisionVerificationConditionalTraceBeforeAssertion(t *testing.T) {
	bindMPNDebugActions(t, "T")
	m := naturalManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpn_dbg")
	m.trace = tags
	view := func(words []uint32) limbView { return limbView{external: words, size: uint32(len(words))} }
	a, b, q := view([]uint32{7}), view([]uint32{3}), view([]uint32{2})
	if err := m.sourceVerifyDivision(a, b, q, view([]uint32{1})); err != nil || output.Len() != 0 {
		t.Fatal("valid division must not emit the conditional record", err, output.String())
	}
	err := m.sourceVerifyDivision(a, b, q, view([]uint32{2}))
	if !modularSourceDebug {
		if err != nil || output.Len() != 0 {
			t.Fatal("erased debug verification", err, output.String())
		}
		return
	}
	if !isMPNVerificationThrow(err) {
		t.Fatal("verification assertion", err)
	}
	if !tags.ShouldTrace("mpn_dbg") {
		if output.Len() != 0 {
			t.Fatal("erased CTRACE emitted", output.String())
		}
		return
	}
	want := "-------- [mpn_dbg] div src/util/mpn.cpp:215 ---------\n" +
		"DIV BUG: quot * denom + rem = [0|8]\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source conditional verification record got %q, want %q", got, want)
	}
}

func TestMPNDivisionVerificationConditionalTraceEndlRefusal(t *testing.T) {
	bindMPNDebugActions(t, "T")
	m := naturalManager(t)
	w := &mpfEndlWriter{failAt: 1, refusal: errors.New("verification endl refused")}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpn_dbg")
	m.trace = tags
	view := func(words []uint32) limbView { return limbView{external: words, size: uint32(len(words))} }
	err := m.sourceVerifyDivision(view([]uint32{7}), view([]uint32{3}), view([]uint32{2}), view([]uint32{2}))
	if !modularSourceDebug {
		if err != nil || len(w.events) != 0 {
			t.Fatal("erased verification", err, w.events)
		}
		return
	}
	if !isMPNVerificationThrow(err) {
		t.Fatal("trace flush changed source assertion", err)
	}
	if !tags.ShouldTrace("mpn_dbg") {
		if len(w.events) != 0 {
			t.Fatal("erased conditional trace wrote", w.events)
		}
		return
	}
	if w.flushes != 2 {
		t.Fatal("body endl and TRACE footer flushes", w.flushes, w.events)
	}
	want := []string{"write:DIV BUG: quot * denom + rem = ", "write:[", "write:0", "write:|", "write:8", "write:]", "write:\n", "flush"}
	if !hasMPFFTraceParts(w.events, want) {
		t.Fatal("conditional trace insertions and endl", w.events)
	}
	flushed := false
	for _, event := range w.events {
		if event == "flush" {
			flushed = true
		} else if flushed {
			t.Fatal("physical write after refused endl", w.events)
		}
	}
}
