package numeral

import (
	"bytes"
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
