package numeral

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryFloatAddSubDebugTraceSourceBranches(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var one, two, out ArbitraryFloat
	serviceMust(t, m.MkOne(11, 53, false, &one))
	serviceMust(t, m.SetDouble(&two, 11, 53, 2))
	for _, tc := range []struct {
		name          string
		sub           bool
		branch, other string
	}{
		{"addition", false, "565", "559"},
		{"subtraction", true, "559", "565"},
	} {
		output.Reset()
		if tc.sub {
			serviceMust(t, m.Sub(ArbitraryFloatNearestEven, &two, &one, &out))
		} else {
			serviceMust(t, m.Add(ArbitraryFloatNearestEven, &two, &one, &out))
		}
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("%s erased trace: %q", tc.name, got)
			}
			continue
		}
		last := -1
		for _, site := range []struct{ line, label string }{
			{"541", "A  = "}, {"542", "B  = "}, {"543", "d  = "},
			{"554", "A' = "}, {"555", "B' = "}, {tc.branch, ""},
			{"571", "sum[-2:sbits+2] = "}, {"577", "NEG="}, {"579", "fs[-1:sbits+2] = "},
		} {
			entry := "-------- [mpf_dbg] add_sub src/util/mpf.cpp:" + site.line + " ---------\n" + site.label
			at := strings.Index(got, entry)
			if at <= last {
				t.Fatalf("%s debug source order at %s: %q", tc.name, site.line, got)
			}
			last = at
		}
		if strings.Contains(got, "-------- [mpf_dbg] add_sub src/util/mpf.cpp:"+tc.other+" ---------") {
			t.Fatalf("%s emitted opposite arithmetic branch: %q", tc.name, got)
		}
	}
	output.Reset()
	serviceMust(t, m.Sub(ArbitraryFloatNearestEven, &one, &one, &out))
	if compiled {
		got := output.String()
		if !strings.Contains(got, "sum[-2:sbits+2] = ") || strings.Contains(got, "NEG=") || strings.Contains(got, "fs[-1:sbits+2] = ") {
			t.Fatalf("zero result reached normalization trace: %q", got)
		}
	}
	serviceMust(t, m.Del(&one))
	serviceMust(t, m.Del(&two))
	serviceMust(t, m.Del(&out))
}
