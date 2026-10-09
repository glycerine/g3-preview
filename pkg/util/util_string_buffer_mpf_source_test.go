// Source: src/util/string_buffer.h:67-76; src/ast/ast_smt2_pp.cpp:262-268.
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
	out, e := NewStringBuffer64(StringBufferConfig{})
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

func TestStringBufferMPFBorrowedCStringGoBufferGrowth(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	text, e := NewArbitraryFloatFormatText(b)
	paramsMust(t, e)
	paramsMust(t, text.AssignLiteral(strings.Repeat("7", 300)))
	view, e := text.CString()
	paramsMust(t, e)
	out, e := NewStringBuffer64(StringBufferConfig{})
	paramsMust(t, e)
	paramsMust(t, out.AppendCString("prefix "))
	before := b.Used()
	b.FailAfter(0)
	paramsMust(t, out.AppendFloatCString(view))
	if b.Used() != before || out.Position() != 307 {
		t.Fatal("Go buffer touched numeric arena", b.Used(), before, out.Stats())
	}
	b.FailAfter(-1)
	got, e := out.CStr()
	paramsMust(t, e)
	if got != "prefix "+strings.Repeat("7", 300) {
		t.Fatal("source-valid literal", got)
	}
	paramsMust(t, text.Close())
	paramsMust(t, out.Close())
	if b.Used() != 0 {
		t.Fatal("actual owner end", b.Used())
	}
}
