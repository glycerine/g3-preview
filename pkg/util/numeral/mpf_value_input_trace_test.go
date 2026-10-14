package numeral

import (
	"bytes"
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryFloatSetInt32TraceEntryAndResult(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var out ArbitraryFloat
	for _, value := range []int32{0, -7, 31} {
		output.Reset()
		serviceMust(t, m.SetInt32(&out, 11, 53, value))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("erased integer setter trace for %d: %q", value, got)
			}
			continue
		}
		entry := "-------- [mpf_dbg] set src/util/mpf.cpp:76 ---------\nset: value = "
		result := "-------- [mpf_dbg] set src/util/mpf.cpp:105 ---------\nset: res = "
		first, second := strings.Index(got, entry), strings.Index(got, result)
		if first < 0 || second <= first || strings.Count(got, "-------- [mpf_dbg] set src/util/mpf.cpp:") != 2 || strings.Count(got, "------------------------------------------------\n") != 2 {
			t.Fatalf("integer setter source order for %d: %q", value, got)
		}
		if !strings.Contains(got[first:], "set: value = "+strconv.FormatInt(int64(value), 10)+"\n") || !strings.Contains(got[second:], "set: res = ") {
			t.Fatalf("integer setter values for %d: %q", value, got)
		}
	}
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatSetNativeTraceFormats(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var out ArbitraryFloat
	for _, tc := range []struct {
		name                  string
		inputLine, resultLine string
		call                  func() error
		input, result         string
	}{
		{"double", "124", "147", func() error { return m.SetDouble(&out, 11, 53, -1.5) }, "set: -1.5 is: raw=" + strconv.FormatUint(math.Float64bits(-1.5), 10) + " (double) sign=1 s=2251799813685248 e=0", "set: res = -1.5 0"},
		{"float", "160", "183", func() error { return m.SetFloat(&out, 8, 24, 1.25) }, "set: 1.25 is: raw=" + strconv.FormatUint(uint64(math.Float32bits(1.25)), 10) + " (float) sign=0 s=2097152 e=0", "set: res = ["},
	} {
		output.Reset()
		serviceMust(t, tc.call())
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("%s erased native setter trace: %q", tc.name, got)
			}
			continue
		}
		entry := "-------- [mpf_dbg] set src/util/mpf.cpp:" + tc.inputLine + " ---------\n" + tc.input
		result := "-------- [mpf_dbg] set src/util/mpf.cpp:" + tc.resultLine + " ---------\n" + tc.result
		first, second := strings.Index(got, entry), strings.Index(got, result)
		if first < 0 || second <= first || strings.Count(got, "-------- [mpf_dbg] set src/util/mpf.cpp:") != 2 || strings.Count(got, "------------------------------------------------\n") != 2 {
			t.Fatalf("%s native setter trace: %q", tc.name, got)
		}
	}
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatSetInt32TraceFormatterRefusal(t *testing.T) {
	m, _, budget := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	var out ArbitraryFloat
	serviceMust(t, m.SetInt32(&out, 11, 53, 0))
	serviceMust(t, m.BindTrace(tags))
	if !tags.ShouldTrace("mpf_dbg") {
		serviceMust(t, m.Del(&out))
		return
	}
	budget.FailAfter(0)
	e := m.traceSetInt32Result(&out)
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || !strings.Contains(output.String(), "src/util/mpf.cpp:105 ---------\nset: res = ") {
		t.Fatalf("integer setter trace formatter refusal: %v, %q", e, output.String())
	}
	output.Reset()
	serviceMust(t, m.SetInt32(&out, 11, 53, 7))
	if !strings.Contains(output.String(), "src/util/mpf.cpp:105 ---------\nset: res = ") {
		t.Fatalf("integer setter trace retry: %q", output.String())
	}
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatSetFloatRawTraceFormatterRefusal(t *testing.T) {
	m, _, budget := mpfNumeralHelpersFixture(t)
	var out ArbitraryFloat
	serviceMust(t, m.SetFloat(&out, 8, 24, 1.25))
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	if !tags.ShouldTrace("mpf_dbg") {
		serviceMust(t, m.Del(&out))
		return
	}
	budget.FailAfter(0)
	e := m.traceSetFloatRawResult(&out)
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || m.rawString.active || !strings.Contains(output.String(), "src/util/mpf.cpp:183 ---------\nset: res = ") {
		t.Fatalf("float raw trace formatter refusal: %v, active %v, output %q", e, m.rawString.active, output.String())
	}
	output.Reset()
	serviceMust(t, m.SetFloat(&out, 8, 24, 2.5))
	if !strings.Contains(output.String(), "src/util/mpf.cpp:183 ---------\nset: res = [") {
		t.Fatalf("float raw trace retry: %q", output.String())
	}
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatSetPlainRationalTrace(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var q Rational
	var out ArbitraryFloat
	for _, tc := range []struct {
		n    int64
		text string
	}{{3, "3/4"}, {0, "0"}} {
		serviceMust(t, m.rational.SetFraction(&q, tc.n, 4))
		output.Reset()
		serviceMust(t, m.SetRational(&out, 8, 24, ArbitraryFloatNearestEven, &q))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("erased rational setter trace: %q", got)
			}
			continue
		}
		lines := []string{"187", "235", "270", "289", "300", "191"}
		if tc.n == 0 {
			lines = []string{"187", "235", "300", "191"}
		}
		last := -1
		for _, line := range lines {
			header := "-------- [mpf_dbg] set src/util/mpf.cpp:" + line + " ---------\n"
			at := strings.Index(got, header)
			if at <= last || strings.Count(got, header) != 1 {
				t.Fatalf("rational setter source order for %d at %s: %q", tc.n, line, got)
			}
			last = at
		}
		blocks := len(lines)
		if tc.n != 0 {
			blocks += 7
			firstRound := strings.Index(got, "round src/util/mpf.cpp:2000")
			lastRound := strings.Index(got, "round src/util/mpf.cpp:2079")
			if firstRound <= strings.Index(got, "set src/util/mpf.cpp:289") || lastRound <= firstRound || lastRound >= strings.Index(got, "set src/util/mpf.cpp:300") {
				t.Fatalf("nested rational Round trace for %d: %q", tc.n, got)
			}
		}
		if !strings.Contains(got, "src/util/mpf.cpp:187 ---------\nset: "+tc.text+" [8/24]\n") || strings.Count(got, "------------------------------------------------\n") != blocks {
			t.Fatalf("rational setter trace for %d: %q", tc.n, got)
		}
	}
	serviceMust(t, m.Del(&out))
	serviceMust(t, m.rational.Del(&q))
}

