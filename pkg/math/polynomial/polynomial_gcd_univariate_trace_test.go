package polynomial

// Source: src/math/polynomial/polynomial.cpp:3770-3860; src/util/trace.h:40-46.

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialUnivariateModularGCDSourceTrace(t *testing.T) {
	for _, shared := range []bool{false, true} {
		t.Run(map[bool]string{false: "coprime", true: "shared"}[shared], func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			u := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, 1}}, nil})
			v := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}}, nil})
			if shared {
				factor := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
				p, err := c.Mul(factor, u)
				serviceMust(t, err)
				polynomialCaller(t, c, p)
				q, err := c.Mul(factor, v)
				serviceMust(t, err)
				polynomialCaller(t, c, q)
				u, v = p, q
			}
			r, err := NewPolynomialRef(c)
			serviceMust(t, err)
			t.Cleanup(func() { serviceMust(t, full.CleanupGCD()); serviceMust(t, r.Close()) })
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("mgcd")
			c.monomials.config.Trace = tags
			serviceMust(t, full.GCD(u, v, r))
			want := map[[3]uint32]int64{{}: 1}
			if shared {
				want = map[[3]uint32]int64{{1, 0, 0}: 1, {}: 1}
			}
			polynomialQueryExpect(t, c, r.root.value, want)
			if tags.ShouldTrace("mgcd") {
				got := output.String()
				record := func(line int, body string) string {
					return "-------- [mgcd] uni_mod_gcd src/math/polynomial/polynomial.cpp:" + strconv.Itoa(line) +
						" ---------\n" + body + "\n------------------------------------------------\n"
				}
				entry := "univ_modular_gcd\nu: x0 + 2\nv: x0 + 3"
				if shared {
					entry = "univ_modular_gcd\nu: x0^2 + 3 x0 + 2\nv: x0^2 + 4 x0 + 3"
				}
				if !strings.Contains(got, record(3770, entry)) {
					t.Fatalf("missing source entry record: %q", got)
				}
				if !strings.Contains(got, record(3805, "trying prime: "+strconv.FormatUint(uint64(upolyBigPrimes[0]), 10))) {
					t.Fatalf("missing first source prime record: %q", got)
				}
				if shared {
					for _, item := range []struct {
						line int
						body string
					}{
						{3825, "new q:\nx0 + 1"}, {3851, "candidate:\nx0 + 1"},
						{3857, "found GCD"}, {3860, "r: x0 + 1"},
					} {
						if !strings.Contains(got, record(item.line, item.body)) {
							t.Fatalf("missing source record %d: %q", item.line, got)
						}
					}
				} else {
					for _, item := range []struct {
						line int
						body string
					}{
						{3825, "new q:\n1"}, {3827, "done, modular gcd is one"},
					} {
						if !strings.Contains(got, record(item.line, item.body)) {
							t.Fatalf("missing source record %d: %q", item.line, got)
						}
					}
				}
			} else if output.Len() != 0 {
				t.Fatalf("ordinary build emitted trace: %q", output.String())
			}
			tags.Disable("mgcd")
			before := output.String()
			serviceMust(t, full.GCD(u, v, r))
			if output.String() != before {
				t.Fatalf("disabled trace emitted: %q", output.String())
			}
			if shared {
				refused := &algebraicIntervalRefusingWriter{stop: 10, err: errors.New("trace refused")}
				tags = diagnostic.NewTraceTagsWithWriter(refused)
				tags.Enable("mgcd")
				c.monomials.config.Trace = tags
				serviceMust(t, full.GCD(u, v, r))
				polynomialQueryExpect(t, c, r.root.value, want)
				if tags.ShouldTrace("mgcd") && refused.writes != 10 {
					t.Fatalf("refused source stream writes: %d", refused.writes)
				}
			}
		})
	}
}
