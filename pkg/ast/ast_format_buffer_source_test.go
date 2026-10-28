package ast

// Source: src/ast/format.cpp:147-152; native caller buffers.

import (
	"bytes"
	"strings"
	"testing"
)

func TestASTFormatNativeBufferOwnsInternedSpelling(t *testing.T) {
	m := astTestManager(t)
	text := "((_ to_fp 8 24) RTZ -1.25)"
	b := bytes.NewBufferString(text)
	h, e := m.FormatBuffer(b)
	astTestMust(t, e)
	astTestPin(t, m.Format, h)
	// The interner owns its spelling after the caller mutates and reuses the
	// bytes. Native buffers have no simulated closed-owner state.
	b.Bytes()[0] = 'x'
	want, e := m.FormatString(text)
	astTestMust(t, e)
	if h != want {
		t.Fatal("independent native input lifetime", h, want)
	}
	b.Reset()
	b.WriteString("#b" + strings.Repeat("01", 2049))
	b.WriteByte(0)
	b.WriteString("ignored after the source C-string terminator")
	got, e := m.FormatBuffer(b)
	astTestMust(t, e)
	astTestPin(t, m.Format, got)
	want, e = m.FormatString("#b" + strings.Repeat("01", 2049))
	astTestMust(t, e)
	if got != want {
		t.Fatal("large native bytes and symbol terminator", got, want)
	}
	b.Reset()
	empty, e := m.FormatBuffer(b)
	astTestMust(t, e)
	want, e = m.FormatString("")
	astTestMust(t, e)
	if empty != want {
		t.Fatal("reset buffer formats its current empty value", empty, want)
	}
	before := m.Stats()
	if _, e = m.FormatBuffer(nil); e == nil || m.Stats() != before {
		t.Fatal("nil native input guard before formatter mutation", e)
	}
}
