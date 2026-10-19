package polynomial

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialGCDMGCDCheckUsesIndependentPRS(t *testing.T) {
	polynomialAssertionActionsFixture(t, "T")
	debug := diagnostic.BorrowProcessDebugContext()
	serviceMust(t, debug.EnableDebug("mgcd_check"))
	t.Cleanup(func() { serviceMust(t, debug.DisableDebug("mgcd_check")) })
	full := polynomialPSCFullFixture(t)
	c := full.core
	var trace bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	tags.Enable("polynomial_gcd")
	tags.Enable("mgcd_bug")
	c.monomials.config.Trace = tags
	common := polynomialQueryTerms(t, c, []int32{1, 1, 1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}, nil})
	left := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, 1}}, nil})
	right := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}}, nil})
	u, e := c.Mul(common, left)
	serviceMust(t, e)
	u = polynomialCaller(t, c, u)
	v, e := c.Mul(common, right)
	serviceMust(t, e)
	v = polynomialCaller(t, c, v)
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, full.CleanupGCD()); serviceMust(t, out.Close()) })
	serviceMust(t, full.GCD(u, v, out))
	equal, e := c.Equal(out.root.value, common)
	serviceMust(t, e)
	if !equal {
		t.Fatal("mgcd_check changed modular GCD result")
	}
	got := trace.String()
	if polynomialSourceDebug && tags.ShouldTrace("polynomial_gcd") {
		if !strings.Contains(got, "[polynomial_gcd] gcd_prs src/math/polynomial/polynomial.cpp:3556") {
			t.Fatalf("mgcd_check did not run independent PRS: %q", got)
		}
	} else if strings.Contains(got, "[polynomial_gcd] gcd_prs src/math/polynomial/polynomial.cpp:3556") {
		t.Fatalf("release build ran debug-only mgcd_check: %q", got)
	}
	if strings.Contains(got, "[mgcd_bug]") {
		t.Fatalf("equal modular and PRS results emitted mismatch trace: %q", got)
	}
}

func TestPolynomialGCDMGCDCheckMismatchTrace(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	var trace bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	tags.Enable("mgcd_bug")
	c.monomials.config.Trace = tags
	u := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	v := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, 1}}, nil})
	serviceMust(t, full.traceMGCDCheckMismatch(u, v, u, v))
	got := trace.String()
	if tags.ShouldTrace("mgcd_bug") {
		for _, want := range []string{
			"[mgcd_bug] gcd src/math/polynomial/polynomial.cpp:4491",
			"u: x0 + 1\nv: x0 + 2\nr1: x0 + 1\nr2: x0 + 2\n",
		} {
			if !strings.Contains(got, want) {
				t.Fatalf("missing source mismatch trace %q in %q", want, got)
			}
		}
	} else if got != "" {
		t.Fatalf("non-trace build emitted mgcd_bug: %q", got)
	}
	trace.Reset()
	serviceMust(t, full.traceMGCDCheckMismatch(u, v, u, u))
	if trace.Len() != 0 {
		t.Fatalf("equal results emitted conditional mismatch trace: %q", trace.String())
	}
}
