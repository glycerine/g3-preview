// Source: src/math/polynomial/polynomial.cpp:4239-4343; src/util/trace.h:40-46.
package polynomial

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialMultivariateModularGCDSourceTrace(t *testing.T) {
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
	tags.Enable("mgcd_call")
	c.monomials.config.Trace = tags
	serviceMust(t, full.GCD(u, v, r))
	same, err := c.Equal(common, r.root.value)
	serviceMust(t, err)
	if !same {
		t.Fatal("modular GCD changed under trace")
	}
	if tags.ShouldTrace("mgcd") {
		got := output.String()
		for _, part := range []string{
			"[mgcd] mod_gcd src/math/polynomial/polynomial.cpp:4239",
			"[mgcd_call] mod_gcd src/math/polynomial/polynomial.cpp:4240",
			"[mgcd] mod_gcd src/math/polynomial/polynomial.cpp:4288",
			"[mgcd] mod_gcd src/math/polynomial/polynomial.cpp:4301",
			"[mgcd] mod_gcd src/math/polynomial/polynomial.cpp:4308",
			"[mgcd] mod_gcd src/math/polynomial/polynomial.cpp:4334",
			"[mgcd] mod_gcd src/math/polynomial/polynomial.cpp:4340",
			"[mgcd] mod_gcd src/math/polynomial/polynomial.cpp:4343",
		} {
			if !strings.Contains(got, part) {
				t.Fatalf("missing source record %q: %q", part, got)
			}
		}
	} else if output.Len() != 0 {
		t.Fatalf("ordinary build emitted trace: %q", output.String())
	}
	tags.Disable("mgcd")
	tags.Disable("mgcd_call")
	before := output.String()
	serviceMust(t, full.GCD(u, v, r))
	if output.String() != before {
		t.Fatalf("disabled tags emitted trace: %q", output.String())
	}
	refused := &algebraicIntervalRefusingWriter{stop: 10, err: errors.New("trace refused")}
	tags = diagnostic.NewTraceTagsWithWriter(refused)
	tags.Enable("mgcd")
	c.monomials.config.Trace = tags
	serviceMust(t, full.GCD(u, v, r))
	same, err = c.Equal(common, r.root.value)
	serviceMust(t, err)
	if !same {
		t.Fatal("modular GCD changed after trace refusal")
	}
	if tags.ShouldTrace("mgcd") && refused.writes != 10 {
		t.Fatalf("refused source stream writes: %d", refused.writes)
	}
}

func TestPolynomialMultivariateModularGCDConstantTrace(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	u := polynomialQueryTerms(t, c, []int32{1, 1, 1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}, nil})
	v := polynomialQueryTerms(t, c, []int32{1, 2, 3}, [][]MonomialPower{{{0, 1}}, {{1, 1}}, nil})
	r, err := NewPolynomialRef(c)
	serviceMust(t, err)
	t.Cleanup(func() { serviceMust(t, full.CleanupGCD()); serviceMust(t, r.Close()) })
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mgcd")
	c.monomials.config.Trace = tags
	serviceMust(t, full.GCD(u, v, r))
	polynomialQueryExpect(t, c, r.root.value, map[[3]uint32]int64{{}: 1})
	if tags.ShouldTrace("mgcd") {
		got := output.String()
		if !strings.Contains(got, "[mgcd] mod_gcd src/math/polynomial/polynomial.cpp:4310 ---------\ndone, modular gcd is one\n") {
			t.Fatalf("missing constant-GCD source record: %q", got)
		}
	} else if output.Len() != 0 {
		t.Fatalf("ordinary build emitted trace: %q", output.String())
	}
}
