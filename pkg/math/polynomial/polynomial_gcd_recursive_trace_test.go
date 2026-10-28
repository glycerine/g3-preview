package polynomial

// Source: src/math/polynomial/polynomial.cpp:4078-4223; src/util/trace.h:40-46.

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialRecursiveModularGCDSourceTrace(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	common := polynomialQueryTerms(t, c, []int32{1, 1, 1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}, nil})
	a := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, 1}}, nil})
	b := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}}, nil})
	u, err := c.Mul(common, a)
	serviceMust(t, err)
	polynomialCaller(t, c, u)
	v, err := c.Mul(common, b)
	serviceMust(t, err)
	polynomialCaller(t, c, v)
	r, err := NewPolynomialRef(c)
	serviceMust(t, err)
	t.Cleanup(func() { serviceMust(t, full.CleanupGCD()); serviceMust(t, r.Close()) })
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mgcd")
	tags.Enable("mgcd_detail")
	c.monomials.config.Trace = tags
	serviceMust(t, full.GCD(u, v, r))
	same, err := c.Equal(common, r.root.value)
	serviceMust(t, err)
	if !same {
		t.Fatal("recursive modular GCD changed under trace")
	}
	if tags.ShouldTrace("mgcd") {
		got := output.String()
		for _, part := range []string{
			"[mgcd] mod_gcd_rec src/math/polynomial/polynomial.cpp:4078",
			"[mgcd] mod_gcd_rec src/math/polynomial/polynomial.cpp:4141 ---------\nx",
			"[mgcd] mod_gcd_rec src/math/polynomial/polynomial.cpp:4085 ---------\nmod_gcd_rec result: ",
			"[mgcd] mod_gcd_rec src/math/polynomial/polynomial.cpp:4223 ---------\nidx: 0\nresult: ",
		} {
			if !strings.Contains(got, part) {
				t.Fatalf("missing recursive source record %q: %q", part, got)
			}
		}
	} else if output.Len() != 0 {
		t.Fatalf("ordinary build emitted trace: %q", output.String())
	}
	if tags.ShouldTrace("mgcd_detail") {
		got := output.String()
		for _, part := range []string{
			"[mgcd_detail] mod_gcd_rec src/math/polynomial/polynomial.cpp:4100 ---------\nidx: 0\nx",
			"[mgcd_detail] mod_gcd_rec src/math/polynomial/polynomial.cpp:4114 ---------\nc_g = ",
			"[mgcd_detail] mod_gcd_rec src/math/polynomial/polynomial.cpp:4149 ---------\ncounter: 0\nidx: 0\nq: ",
			"[mgcd_detail] mod_gcd_rec src/math/polynomial/polynomial.cpp:4152 ---------\nreseting...\n",
			"[mgcd_detail] mod_gcd_rec src/math/polynomial/polynomial.cpp:4167 ---------\nadding sample point...\n",
			"[mgcd_detail] mod_gcd_rec src/math/polynomial/polynomial.cpp:4184 ---------\nidx: 0\ncandidate H: ",
			"[mgcd_detail] mod_gcd_rec src/math/polynomial/polynomial.cpp:4186 ---------\nidx: 0\nlc_H: ",
			"[mgcd_detail] mod_gcd_rec src/math/polynomial/polynomial.cpp:4205 ---------\nC: ",
		} {
			if !strings.Contains(got, part) {
				t.Fatalf("missing recursive detail record %q: %q", part, got)
			}
		}
	}
	tags.Disable("mgcd")
	tags.Disable("mgcd_detail")
	before := output.String()
	serviceMust(t, full.GCD(u, v, r))
	if output.String() != before {
		t.Fatalf("disabled tag emitted trace: %q", output.String())
	}
	refused := &algebraicIntervalRefusingWriter{stop: 10, err: errors.New("trace refused")}
	tags = diagnostic.NewTraceTagsWithWriter(refused)
	tags.Enable("mgcd")
	tags.Enable("mgcd_detail")
	c.monomials.config.Trace = tags
	serviceMust(t, full.GCD(u, v, r))
	same, err = c.Equal(common, r.root.value)
	serviceMust(t, err)
	if !same {
		t.Fatal("recursive modular GCD changed after trace refusal")
	}
	if tags.ShouldTrace("mgcd") && refused.writes != 10 {
		t.Fatalf("refused source stream writes: %d", refused.writes)
	}
}
