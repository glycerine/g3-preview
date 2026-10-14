package numeral

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryFloatDivDebugTraceSourceOrderAndShortcut(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var one, zero, out ArbitraryFloat
	serviceMust(t, m.MkOne(11, 53, false, &one))
	serviceMust(t, m.MkZero(11, 53, false, &zero))
	serviceMust(t, m.Div(ArbitraryFloatNearestEven, &one, &one, &out))
	if compiled {
		got := output.String()
		last := -1
		for _, site := range []struct {
			line, label string
		}{{"670", "X = "}, {"671", "Y = "}, {"722", "A = "}, {"723", "B = "}, {"727", "A' = "}, {"728", "B' = "}, {"734", "QUOTIENT = "}, {"742", "QUOTIENT' = "}} {
			entry := "-------- [mpf_dbg] div src/util/mpf.cpp:" + site.line + " ---------\n" + site.label
			at := strings.Index(got, entry)
			if at <= last {
				t.Fatalf("division debug source order at %s: %q", site.line, got)
			}
			last = at
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased division debug trace: %q", output.String())
	}
	output.Reset()
	serviceMust(t, m.Div(ArbitraryFloatNearestEven, &zero, &one, &out))
	if compiled {
		got := output.String()
		if strings.Count(got, "-------- [mpf_dbg] div src/util/mpf.cpp:") != 2 || strings.Contains(got, "QUOTIENT = ") {
			t.Fatalf("zero shortcut reached quotient debug trace: %q", got)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased shortcut debug trace: %q", output.String())
	}
	serviceMust(t, m.Del(&one))
	serviceMust(t, m.Del(&zero))
	serviceMust(t, m.Del(&out))
}
