package numeral

import (
	"bytes"
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
