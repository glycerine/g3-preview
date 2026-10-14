package numeral

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryFloatRoundDecisionTraceBranches(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	for _, tc := range []struct {
		name                           string
		sig                            int32
		exponent                       int64
		mode                           ArbitraryFloatRoundingMode
		increment, overflow, direction string
	}{
		{"halfway even", (1 << 26) + 4, 0, ArbitraryFloatNearestEven, "0", "0", "(LEFT)"},
		{"halfway away", (1 << 26) + 4, 0, ArbitraryFloatNearestAway, "1", "0", "(LEFT)"},
		{"denormal", 1 << 26, -149, ArbitraryFloatNearestEven, "0", "0", "(RIGHT)"},
		{"overflow", 1 << 26, 2000, ArbitraryFloatNearestEven, "0", "1", "(LEFT)"},
	} {
		var x ArbitraryFloat
		sig := NewInt(tc.sig)
		serviceMust(t, m.SetComponents(&x, 8, 24, false, tc.exponent, &sig))
		output.Reset()
		serviceMust(t, m.Round(tc.mode, &x))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("%s erased round trace: %q", tc.name, got)
			}
		} else {
			last := -1
			for _, site := range []struct{ line, body string }{
				{"2000", "e_min_norm = -126\ne_max_norm = 127\nbeta = "},
				{"2009", "Shift distance: "},
				{"2040", "sign=0 last="},
				{"2055", "Rounding increment -> significand +" + tc.increment},
				{"2071", "SIGovf = " + tc.overflow},
				{"2078", "OVF2 = " + tc.overflow},
				{"2079", "o_has_max_exp = " + tc.overflow},
			} {
				header := "-------- [mpf_dbg] round src/util/mpf.cpp:" + site.line + " ---------\n" + site.body
				at := strings.Index(got, header)
				if at <= last || strings.Count(got, "src/util/mpf.cpp:"+site.line+" ---------") != 1 {
					t.Fatalf("%s round source order at %s: %q", tc.name, site.line, got)
				}
				last = at
			}
			if !strings.Contains(got, tc.direction+"\n") || !strings.Contains(got, "sigma_cap = -26\n") || strings.Count(got, "------------------------------------------------\n") != 7 {
				t.Fatalf("%s round record count: %q", tc.name, got)
			}
		}
		serviceMust(t, m.CleanupRound())
		serviceMust(t, m.Del(&x))
	}
}
