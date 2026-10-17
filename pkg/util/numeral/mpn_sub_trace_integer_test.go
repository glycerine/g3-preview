package numeral

import (
	"bytes"
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMPNOppositeSignIntegerSourceTraceOrder(t *testing.T) {
	for _, tc := range []struct {
		name, a, b, result, comparison, subtraction string
	}{
		{"greater", "4294967301", "-4294967298", "3", " > ", "4294967301 - 4294967298 == "},
		{"less", "4294967298", "-4294967301", "-3", " < ", "4294967301 - 4294967298 == "},
		{"equal", "4294967301", "-4294967301", "0", " == ", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := integerManager(t)
			var a, b, c Int
			serviceMust(t, m.SetString(&a, tc.a))
			serviceMust(t, m.SetString(&b, tc.b))
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("mpn")
			serviceMust(t, m.BindTrace(tags))
			serviceMust(t, m.Add(&a, &b, &c))
			if got := integerString(t, m, &c); got != tc.result {
				t.Fatalf("result = %s, want %s", got, tc.result)
			}
			if !SourceExtendedGCDDebug || !tags.ShouldTrace("mpn") {
				return
			}
			record := func(function string, line int, body string) string {
				return "-------- [mpn] " + function + " src/util/mpn.cpp:" + strconv.Itoa(line) + " ---------\n" + body + "------------------------------------------------\n"
			}
			want := record("trace", 420, tc.a) + record("compare", 55, tc.comparison) +
				record("trace_nl", 427, tc.b[1:]+"\n")
			if tc.subtraction != "" {
				want += record("trace", 411, "[mpn] "+tc.subtraction) + record("trace_nl", 427, "3\n")
			}
			if got := output.String(); got != want {
				t.Fatalf("source trace = %q, want %q", got, want)
			}
		})
	}
}

func TestMPNSubResultTracePrecedesMPZBorrowAssertion(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	d, _ := bindMPNDebugActions(t, "")
	m, _ := integerManager(t)
	var a, b Int
	serviceMust(t, m.SetString(&a, "4294967301"))
	serviceMust(t, m.SetString(&b, "-4294967298"))
	w := &mpnCloseDebugOnFlush{debug: d, closeAt: 5}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpn")
	if !tags.ShouldTrace("mpn") {
		t.Skip("source MPN TRACE erased")
	}
	serviceMust(t, m.BindTrace(tags))
	c := NewInt(99)
	err := m.Add(&a, &b, &c)
	if !errors.Is(err, arena.ErrClosed) || w.closeErr != nil || w.flushes < 5 || c.Val != 99 || c.Cell != 0 || m.Digit(m.Tmp[0], 0) != 3 {
		t.Fatal("subtraction result trace must precede borrow assertion and destination publication", err, w.closeErr, w.flushes, c)
	}
}
