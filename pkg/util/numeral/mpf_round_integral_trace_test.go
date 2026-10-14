package numeral

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryFloatRoundIntegralTraceBranches(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var x, out ArbitraryFloat
	for _, tc := range []struct {
		name  string
		value float64
		mode  ArbitraryFloatRoundingMode
		lines []string
		parts []string
	}{
		{"zero shortcut", 0, ArbitraryFloatNearestEven, []string{"1050", "1167"}, nil},
		{"small tie even", 0.5, ArbitraryFloatNearestEven, []string{"1050", "1076", "1167"}, []string{"tie = 1\n"}},
		{"small tie away", 0.5, ArbitraryFloatNearestAway, []string{"1050", "1076", "1167"}, []string{"tie = 1\n"}},
		{"general below tie", 1.25, ArbitraryFloatNearestEven, []string{"1050", "1100", "1111", "1112", "1116", "1125", "1167"}, []string{"tie= 0; <tie = 1; >tie = 0\n"}},
		{"general tie", 1.5, ArbitraryFloatNearestEven, []string{"1050", "1100", "1111", "1112", "1116", "1125", "1129", "1167"}, []string{"tie= 1; <tie = 0; >tie = 0\n", "div++ (1)\n"}},
		{"general above tie", 1.75, ArbitraryFloatNearestEven, []string{"1050", "1100", "1111", "1112", "1116", "1125", "1137", "1167"}, []string{"tie= 0; <tie = 0; >tie = 1\n", "div++ (2)\n"}},
	} {
		serviceMust(t, m.SetDouble(&x, 11, 53, tc.value))
		output.Reset()
		serviceMust(t, m.RoundToIntegral(tc.mode, &x, &out))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("%s erased integral trace: %q", tc.name, got)
			}
			continue
		}
		last := -1
		for _, line := range tc.lines {
			header := "-------- [mpf_dbg] round_to_integral src/util/mpf.cpp:" + line + " ---------\n"
			at := strings.Index(got, header)
			if at <= last || strings.Count(got, header) != 1 {
				t.Fatalf("%s integral trace order at %s: %q", tc.name, line, got)
			}
			last = at
		}
		blocks := len(tc.lines)
		if len(tc.lines) > 3 {
			blocks++ // retained unpack TRACE precedes the raw A record
		}
		if strings.Count(got, "------------------------------------------------\n") != blocks {
			t.Fatalf("%s integral trace block count: %q", tc.name, got)
		}
		if len(tc.lines) > 3 && (strings.Index(got, "unpack src/util/mpf.cpp:1923") <= strings.Index(got, "round_to_integral src/util/mpf.cpp:1050") || strings.Index(got, "unpack src/util/mpf.cpp:1923") >= strings.Index(got, "round_to_integral src/util/mpf.cpp:1100") || !strings.Contains(got, "src/util/mpf.cpp:1100 ---------\nA = [") || !strings.Contains(got, "src/util/mpf.cpp:1116 ---------\ndiv=")) {
			t.Fatalf("%s integral raw/quotient records: %q", tc.name, got)
		}
		for _, part := range tc.parts {
			if !strings.Contains(got, part) {
				t.Fatalf("%s integral trace missing %q: %q", tc.name, part, got)
			}
		}
	}
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatRoundIntegralReachedEndlFlushes(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	w := &mpfEndlWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var x, out ArbitraryFloat
	serviceMust(t, m.SetDouble(&x, 11, 53, 1.75))
	w.events = nil
	w.flushes = 0
	serviceMust(t, m.RoundToIntegral(ArbitraryFloatNearestEven, &x, &out))
	if !tags.ShouldTrace("mpf_dbg") {
		if len(w.events) != 0 {
			t.Fatal("erased integral trace reached writer", w.events)
		}
	} else {
		for _, line := range []int{1050, 1100, 1111, 1112, 1116, 1125, 1137, 1167} {
			start := -1
			for i := 0; i+3 < len(w.events); i++ {
				if w.events[i] == "write:src/util/mpf.cpp" && w.events[i+1] == "write::" && w.events[i+2] == "write:"+strconv.Itoa(line) && w.events[i+3] == "write: ---------\n" {
					start = i + 4
					break
				}
			}
			if start < 0 {
				t.Fatalf("missing reached integral site %d: %q", line, w.events)
			}
			bodyFlushes := 0
			for i := start; i < len(w.events) && w.events[i] != "write:------------------------------------------------\n"; i++ {
				if i > start && w.events[i-1] == "write:\n" && w.events[i] == "flush" {
					bodyFlushes++
				}
			}
			if bodyFlushes != 1 {
				t.Fatalf("integral site %d: got %d body flushes, events %q", line, bodyFlushes, w.events)
			}
		}
	}
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatRoundIntegralTraceSourceInsertions(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	w := &mpffArithmeticTraceWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpf_dbg")
	if !tags.ShouldTrace("mpf_dbg") {
		return
	}
	serviceMust(t, m.BindTrace(tags))
	var x, out ArbitraryFloat
	for _, tc := range []struct {
		name    string
		value   float64
		general bool
		tie     []string
	}{
		{"zero shortcut", 0, false, nil},
		{"small tie", 0.5, false, []string{"tie = ", "1", "\n"}},
		{"below tie", 1.25, true, []string{"tie= ", "0", "; <tie = ", "1", "; >tie = ", "0", "\n"}},
		{"at tie", 1.5, true, []string{"tie= ", "1", "; <tie = ", "0", "; >tie = ", "0", "\n"}},
		{"above tie", 1.75, true, []string{"tie= ", "0", "; <tie = ", "0", "; >tie = ", "1", "\n"}},
	} {
		serviceMust(t, m.SetDouble(&x, 11, 53, tc.value))
		w.parts = nil
		serviceMust(t, m.RoundToIntegral(ArbitraryFloatNearestEven, &x, &out))
		for _, label := range []string{"X = ", "INTEGRAL = "} {
			if !hasFloatTraceValueInsertion(w.parts, label) {
				t.Fatalf("%s missing %q insertion in %q", tc.name, label, w.parts)
			}
		}
		if tc.general {
			for _, label := range []string{"A = ", "shift=", "shiftm1_p="} {
				if !hasFloatTraceValueInsertion(w.parts, label) {
					t.Fatalf("%s missing %q insertion in %q", tc.name, label, w.parts)
				}
			}
			foundQuotient := false
			for i := 0; i+4 < len(w.parts); i++ {
				if w.parts[i] == "div=" && w.parts[i+1] != "" && w.parts[i+2] == " rem=" && w.parts[i+3] != "" && w.parts[i+4] == "\n" {
					foundQuotient = true
					break
				}
			}
			if !foundQuotient {
				t.Fatalf("%s missing quotient insertions in %q", tc.name, w.parts)
			}
		}
		if len(tc.tie) != 0 && !hasMPFFTraceParts(w.parts, tc.tie) {
			t.Fatalf("%s missing tie insertions %q in %q", tc.name, tc.tie, w.parts)
		}
	}
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.Del(&out))
}

func hasFloatTraceValueInsertion(parts []string, label string) bool {
	for i := 0; i+2 < len(parts); i++ {
		if parts[i] == label && parts[i+1] != "" && parts[i+1] != "\n" && parts[i+2] == "\n" {
			return true
		}
	}
	return false
}
