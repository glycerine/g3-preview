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

func TestPolynomialSkeletonLexComparatorTraceUsesLiveGoSortCalls(t *testing.T) {
	m := polynomialCellFixture(t)
	p, _ := polynomialSkeletonInput(t, m)
	defer func() { serviceMust(t, m.DecRef(p)) }()
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("lex_bug")
	m.monomials.config.Trace = tags
	s, err := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, err)
	serviceMust(t, s.CloseChecked())
	if tags.ShouldTrace("lex_bug") {
		const head = "-------- [lex_bug] operator() src/math/polynomial/polynomial.cpp:684 ---------\n"
		const foot = "------------------------------------------------\n"
		valid := map[string]bool{
			"x0^3 x2": true, "x0 x2": true, "x1^2": true, "x0^2 x1^2": true, "x0^4": true,
		}
		parts := strings.Split(output.String(), foot)
		if len(parts) < 3 || parts[len(parts)-1] != "" {
			t.Fatalf("missing comparator records: %q", output.String())
		}
		for _, part := range parts[:len(parts)-1] {
			if !strings.HasPrefix(part, head) {
				t.Fatalf("comparator source header: %q", part)
			}
			lines := strings.Split(strings.TrimPrefix(part, head), "\n")
			if len(lines) != 4 || lines[0] != "min: x0" || !valid[lines[1]] || !valid[lines[2]] || lines[1] == lines[2] || lines[3] != "" {
				t.Fatalf("comparator source operands: %q", part)
			}
		}
	} else if output.Len() != 0 {
		t.Fatalf("ordinary build emitted comparator trace: %q", output.String())
	}
	want := output.String()
	tags.Disable("lex_bug")
	s, err = NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, err)
	serviceMust(t, s.CloseChecked())
	if output.String() != want {
		t.Fatalf("disabled comparator trace: %q", output.String())
	}

	refusal := errors.New("comparator trace refused")
	out := &algebraicIntervalRefusingWriter{stop: 11, err: refusal}
	tags = diagnostic.NewTraceTagsWithWriter(out)
	tags.Enable("lex_bug")
	m.monomials.config.Trace = tags
	s, err = NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, err)
	serviceMust(t, s.CloseChecked())
	if tags.ShouldTrace("lex_bug") {
		if out.writes != 11 || !strings.HasSuffix(out.String(), "min: x") {
			t.Fatalf("refused comparator trace prefix: writes=%d output=%q", out.writes, out.String())
		}
	} else if out.writes != 0 {
		t.Fatalf("ordinary build emitted comparator trace: %q", out.String())
	}
	tags.Disable("lex_bug")
}
