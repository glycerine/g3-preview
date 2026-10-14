package numeral

import (
	"bytes"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryFloatSBVTraceFractionalAndExactBranches(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	tags.Enable("mpf_dbg_sbv")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var x ArbitraryFloat
	var out Rational
	for _, tc := range []struct {
		value float64
		want  string
		inner bool
		sign  string
	}{
		{1.5, "2", true, "0"},
		{-1.5, "-2", true, "1"},
		{float64(uint64(1) << 52), "4503599627370496", false, "0"},
	} {
		serviceMust(t, m.SetDouble(&x, 11, 53, tc.value))
		output.Reset()
		serviceMust(t, m.ToSBVMPQ(ArbitraryFloatNearestEven, &x, &out))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("erased SBV trace for %v: %q", tc.value, got)
			}
			continue
		}
		inner := strings.Index(got, "to_sbv_mpq src/util/mpf.cpp:1214 ---------\nSBV: (")
		final := strings.Index(got, "to_sbv_mpq src/util/mpf.cpp:1225 ---------\nSBV = "+tc.want+"\n")
		unpack := strings.Index(got, "unpack src/util/mpf.cpp:1923 ---------\nunpack ")
		blocks := 2
		if tc.inner {
			blocks = 3
		}
		if unpack < 0 || final <= unpack || (inner >= 0) != tc.inner || tc.inner && (inner <= unpack || inner >= final || !strings.Contains(got, "\nsign="+tc.sign+" last=")) || strings.Count(got, "------------------------------------------------\n") != blocks {
			t.Fatalf("SBV trace branches for %v: %q", tc.value, got)
		}
	}
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.rational.Del(&out))
}

func TestArbitraryFloatIEEEBVTraceFiniteAndInfinity(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var x ArbitraryFloat
	out := NewInt(0)
	for _, value := range []float64{1.5, math.Inf(1)} {
		serviceMust(t, m.SetDouble(&x, 11, 53, value))
		output.Reset()
		serviceMust(t, m.ToIEEEBVMPZ(&x, &out))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("erased IEEE BV trace for %v: %q", value, got)
			}
			continue
		}
		want := strconv.FormatUint(math.Float64bits(value), 10)
		if !strings.Contains(got, "to_ieee_bv_mpz src/util/mpf.cpp:1252 ---------\nIEEE_BV = "+want+"\n") || strings.Count(got, "------------------------------------------------\n") != 1 {
			t.Fatalf("IEEE BV source record for %v: %q", value, got)
		}
	}
	serviceMust(t, m.integer.Del(&out))
	serviceMust(t, m.Del(&x))
}
