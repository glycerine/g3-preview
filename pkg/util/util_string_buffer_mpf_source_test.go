// Source: src/util/string_buffer.h:67-76; src/ast/ast_smt2_pp.cpp:262-268.
// Licensed under the MIT license. Source authors: Leonardo de Moura; Christoph Wintersteiger.
package util

import (
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestStringBufferMPFBorrowedCStringLiteralAppendAndLifetime(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	text, e := NewArbitraryFloatFormatText(b)
	paramsMust(t, e)
	paramsMust(t, text.AssignLiteral(strings.Repeat("1", 80)))
	view, e := text.CString()
	paramsMust(t, e)
	out, e := NewStringBuffer64(StringBufferConfig{Budget: b})
	paramsMust(t, e)
	paramsMust(t, out.AppendCString("RTZ "))
	paramsMust(t, out.AppendFloatCString(view))
	got, e := out.CStr()
	paramsMust(t, e)
	if got != "RTZ "+strings.Repeat("1", 80) {
		t.Fatal("literal borrowed bytes", got)
	}
	paramsMust(t, text.SourceFinalize())
	before := out.Stats()
	if out.AppendFloatCString(view) == nil || out.Stats() != before {
		t.Fatal("ended source CString changed buffer")
	}
	paramsMust(t, text.Close())
	paramsMust(t, out.Close())
	if b.Used() != 0 {
		t.Fatal("actual text/buffer owner end", b.Used())
	}
}

func TestStringBufferMPFBorrowedCStringEveryExpansionPrefix(t *testing.T) {
	for ordinal := int64(0); ordinal <= 3; ordinal++ {
		b := arena.NewBudget(1 << 20)
		text, e := NewArbitraryFloatFormatText(b)
		paramsMust(t, e)
		paramsMust(t, text.AssignLiteral(strings.Repeat("7", 300)))
		view, e := text.CString()
		paramsMust(t, e)
		out, e := NewStringBuffer64(StringBufferConfig{Budget: b})
		paramsMust(t, e)
		paramsMust(t, out.AppendCString("prefix "))
		base := b.Used() - out.Stats().OwnerBytes - out.Stats().OverflowBytes
		b.FailAfter(ordinal)
		call := out.AppendFloatCString(view)
		stats := out.Stats()
		if b.Used() != base+stats.OwnerBytes+stats.OverflowBytes {
			t.Fatal("before-lift exact buffer/text cold graph", ordinal, call)
		}
		if call != nil && stats.Position != 7 {
			t.Fatal("source memcpy/position reached after refused expansion", ordinal, stats)
		}
		if call == nil && stats.Position != 307 {
			t.Fatal("completed borrowed append size", stats)
		}
		b.FailAfter(-1)
		if call != nil {
			paramsMust(t, out.AppendFloatCString(view))
		}
		got, e := out.CStr()
		paramsMust(t, e)
		if got != "prefix "+strings.Repeat("7", 300) {
			t.Fatal("source-valid literal retry", ordinal, got)
		}
		paramsMust(t, text.Close())
		paramsMust(t, out.Close())
		if b.Used() != 0 {
			t.Fatal("actual owner end", ordinal, b.Used())
		}
	}
}
