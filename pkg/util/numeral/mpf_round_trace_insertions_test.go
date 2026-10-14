package numeral

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type mpfRoundRefusalWriter struct {
	parts  []string
	refuse string
}

func (w *mpfRoundRefusalWriter) Write(p []byte) (int, error) {
	part := string(p)
	if part == w.refuse {
		return 0, errors.New("round trace output refused")
	}
	w.parts = append(w.parts, part)
	return len(p), nil
}

func TestArbitraryFloatRoundSourceInsertionBoundaries(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	for _, tc := range []struct {
		name, direction, branch, beta, below, sigma, last string
		exponent                                          int64
	}{
		{"normal", "(LEFT)", "NORMAL: ", "0", "0", "0", "0", 0},
		{"denormal", "(RIGHT)", "DENORMAL: ", "-149", "1", "-23", "1", -149},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var x ArbitraryFloat
			sig := NewInt(1 << 26)
			serviceMust(t, m.SetComponents(&x, 8, 24, false, tc.exponent, &sig))
			w := &mpffArithmeticTraceWriter{}
			tags := diagnostic.NewTraceTagsWithWriter(w)
			tags.Enable("mpf_dbg")
			serviceMust(t, m.BindTrace(tags))
			serviceMust(t, m.Round(ArbitraryFloatNearestEven, &x))
			if tags.ShouldTrace("mpf_dbg") {
				for _, want := range [][]string{
					{"e_min_norm = ", "-126", "\n", "e_max_norm = ", "127", "\n"},
					{"beta = ", tc.beta, ", (beta < e_min) = ", tc.below, "\n", "LZ = ", "1", "\n", "sigma = ", tc.sigma, "\n", "sigma_cap = ", "-26", "\n"},
					{"Shift distance: ", tc.sigma, " ", tc.direction, "\n"},
					{"sign=", "0", " last=", tc.last, " round=", "0", " sticky=", "0", "\n"},
					{"Rounding increment -> significand +", "0", "\n"},
					{"SIGovf = ", "0", "\n"},
					{"OVF2 = ", "0", "\n"},
					{"o_has_max_exp = ", "0", "\n"},
				} {
					if !hasMPFFTraceParts(w.parts, want) {
						t.Fatalf("missing source insertions %q in %q", want, w.parts)
					}
				}
				branch := false
				for i := 0; i+2 < len(w.parts); i++ {
					if w.parts[i] == tc.branch && w.parts[i+1] != "" && w.parts[i+2] == "\n" {
						branch = true
						break
					}
				}
				if !branch {
					t.Fatalf("result-class source insertions %q in %q", tc.branch, w.parts)
				}
				for _, record := range []struct {
					label, suffix string
					binary        bool
				}{
					{"RND: ", "", true},
					{"Shifted: ", "", true},
					{"before rounding decision: ", "", false},
					{"Rounded significand: ", "", false},
					{"Post-normalized: ", "", false},
					{"ROUNDED = ", " (hidden bit, unbiased exp).", true},
				} {
					found := false
					for i := 0; i+2 < len(w.parts); i++ {
						if w.parts[i] != record.label || w.parts[i+1] == "" || w.parts[i+2] != "\n" {
							continue
						}
						if record.binary {
							if i+4 >= len(w.parts) || !strings.HasPrefix(w.parts[i+3], "#b") {
								continue
							}
							if record.suffix == "" && w.parts[i+4] != "\n" {
								continue
							}
							if record.suffix != "" && (i+5 >= len(w.parts) || w.parts[i+4] != record.suffix || w.parts[i+5] != "\n") {
								continue
							}
						}
						found = true
						break
					}
					if !found {
						t.Fatalf("value source insertions %q in %q", record.label, w.parts)
					}
				}
			}
			serviceMust(t, m.CleanupRound())
			serviceMust(t, m.Del(&x))
		})
	}
}

func TestArbitraryFloatRoundNormalizationRefusedInsertionPrefix(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var x ArbitraryFloat
	sig := NewInt(1 << 26)
	serviceMust(t, m.SetComponents(&x, 8, 24, false, 0, &sig))
	w := &mpfRoundRefusalWriter{refuse: "sigma_cap = "}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	serviceMust(t, m.Round(ArbitraryFloatNearestEven, &x))
	if tags.ShouldTrace("mpf_dbg") {
		if !hasMPFFTraceParts(w.parts, []string{"LZ = ", "1", "\n", "sigma = ", "0", "\n"}) {
			t.Fatalf("source prefix before refused insertion: %q", w.parts)
		}
		if strings.Contains(strings.Join(w.parts, ""), "sigma_cap = ") || strings.Contains(strings.Join(w.parts, ""), "Shift distance:") {
			t.Fatalf("failed stream wrote after refused insertion: %q", w.parts)
		}
	}
	serviceMust(t, m.CleanupRound())
	serviceMust(t, m.Del(&x))
}

func TestArbitraryFloatRoundOverflowFlagInsertions(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var x ArbitraryFloat
	sig := NewInt(1 << 26)
	serviceMust(t, m.SetComponents(&x, 8, 24, false, 2000, &sig))
	w := &mpffArithmeticTraceWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	serviceMust(t, m.Round(ArbitraryFloatNearestEven, &x))
	if tags.ShouldTrace("mpf_dbg") {
		for _, want := range [][]string{
			{"SIGovf = ", "1", "\n"},
			{"OVF2 = ", "1", "\n"},
			{"o_has_max_exp = ", "1", "\n"},
		} {
			if !hasMPFFTraceParts(w.parts, want) {
				t.Fatalf("overflow source insertions %q in %q", want, w.parts)
			}
		}
	}
	serviceMust(t, m.CleanupRound())
	serviceMust(t, m.Del(&x))
}
