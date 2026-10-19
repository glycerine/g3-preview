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

func TestPolynomialGCDMGCDCheckRereadsTagAndConstructsDebugLocals(t *testing.T) {
	for _, initial := range []bool{false, true} {
		t.Run(map[bool]string{false: "disabled", true: "enabled-then-disabled"}[initial], func(t *testing.T) {
			polynomialAssertionActionsFixture(t, "T")
			debug := diagnostic.BorrowProcessDebugContext()
			serviceMust(t, debug.DisableDebug("mgcd_check"))
			if initial {
				serviceMust(t, debug.EnableDebug("mgcd_check"))
			}
			t.Cleanup(func() { serviceMust(t, debug.DisableDebug("mgcd_check")) })
			full := polynomialPSCFullFixture(t)
			c := full.core
			checked := false
			writer := &sparseBorrowProbeWriter{probe: func() {
				if len(full.gcdFrames) != 0 && full.gcdFrames[0].refs[2].owner != nil {
					checked = true
				}
			}}
			tags := diagnostic.NewTraceTagsWithWriter(writer)
			tags.Enable("polynomial_gcd")
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
			switched := false
			original := c.config.Cooperation
			c.config.Cooperation = func() error {
				if !switched && full.gcdDepth >= 2 {
					switched = true
					caller := full.gcdFrames[0]
					want := uint8(2)
					if polynomialSourceDebug {
						want = 4
					}
					if caller.depth != want {
						t.Errorf("debug locals before modular child: depth %d, want %d", caller.depth, want)
					}
					if polynomialSourceDebug {
						for i := 0; i < 2; i++ {
							expected := Polynomial{}
							if initial {
								expected = []Polynomial{u, v}[i]
							}
							if caller.refs[i].owner != c || caller.refs[i].root.value != expected {
								t.Errorf("debug original %d has wrong construction/capture", i)
							}
						}
					}
					serviceMust(t, debug.DisableDebug("mgcd_check"))
				}
				return original()
			}
			serviceMust(t, full.GCD(u, v, out))
			if !switched {
				t.Fatal("actual modular cooperation not reached")
			}
			if checked {
				t.Fatal("disabled second tag read still constructed PRS check result")
			}
			equal, e := c.Equal(out.root.value, common)
			serviceMust(t, e)
			if !equal {
				t.Fatal("tag transition changed modular GCD result")
			}
		})
	}
}
