package numeral

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryFloatSqrtTraceSourceBranches(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var x, out ArbitraryFloat
	for _, tc := range []struct {
		name             string
		value            float64
		general, inexact bool
	}{
		{"zero shortcut", 0, false, false},
		{"perfect square", 4, true, false},
		{"inexact square root", 2, true, true},
	} {
		serviceMust(t, m.SetDouble(&x, 11, 53, tc.value))
		output.Reset()
		serviceMust(t, m.Sqrt(ArbitraryFloatNearestEven, &x, &out))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("%s erased sqrt trace: %q", tc.name, got)
			}
			continue
		}
		entry := strings.Index(got, "sqrt src/util/mpf.cpp:1008 ---------\nX = ")
		unpacked := strings.Index(got, "sqrt src/util/mpf.cpp:1027 ---------\nA = ")
		adjusted := strings.Index(got, "sqrt src/util/mpf.cpp:1036 ---------\ndec'ed ")
		result := strings.Index(got, "sqrt src/util/mpf.cpp:1044 ---------\nSQRT = ")
		if entry < 0 || result <= entry || (unpacked >= 0) != tc.general || (adjusted >= 0) != tc.inexact {
			t.Fatalf("%s sqrt branch records: %q", tc.name, got)
		}
		if tc.general && (unpacked <= entry || result <= unpacked) || tc.inexact && (adjusted <= unpacked || adjusted >= result) {
			t.Fatalf("%s sqrt source order: %q", tc.name, got)
		}
		if tc.general && !strings.Contains(got, "round src/util/mpf.cpp:2000") || !tc.general && strings.Contains(got, "round src/util/mpf.cpp:2000") {
			t.Fatalf("%s reached nested Round: %q", tc.name, got)
		}
	}
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatSqrtTraceSourceInsertions(t *testing.T) {
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
		name  string
		value float64
		want  []string
	}{
		{"zero shortcut", 0, []string{"X = ", "SQRT = "}},
		{"perfect square", 4, []string{"X = ", "A = ", "SQRT = "}},
		{"inexact square root", 2, []string{"X = ", "A = ", "dec'ed ", "SQRT = "}},
	} {
		serviceMust(t, m.SetDouble(&x, 11, 53, tc.value))
		w.parts = nil
		serviceMust(t, m.Sqrt(ArbitraryFloatNearestEven, &x, &out))
		last := -1
		for _, label := range tc.want {
			found := -1
			for i := last + 1; i+2 < len(w.parts); i++ {
				if w.parts[i] == label && w.parts[i+1] != "" && w.parts[i+1] != "\n" && w.parts[i+2] == "\n" {
					found = i
					break
				}
			}
			if found < 0 {
				t.Fatalf("%s missing source insertion boundary after %d for %q in %q", tc.name, last, label, w.parts)
			}
			last = found
		}
	}
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.Del(&out))
}
