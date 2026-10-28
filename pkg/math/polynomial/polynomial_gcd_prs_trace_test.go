package polynomial

// Source: src/math/polynomial/polynomial.cpp:3556-3604; src/util/trace.h:40-46.

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func polynomialGCDFocusedTraceRecords(output string) []string {
	var result []string
	for _, part := range strings.Split(output, "------------------------------------------------\n") {
		if strings.Contains(part, " gcd_prs src/math/polynomial/polynomial.cpp:") {
			result = append(result, part+"------------------------------------------------\n")
		}
	}
	return result
}

func TestPolynomialGCDPRSSourceTraceBranches(t *testing.T) {
	for route := uint32(0); route < 2; route++ {
		t.Run([]string{"shared", "constant"}[route], func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			full.usePRSGCD = true
			m := full.core
			u := polynomialQueryTerms(t, m, []int32{6, 6, -12}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
			vCoeffs := []int32{10, 20, -30}
			vPowers := [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil}
			if route == 1 {
				vCoeffs = []int32{10, 10}
				vPowers = [][]MonomialPower{{{0, 2}}, nil}
			}
			v := polynomialQueryTerms(t, m, vCoeffs, vPowers)
			wantPowers := [][]MonomialPower{{{0, 1}}, nil}
			wantCoeffs := []int32{2, -2}
			if route == 1 {
				wantPowers = [][]MonomialPower{nil}
				wantCoeffs = []int32{2}
			}
			wantResult := polynomialQueryTerms(t, m, wantCoeffs, wantPowers)
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			for _, tag := range [...]string{"polynomial_gcd", "polynomial_gcd_detail", "polynomial"} {
				tags.Enable(tag)
			}
			m.monomials.config.Trace = tags
			r, err := NewPolynomialRef(m)
			serviceMust(t, err)
			t.Cleanup(func() { serviceMust(t, r.Close()) })
			serviceMust(t, full.GCD(u, v, r))
			same, err := m.Equal(r.root.value, wantResult)
			serviceMust(t, err)
			if !same {
				t.Fatal("trace changed PRS result")
			}
			records := polynomialGCDFocusedTraceRecords(output.String())
			if tags.ShouldTrace("polynomial_gcd") {
				record := func(tag string, line int, body string) string {
					return "-------- [" + tag + "] gcd_prs src/math/polynomial/polynomial.cpp:" + strconv.Itoa(line) +
						" ---------\n" + body + "\n------------------------------------------------\n"
				}
				vText, pvText, nextText := "10 x0^2 + 20 x0 - 30", "x0^2 + 2 x0 - 3", "- x0 + 1"
				exitRecord := record("polynomial", 3597, "rem is zero...\npp_v: - x0 + 1")
				if route == 1 {
					vText, pvText, nextText = "10 x0^2 + 10", "x0^2 + 1", "x0 - 3"
					exitRecord = record("polynomial", 3604, "rem is a constant: 10\nr: 1\nd_a: 2")
				}
				want := []string{
					record("polynomial_gcd", 3556, "gcd prs with x0 for\nu: 6 x0^2 + 6 x0 - 12\nv: "+vText),
					record("polynomial_gcd_detail", 3572, "After GCD of the content\nu: 6 x0^2 + 6 x0 - 12\nv: "+vText+
						"\ni_u: 6\ni_v: 10\nc_u: 1\nc_v: 1\npp_u: x0^2 + x0 - 2\npp_v: "+pvText+"\nd_r: 1\nd_a: 2"),
					record("polynomial_gcd_detail", 3591, "iteration: 0\ngcd loop\npp_u: x0^2 + x0 - 2\npp_v: "+pvText+"\ndelta: 0"),
					record("polynomial_gcd_detail", 3591, "iteration: 1\ngcd loop\npp_u: "+pvText+"\npp_v: "+nextText+"\ndelta: 1"),
					exitRecord,
				}
				if len(records) != len(want) {
					t.Fatalf("PRS trace count: got %#v, want %#v", records, want)
				}
				for i := range want {
					if records[i] != want[i] {
						t.Fatalf("PRS record %d: got %q, want %q", i, records[i], want[i])
					}
				}
			} else if len(records) != 0 {
				t.Fatalf("ordinary build emitted PRS traces: %#v", records)
			}
			for _, tag := range [...]string{"polynomial_gcd", "polynomial_gcd_detail", "polynomial"} {
				tags.Disable(tag)
			}
			before := output.String()
			serviceMust(t, full.GCD(u, v, r))
			if output.String() != before {
				t.Fatalf("disabled PRS trace: %q", output.String())
			}

			refusal := errors.New("PRS trace refused")
			refused := &algebraicIntervalRefusingWriter{stop: 10, err: refusal}
			tags = diagnostic.NewTraceTagsWithWriter(refused)
			tags.Enable("polynomial_gcd")
			m.monomials.config.Trace = tags
			serviceMust(t, full.GCD(u, v, r))
			if tags.ShouldTrace("polynomial_gcd") {
				if refused.writes != 10 || !strings.HasSuffix(refused.String(), "---------\n") {
					t.Fatalf("refused PRS entry prefix: writes=%d output=%q", refused.writes, refused.String())
				}
			} else if refused.writes != 0 {
				t.Fatalf("ordinary build emitted refused PRS trace: %q", refused.String())
			}
			tags.Disable("polynomial_gcd")
		})
	}
}
