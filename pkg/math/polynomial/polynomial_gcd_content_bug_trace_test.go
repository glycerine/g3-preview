// Source: src/math/polynomial/polynomial.cpp:4002-4006; src/util/trace.h:46.
package polynomial

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialICCPZPXContentBugConditionalSourceTrace(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	_, err := c.monomials.MkVar()
	serviceMust(t, err)
	_, err = c.monomials.MkVar()
	serviceMust(t, err)
	serviceMust(t, c.modular.SetZPUint64(11))
	g := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	pp := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{1, 1}}, nil})
	divisible, err := c.Mul(g, pp)
	serviceMust(t, err)
	polynomialCaller(t, c, divisible)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("content_bug")
	c.monomials.config.Trace = tags
	serviceMust(t, full.traceICCPZPXContentBug(g, divisible, 0, nil))
	if output.Len() != 0 {
		t.Fatalf("divisible content emitted conditional trace: %q", output.String())
	}
	serviceMust(t, full.traceICCPZPXContentBug(g, pp, 0, nil))
	if tags.ShouldTrace("content_bug") {
		want := "-------- [content_bug] iccp_ZpX src/math/polynomial/polynomial.cpp:4002 ---------\n" +
			"GF(11)\npp: x1 + 1\nvar: x0\ncontent: x0 + 1\n------------------------------------------------\n"
		if output.String() != want {
			t.Fatalf("conditional source trace: got %q, want %q", output.String(), want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("ordinary build emitted trace: %q", output.String())
	}
	tags.Disable("content_bug")
	before := output.String()
	serviceMust(t, full.traceICCPZPXContentBug(g, pp, 0, nil))
	if output.String() != before {
		t.Fatalf("disabled tag emitted trace: %q", output.String())
	}
	refused := &algebraicIntervalRefusingWriter{stop: 10, err: errors.New("trace refused")}
	tags = diagnostic.NewTraceTagsWithWriter(refused)
	tags.Enable("content_bug")
	c.monomials.config.Trace = tags
	serviceMust(t, full.traceICCPZPXContentBug(g, pp, 0, nil))
	if tags.ShouldTrace("content_bug") {
		if refused.writes != 10 || !strings.HasSuffix(refused.String(), "---------\n") {
			t.Fatalf("refused conditional header: writes=%d output=%q", refused.writes, refused.String())
		}
	} else if refused.writes != 0 {
		t.Fatalf("ordinary build emitted trace into refused stream: %q", refused.String())
	}
}
