package numeral

import (
	"bytes"
	"errors"
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
