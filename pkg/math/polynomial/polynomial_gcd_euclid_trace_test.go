package polynomial

// Source: src/math/polynomial/polynomial.cpp:3641-3642; src/util/trace.h:46.

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialEuclidNonUnivariateConditionalSourceTrace(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	_, err := c.monomials.MkVar()
	serviceMust(t, err)
	_, err = c.monomials.MkVar()
	serviceMust(t, err)
	serviceMust(t, c.modular.SetZPUint64(11))
	u := polynomialQueryTerms(t, c, []int32{1, 1, 1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}, nil})
	v := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, 1}}, nil})
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mgcd")
	c.monomials.config.Trace = tags
	serviceMust(t, full.traceEuclidNonUnivariate(u, v))
	if tags.ShouldTrace("mgcd") {
		want := "-------- [mgcd] euclid_gcd src/math/polynomial/polynomial.cpp:3641 ---------\n" +
			"euclid_gcd, polynomials are not univariate\nx1 + x0 + 1\nx0 + 2\n------------------------------------------------\n"
		if output.String() != want {
			t.Fatalf("source conditional trace: got %q, want %q", output.String(), want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("ordinary build emitted trace: %q", output.String())
	}
	plain := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	beforeValid := output.String()
	serviceMust(t, full.traceEuclidNonUnivariate(plain, v))
	if output.String() != beforeValid {
		t.Fatalf("univariate inputs emitted conditional trace: %q", output.String())
	}
	if polynomialSourceDebug {
		polynomialThrowAssertionFixture(t)
		r, e := NewPolynomialRef(c)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, r.Close()) })
		before := output.String()
		if e = full.gcdEuclid(u, v, &r.root); !errors.Is(e, ErrPolynomialAssertion) {
			t.Fatalf("source univariate assertion: %v", e)
		}
		if tags.ShouldTrace("mgcd") && output.String() != before+before {
			t.Fatalf("diagnostic must precede assertion: %q", output.String())
		}
	}
	tags.Disable("mgcd")
	before := output.String()
	serviceMust(t, full.traceEuclidNonUnivariate(u, v))
	if output.String() != before {
		t.Fatalf("disabled tag emitted trace: %q", output.String())
	}
	refused := &algebraicIntervalRefusingWriter{stop: 10, err: errors.New("trace refused")}
	tags = diagnostic.NewTraceTagsWithWriter(refused)
	tags.Enable("mgcd")
	c.monomials.config.Trace = tags
	serviceMust(t, full.traceEuclidNonUnivariate(u, v))
	if tags.ShouldTrace("mgcd") && refused.writes != 10 {
		t.Fatalf("refused source stream writes: %d", refused.writes)
	}
}
