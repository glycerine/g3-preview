// Source: src/ast/format.cpp:147-152; src/util/string_buffer.h:62-65.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import "testing"

func TestASTFormatBorrowedBufferUsesActualStringEngine(t *testing.T) {
	m := astTestManager(t)
	defer func() { astTestMust(t, m.Close()) }()
	b, e := NewStringBuffer64(StringBufferConfig{Budget: m.budget})
	astTestMust(t, e)
	defer func() { astTestMust(t, b.Close()) }()
	astTestMust(t, b.AppendCString("((_ to_fp 8 24) RTZ "))
	astTestMust(t, b.AppendCString("-1.25)"))
	h, e := m.FormatBuffer64(b)
	astTestMust(t, e)
	want, e := m.FormatString("((_ to_fp 8 24) RTZ -1.25)")
	astTestMust(t, e)
	if h != want {
		t.Fatal("same actual intern/register engine", h, want)
	}
	astTestMust(t, b.Close())
	before := m.Stats()
	if _, e = m.FormatBuffer64(b); e == nil || m.Stats() != before {
		t.Fatal("ended borrowed buffer must refuse before formatter mutation", e)
	}
}