func TestArbitraryFloatSetFormattedLiteralTrace(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var out ArbitraryFloat
	for _, tc := range []struct {
		value, entry, parts string
	}{
		{"-1.25P2", "set: -1.25P2 [8/24]", "sgn = 1 f = 1.25 e = 2"},
		{"+0.5", "set: +0.5 [8/24]", "sgn = 0 f = 0.5 e = 0"},
		{"1.5\x00ignored", "set: 1.5 [8/24]", "sgn = 0 f = 1.5 e = 0"},
	} {
		output.Reset()
		serviceMust(t, m.SetFormattedLiteral(&out, 8, 24, ArbitraryFloatNearestEven, tc.value))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("erased literal setter trace: %q", got)
			}
			continue
		}
		last := -1
		for _, site := range []struct{ line, body string }{{"195", tc.entry}, {"219", tc.parts}, {"235", "set: sig = "}, {"270", "Normalized: sig = "}, {"289", "sig = "}, {"300", "set: res = "}, {"230", "set: res = "}} {
			header := "-------- [mpf_dbg] set src/util/mpf.cpp:" + site.line + " ---------\n" + site.body
			at := strings.Index(got, header)
			if at <= last || strings.Count(got, "src/util/mpf.cpp:"+site.line+" ---------") != 1 {
				t.Fatalf("literal setter trace order for %q at %s: %q", tc.value, site.line, got)
			}
			last = at
		}
		firstRound := strings.Index(got, "round src/util/mpf.cpp:2000")
		lastRound := strings.Index(got, "round src/util/mpf.cpp:2079")
		if firstRound <= strings.Index(got, "set src/util/mpf.cpp:289") || lastRound <= firstRound || lastRound >= strings.Index(got, "set src/util/mpf.cpp:300") || strings.Count(got, "------------------------------------------------\n") != 14 {
			t.Fatalf("literal setter trace blocks for %q: %q", tc.value, got)
		}
	}
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatSetExponentRationalTraceBranches(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var q Rational
	var out ArbitraryFloat
	exponent := NewInt(0)
	for _, tc := range []struct {
		numerator int64
		lines     []string
	}{
		{3, []string{"235", "270", "289", "300"}},
		{0, []string{"235", "300"}},
	} {
		serviceMust(t, m.rational.SetFraction(&q, tc.numerator, 4))
		out.exponent = 17
		output.Reset()
		serviceMust(t, m.SetExponentRational(&out, 8, 24, ArbitraryFloatNearestEven, &exponent, &q))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("erased exponent/rational trace: %q", got)
			}
			continue
		}
		last := -1
		for _, line := range tc.lines {
			header := "-------- [mpf_dbg] set src/util/mpf.cpp:" + line + " ---------\n"
			at := strings.Index(got, header)
			if at <= last || strings.Count(got, header) != 1 {
				t.Fatalf("exponent/rational source order for %d at %s: %q", tc.numerator, line, got)
			}
			last = at
		}
		blocks := len(tc.lines)
		if tc.numerator != 0 {
			blocks += 7
			firstRound := strings.Index(got, "round src/util/mpf.cpp:2000")
			lastRound := strings.Index(got, "round src/util/mpf.cpp:2079")
			if firstRound <= strings.Index(got, "set src/util/mpf.cpp:289") || lastRound <= firstRound || lastRound >= strings.Index(got, "set src/util/mpf.cpp:300") {
				t.Fatalf("nested exponent/rational Round trace: %q", got)
			}
		}
		if strings.Count(got, "------------------------------------------------\n") != blocks {
			t.Fatalf("exponent/rational branch for %d: %q", tc.numerator, got)
		}
		if tc.numerator == 3 && !strings.Contains(got, "src/util/mpf.cpp:289 ---------\nsig = ") || tc.numerator == 3 && !strings.Contains(got, " exp = 17\n") {
			t.Fatalf("pre-round field publication: %q", got)
		}
	}
	serviceMust(t, m.Del(&out))
	serviceMust(t, m.rational.Del(&q))
}
