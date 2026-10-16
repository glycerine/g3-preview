// Source: src/math/polynomial/polynomial.cpp:3012-3016; src/util/trace.h:40-46.
package polynomial

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialSkeletonSourceTraceOfGroupedPowers(t *testing.T) {
	m := polynomialCellFixture(t)
	p, _ := polynomialSkeletonInput(t, m)
	defer func() { serviceMust(t, m.DecRef(p)) }()
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("skeleton")
	m.monomials.config.Trace = tags
	s, err := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, err)
	var display polynomialDisplayTestSink
	serviceMust(t, m.DisplayInto(p, &display, false))
	want := ""
	if tags.ShouldTrace("skeleton") {
		want = "-------- [skeleton] skeleton src/math/polynomial/polynomial.cpp:3012 ---------\n" +
			"x: x0\nmax: 2\np: " + display.text.String() + "\n" +
			"skeleton: (x0^4)*1 (x0^0 x0^2)*x1^2 (x0^1 x0^3)*x2\n" +
			"------------------------------------------------\n"
	}
	if output.String() != want {
		t.Fatalf("grouped skeleton source record: got %q, want %q", output.String(), want)
	}
	serviceMust(t, s.CloseChecked())
	tags.Disable("skeleton")
	s, err = NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, err)
	serviceMust(t, s.CloseChecked())
	if output.String() != want {
		t.Fatalf("disabled skeleton trace: %q", output.String())
	}

	refusal := errors.New("skeleton trace refused")
	out := &algebraicIntervalRefusingWriter{stop: 10, err: refusal}
	tags = diagnostic.NewTraceTagsWithWriter(out)
	tags.Enable("skeleton")
	m.monomials.config.Trace = tags
	s, err = NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, err)
	serviceMust(t, s.CloseChecked())
	if tags.ShouldTrace("skeleton") {
		if out.writes != 10 || !strings.HasSuffix(out.String(), "---------\n") {
			t.Fatalf("refused skeleton trace prefix: writes=%d output=%q", out.writes, out.String())
		}
	} else if out.writes != 0 {
		t.Fatalf("ordinary build emitted skeleton trace: %q", out.String())
	}
	tags.Disable("skeleton")
}
