package numeral

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
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
				{"1967", "RND: "},
				{"2000", "e_min_norm = -126\ne_max_norm = 127\nbeta = "},
				{"2009", "Shift distance: "},
				{"2026", "Shifted: "},
				{"2040", "sign=0 last="},
				{"2041", "before rounding decision: "},
				{"2055", "Rounding increment -> significand +" + tc.increment},
				{"2059", "Rounded significand: "},
				{"2070", "Post-normalized: "},
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
			blocks := 14
			branch := "round src/util/mpf.cpp:2087"
			if tc.direction == "(RIGHT)" {
				branch = "round src/util/mpf.cpp:2091"
			}
			if tc.overflow == "1" {
				blocks = 13
			} else if at := strings.Index(got, branch); at <= last {
				t.Fatalf("%s result class after decisions: %q", tc.name, got)
			}
			if at := strings.Index(got, "round src/util/mpf.cpp:2096 ---------\nROUNDED = "); at <= last {
				t.Fatalf("%s final round result: %q", tc.name, got)
			}
			if !strings.Contains(got, " (hidden bit, unbiased exp).\n") || strings.Count(got, "#b") < 6 {
				t.Fatalf("%s retained binary result fields: %q", tc.name, got)
			}
			if !strings.Contains(got, tc.direction+"\n") || !strings.Contains(got, "sigma_cap = -26\n") || strings.Count(got, "------------------------------------------------\n") != blocks {
				t.Fatalf("%s round record count: %q", tc.name, got)
			}
		}
		serviceMust(t, m.CleanupRound())
		serviceMust(t, m.Del(&x))
	}
}

func TestArbitraryFloatRoundResultClassTrace(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	for _, tc := range []struct {
		name        string
		exponent    int64
		line, label string
	}{
		{"normal", 0, "2087", "NORMAL: "},
		{"denormal", -149, "2091", "DENORMAL: "},
		{"overflow", 2000, "", ""},
	} {
		var x ArbitraryFloat
		sig := NewInt(1 << 26)
		serviceMust(t, m.SetComponents(&x, 8, 24, false, tc.exponent, &sig))
		output.Reset()
		serviceMust(t, m.Round(ArbitraryFloatNearestEven, &x))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("%s erased round trace: %q", tc.name, got)
			}
		} else if tc.line == "" {
			if strings.Contains(got, "round src/util/mpf.cpp:2087") || strings.Contains(got, "round src/util/mpf.cpp:2091") {
				t.Fatalf("overflow reached result class: %q", got)
			}
		} else {
			header := "-------- [mpf_dbg] round src/util/mpf.cpp:" + tc.line + " ---------\n" + tc.label
			if !strings.Contains(got, header) || strings.Count(got, header) != 1 {
				t.Fatalf("%s result class record: %q", tc.name, got)
			}
		}
		serviceMust(t, m.CleanupRound())
		serviceMust(t, m.Del(&x))
	}
}

func TestArbitraryFloatRoundEntryTraceRefusalBeforeNumericScope(t *testing.T) {
	m, _, budget := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	if !tags.ShouldTrace("mpf_dbg") {
		return
	}
	serviceMust(t, m.BindTrace(tags))
	var x ArbitraryFloat
	sig := NewInt(1 << 26)
	serviceMust(t, m.SetComponents(&x, 8, 24, false, 0, &sig))
	budget.FailAfter(0)
	e := m.Round(ArbitraryFloatNearestEven, &x)
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || m.RoundScope.active || m.RoundScope.Invocation != 0 {
		t.Fatalf("entry formatter refusal must precede numeric scope: %v, %#v", e, m.RoundScope)
	}
	if !strings.Contains(output.String(), "round src/util/mpf.cpp:1967 ---------\nRND: ") {
		t.Fatalf("entry trace prefix on formatter refusal: %q", output.String())
	}
	serviceMust(t, m.Round(ArbitraryFloatNearestEven, &x))
	serviceMust(t, m.CleanupRound())
	serviceMust(t, m.Del(&x))
}

func TestArbitraryFloatRoundSqrtDecisionTrace(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	w := &mpffArithmeticTraceWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	for _, tc := range []struct {
		tail int32
		mode ArbitraryFloatRoundingMode
		body string
		bits [4]string
	}{
		{4, ArbitraryFloatNearestEven, "last=0 round=1 sticky=0 --> inc=0", [4]string{"0", "1", "0", "0"}},
		{5, ArbitraryFloatNearestEven, "last=0 round=1 sticky=1 --> inc=1", [4]string{"0", "1", "1", "1"}},
		{4, ArbitraryFloatTowardPositive, "last=0 round=1 sticky=0 --> inc=1", [4]string{"0", "1", "0", "1"}},
		{8, ArbitraryFloatNearestEven, "last=1 round=0 sticky=0 --> inc=0", [4]string{"1", "0", "0", "0"}},
	} {
		var x ArbitraryFloat
		sig := NewInt((1 << 26) + tc.tail)
		serviceMust(t, m.SetComponents(&x, 8, 24, false, 0, &sig))
		w.parts = nil
		serviceMust(t, m.RoundSqrt(tc.mode, &x))
		got := strings.Join(w.parts, "")
		if !compiled {
			if got != "" {
				t.Fatalf("erased sqrt round trace: %q", got)
			}
		} else {
			entry := "-------- [mpf_dbg] round_sqrt src/util/mpf.cpp:2101 ---------\nRND-SQRT: "
			header := "-------- [mpf_dbg] round_sqrt src/util/mpf.cpp:2123 ---------\n" + tc.body + "\n"
			result := "-------- [mpf_dbg] round_sqrt src/util/mpf.cpp:2130 ---------\nROUNDED = "
			if first, decision, last := strings.Index(got, entry), strings.Index(got, header), strings.Index(got, result); first < 0 || decision <= first || last <= decision || strings.Count(got, "------------------------------------------------\n") != 3 {
				t.Fatalf("sqrt round source decision: %q", got)
			}
			want := []string{"last=", tc.bits[0], " round=", tc.bits[1], " sticky=", tc.bits[2], " --> inc=", tc.bits[3], "\n"}
			if !hasMPFFTraceParts(w.parts, want) {
				t.Fatalf("sqrt round source insertions %q missing in %q", want, w.parts)
			}
		}
		serviceMust(t, m.CleanupRoundSqrt())
		serviceMust(t, m.Del(&x))
	}
}
