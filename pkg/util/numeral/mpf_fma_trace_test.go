package numeral

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryFloatFMAEntryDebugTraceBeforeShortcuts(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var zero, one, out ArbitraryFloat
	serviceMust(t, m.MkZero(11, 53, false, &zero))
	serviceMust(t, m.MkOne(11, 53, false, &one))
	for _, x := range []*ArbitraryFloat{&zero, &one} {
		output.Reset()
		serviceMust(t, m.FMA(ArbitraryFloatNearestEven, x, &one, &one, &out))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("erased FMA trace: %q", got)
			}
			continue
		}
		last := -1
		for _, site := range []struct{ line, label string }{{"752", "X = "}, {"753", "Y = "}, {"754", "Z = "}} {
			entry := "-------- [mpf_dbg] fma src/util/mpf.cpp:" + site.line + " ---------\n" + site.label
			at := strings.Index(got, entry)
			if at <= last {
				t.Fatalf("FMA entry order at %s: %q", site.line, got)
			}
			last = at
		}
		if x == &zero && strings.Count(got, "-------- [mpf_dbg] fma src/util/mpf.cpp:") != 3 {
			t.Fatalf("shortcut reached FMA inner trace: %q", got)
		}
	}
	serviceMust(t, m.Del(&zero))
	serviceMust(t, m.Del(&one))
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatFMASwapAndArithmeticBranchTrace(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var one, z, out ArbitraryFloat
	serviceMust(t, m.MkOne(11, 53, false, &one))
	for _, tc := range []struct {
		name           string
		addend         float64
		swap, subtract bool
	}{
		{"large positive addend", 100, true, false},
		{"small negative addend", -0.5, false, true},
	} {
		serviceMust(t, m.SetDouble(&z, 11, 53, tc.addend))
		output.Reset()
		serviceMust(t, m.FMA(ArbitraryFloatNearestEven, &one, &one, &z, &out))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("%s erased trace: %q", tc.name, got)
			}
			continue
		}
		prefix := "-------- [mpf_dbg] fma src/util/mpf.cpp:"
		swapAt := strings.Index(got, prefix+"852 ---------\nSwap!\n")
		if (swapAt >= 0) != tc.swap {
			t.Fatalf("%s swap trace: %q", tc.name, got)
		}
		branch, opposite := "892 ---------\nADDING\n", "880 ---------\nSUBTRACTING\n"
		if tc.subtract {
			branch, opposite = opposite, branch
		}
		at := strings.Index(got, prefix+branch)
		if at < 0 || strings.Contains(got, prefix+opposite) || tc.swap && at <= swapAt {
			t.Fatalf("%s selected FMA branch: %q", tc.name, got)
		}
	}
	serviceMust(t, m.Del(&one))
	serviceMust(t, m.Del(&z))
	serviceMust(t, m.Del(&out))
}
