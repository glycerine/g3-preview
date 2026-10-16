// Source: src/math/polynomial/polynomial.cpp:3678-3760; src/util/trace.h:40-46.
package polynomial

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func polynomialCRATraceRecords(output string) []string {
	var result []string
	for _, part := range strings.Split(output, "------------------------------------------------\n") {
		if strings.Contains(part, " CRA_combine_images src/math/polynomial/polynomial.cpp:") {
			result = append(result, part+"------------------------------------------------\n")
		}
	}
	return result
}

func TestPolynomialCRASourceTraceMergeBranches(t *testing.T) {
	for route := uint32(0); route < 5; route++ {
		t.Run([]string{"equal", "opposite", "different", "C1-tail", "C2-tail"}[route], func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			m := full.core
			aCoeffs, aPowers := []int32{-1, 1}, [][]MonomialPower{{{0, 1}}, nil}
			bCoeffs, bPowers := []int32{2, 1}, [][]MonomialPower{{{0, 1}}, nil}
			if route == 1 {
				aCoeffs = []int32{1, 1}
				bCoeffs = []int32{-1, -2}
			}
			if route == 2 {
				aCoeffs, aPowers = []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil}
				bCoeffs, bPowers = []int32{1, 2}, [][]MonomialPower{{{1, 1}}, nil}
			}
			if route == 3 {
				aCoeffs, aPowers = []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil}
				bCoeffs, bPowers = []int32{1}, [][]MonomialPower{{{1, 1}}}
			}
			if route == 4 {
				aCoeffs, aPowers = []int32{1}, [][]MonomialPower{{{0, 2}}}
				bCoeffs, bPowers = []int32{1, 2}, [][]MonomialPower{{{1, 1}}, nil}
			}
			a := polynomialQueryTerms(t, m, aCoeffs, aPowers)
			b := polynomialQueryTerms(t, m, bCoeffs, bPowers)
			r, err := NewPolynomialRef(m)
			serviceMust(t, err)
			t.Cleanup(func() { serviceMust(t, r.Close()) })
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("CRA")
			m.monomials.config.Trace = tags
			first, second := unumeral.NewInt(3), unumeral.NewInt(5)
			serviceMust(t, full.gcdCRA(a, &first, b, &second, r))
			if second != unumeral.NewInt(15) {
				t.Fatalf("CRA bound changed: %v", second)
			}
			wants := []map[[3]uint32]int64{
				{{1, 0, 0}: 2, {}: 1},
				{{1, 0, 0}: 4, {}: -2},
				{{2, 0, 0}: -5, {0, 1, 0}: 6, {}: 7},
				{{2, 0, 0}: -5, {0, 1, 0}: 6, {}: -5},
				{{2, 0, 0}: -5, {0, 1, 0}: 6, {}: -3},
			}
			polynomialQueryExpect(t, m, r.root.value, wants[route])
			records := polynomialCRATraceRecords(output.String())
			if tags.ShouldTrace("CRA") {
				record := func(line int, body string) string {
					return "-------- [CRA] CRA_combine_images src/math/polynomial/polynomial.cpp:" + strconv.Itoa(line) +
						" ---------\n" + body + "\n------------------------------------------------\n"
				}
				images := []string{
					"C1: - x0 + 1\nC2: 2 x0 + 1",
					"C1: x0 + 1\nC2: - x0 - 2",
					"C1: x0^2 + 1\nC2: x1 + 2",
					"C1: x0^2 + 1\nC2: x1",
					"C1: x0^2\nC2: x1 + 2",
				}
				want := []string{
					record(3678, images[route]),
					record(3688, "b1: 3, b2: 5, inv1: 2, inv2: -1"),
					record(3694, "inv1: 2, inv2: 2"),
					record(3699, "a1: 10, a2: 6"),
					record(3709, "lower: -7, upper: 7"),
				}
				switch route {
				case 0:
					want = append(want,
						record(3748, "C1->a(i1): -1, C2->a(i2): 2, new_a: 2\ntmp1: -10, tmp2: 12, tmp3: 2"),
						record(3748, "C1->a(i1): 1, C2->a(i2): 1, new_a: 1\ntmp1: 10, tmp2: 6, tmp3: 16"))
				case 1:
					want = append(want,
						record(3748, "C1->a(i1): 1, C2->a(i2): -1, new_a: 4\ntmp1: 10, tmp2: -6, tmp3: 4"),
						record(3748, "C1->a(i1): 1, C2->a(i2): -2, new_a: -2\ntmp1: 10, tmp2: -12, tmp3: -2"))
				case 2:
					want = append(want, record(3760, "C2 mon bigger, adding it..."),
						record(3755, "C1 mon biggerr, adding it..."),
						record(3748, "C1->a(i1): 1, C2->a(i2): 2, new_a: 7\ntmp1: 10, tmp2: 12, tmp3: 22"))
				case 3:
					want = append(want, record(3760, "C2 mon bigger, adding it..."),
						record(3737, "adding C1 rest"), record(3737, "adding C1 rest"))
				case 4:
					want = append(want, record(3760, "C2 mon bigger, adding it..."),
						record(3755, "C1 mon biggerr, adding it..."), record(3729, "adding C2 rest"))
				}
				if len(records) != len(want) {
					t.Fatalf("CRA trace count: got %#v, want %#v", records, want)
				}
				for i := range want {
					if records[i] != want[i] {
						t.Fatalf("CRA record %d: got %q, want %q", i, records[i], want[i])
					}
				}
			} else if len(records) != 0 {
				t.Fatalf("ordinary build emitted CRA trace: %#v", records)
			}
			tags.Disable("CRA")
			before := output.String()
			first, second = unumeral.NewInt(3), unumeral.NewInt(5)
			serviceMust(t, full.gcdCRA(a, &first, b, &second, r))
			if output.String() != before {
				t.Fatalf("disabled CRA trace: %q", output.String())
			}
			if route == 0 {
				refusal := errors.New("CRA trace refused")
				refused := &algebraicIntervalRefusingWriter{stop: 10, err: refusal}
				tags = diagnostic.NewTraceTagsWithWriter(refused)
				tags.Enable("CRA")
				m.monomials.config.Trace = tags
				first, second = unumeral.NewInt(3), unumeral.NewInt(5)
				serviceMust(t, full.gcdCRA(a, &first, b, &second, r))
				polynomialQueryExpect(t, m, r.root.value, wants[route])
				if tags.ShouldTrace("CRA") {
					if refused.writes != 10 || !strings.HasSuffix(refused.String(), "---------\n") {
						t.Fatalf("refused CRA entry prefix: writes=%d output=%q", refused.writes, refused.String())
					}
				} else if refused.writes != 0 {
					t.Fatalf("ordinary build emitted CRA trace: %q", refused.String())
				}
				tags.Disable("CRA")
			}
		})
	}
}
