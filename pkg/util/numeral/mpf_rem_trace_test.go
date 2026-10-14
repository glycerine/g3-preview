package numeral

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryFloatRemResultTraceSpecialAndGeneral(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	tags.Enable("mpf_dbg_rem")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var x, y, out ArbitraryFloat
	serviceMust(t, m.SetDouble(&y, 11, 53, 2))
	for _, tc := range []struct {
		value float64
		want  string
		inner bool
	}{
		{0, "+zero", false},
		{5, "1.0 0", true},
	} {
		serviceMust(t, m.SetDouble(&x, 11, 53, tc.value))
		output.Reset()
		serviceMust(t, m.Rem(&x, &y, &out))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("erased remainder trace for %v: %q", tc.value, got)
			}
			continue
		}
		result := "-------- [mpf_dbg] rem src/util/mpf.cpp:1484 ---------\nREMAINDER = " + tc.want + "\n"
		at := strings.Index(got, result)
		if at < 0 || strings.Count(got, result) != 1 || strings.Index(got[at:], "------------------------------------------------\n") < 0 {
			t.Fatalf("remainder final source record for %v: %q", tc.value, got)
		}
		if tc.inner && (!strings.Contains(got[:at], "unpack src/util/mpf.cpp:1923") || !strings.Contains(got[:at], "round src/util/mpf.cpp:2096")) || !tc.inner && strings.Contains(got, "unpack src/util/mpf.cpp:1923") {
			t.Fatalf("remainder nested source order for %v: %q", tc.value, got)
		}
		entry := "-------- [mpf_dbg_rem] rem src/util/mpf.cpp:1441 ---------\nX = "
		objectResult := "-------- [mpf_dbg_rem] rem src/util/mpf.cpp:1483 ---------\nR = "
		entryAt, resultAt := strings.Index(got, entry), strings.Index(got, objectResult)
		if entryAt < 0 || resultAt <= entryAt || resultAt >= at || strings.Count(got, "go-words:") < 3 || strings.Count(got, entry) != 1 || strings.Count(got, objectResult) != 1 {
			t.Fatalf("remainder object diagnostic order for %v: %q", tc.value, got)
		}
	}
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.Del(&y))
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatRemObjectReachedEndlFlushes(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	w := &mpfEndlWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpf_dbg_rem")
	serviceMust(t, m.BindTrace(tags))
	var x, y, out ArbitraryFloat
	serviceMust(t, m.SetDouble(&x, 11, 53, 5))
	serviceMust(t, m.SetDouble(&y, 11, 53, 2))
	w.events = nil
	w.flushes = 0
	serviceMust(t, m.Rem(&x, &y, &out))
	if !tags.ShouldTrace("mpf_dbg_rem") {
		if len(w.events) != 0 {
			t.Fatal("erased remainder trace reached writer", w.events)
		}
	} else {
		for _, tc := range []struct{ line, want int }{{1441, 2}, {1483, 1}} {
			start := -1
			for i := 0; i+3 < len(w.events); i++ {
				if w.events[i] == "write:src/util/mpf.cpp" && w.events[i+1] == "write::" && w.events[i+2] == "write:"+strconv.Itoa(tc.line) && w.events[i+3] == "write: ---------\n" {
					start = i + 4
					break
				}
			}
			if start < 0 {
				t.Fatalf("missing remainder object site %d: %q", tc.line, w.events)
			}
			bodyFlushes := 0
			for i := start; i < len(w.events) && w.events[i] != "write:------------------------------------------------\n"; i++ {
				if i > start && w.events[i-1] == "write:\n" && w.events[i] == "flush" {
					bodyFlushes++
				}
			}
			if bodyFlushes != tc.want {
				t.Fatalf("remainder object site %d: got %d body flushes, want %d: %q", tc.line, bodyFlushes, tc.want, w.events)
			}
		}
	}
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.Del(&y))
	serviceMust(t, m.Del(&out))
	const hidden = 1 << 23
	partialX := ArbitraryFloat{format: arbitraryFloatFormat(11, 24, false), exponent: 40, significand: NewInt(hidden)}
	partialY := ArbitraryFloat{format: arbitraryFloatFormat(11, 24, true), exponent: 0, significand: NewInt(hidden + hidden/2)}
	w.events = nil
	serviceMust(t, m.PartialRemainder(&partialX, &partialY, 0, false))
	if tags.ShouldTrace("mpf_dbg_rem") {
		joined := strings.Join(w.events, "|")
		if !strings.Contains(joined, "write:partial remainder = |") || !strings.Contains(joined, "write:\n|flush|write:------------------------------------------------\n|flush") {
			t.Fatalf("partial remainder object std::endl: %q", w.events)
		}
	} else if len(w.events) != 0 {
		t.Fatal("erased partial remainder trace reached writer", w.events)
	}
	serviceMust(t, m.Del(&partialX))
	serviceMust(t, m.Del(&partialY))
}
