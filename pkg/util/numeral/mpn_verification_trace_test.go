package numeral

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMPNDivisionVerificationConditionalTraceBeforeAssertion(t *testing.T) {
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
	if !errors.Is(err, ErrIntegerAssertion) {
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
