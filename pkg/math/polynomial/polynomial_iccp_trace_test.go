package polynomial

// Source: src/math/polynomial/polynomial.cpp:3454-3468; src/util/trace.h:40-46.

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialICCPEntrySourceTraceBeforeShortcuts(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	content, err := NewPolynomialRef(m)
	serviceMust(t, err)
	pp, err := NewPolynomialRef(m)
	serviceMust(t, err)
	var scalar unumeral.Int
	t.Cleanup(func() {
		serviceMust(t, m.integer.Del(&scalar))
		serviceMust(t, pp.Close())
		serviceMust(t, content.Close())
	})
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("polynomial")
	m.monomials.config.Trace = tags
	record := func(body string) string {
		return "-------- [polynomial] iccp src/math/polynomial/polynomial.cpp:3454 ---------\n" +
			"iccp x0\n" + body + "\n------------------------------------------------\n"
	}
	serviceMust(t, full.IntegerContentPrimitiveParts(m.zero, 0, &scalar, content, pp))
	want := ""
	if tags.ShouldTrace("polynomial") {
		want = record("0")
	}
	if output.String() != want {
		t.Fatalf("zero shortcut source trace: got %q, want %q", output.String(), want)
	}
	serviceMust(t, full.IntegerContentPrimitiveParts(m.unit, 0, &scalar, content, pp))
	if tags.ShouldTrace("polynomial") {
		want += record("1")
	}
	if output.String() != want {
		t.Fatalf("constant shortcut source trace: got %q, want %q", output.String(), want)
	}
	tags.Disable("polynomial")
	serviceMust(t, full.IntegerContentPrimitiveParts(m.zero, 0, &scalar, content, pp))
	if output.String() != want {
		t.Fatalf("disabled ICCP trace: %q", output.String())
	}

	refusal := errors.New("ICCP trace refused")
	out := &algebraicIntervalRefusingWriter{stop: 11, err: refusal}
	tags = diagnostic.NewTraceTagsWithWriter(out)
	tags.Enable("polynomial")
	m.monomials.config.Trace = tags
	serviceMust(t, full.IntegerContentPrimitiveParts(m.zero, 0, &scalar, content, pp))
	if tags.ShouldTrace("polynomial") {
		if out.writes != 11 || !strings.HasSuffix(out.String(), "iccp x") {
			t.Fatalf("refused ICCP trace prefix: writes=%d output=%q", out.writes, out.String())
		}
	} else if out.writes != 0 {
		t.Fatalf("ordinary build emitted ICCP trace: %q", out.String())
	}
	tags.Disable("polynomial")
}

func TestPolynomialICCPDegreeSourceTraceBeforeFilterShortcut(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	p := polynomialQueryTerms(t, m, []int32{2}, [][]MonomialPower{{{0, 2}}})
	content, err := NewPolynomialRef(m)
	serviceMust(t, err)
	pp, err := NewPolynomialRef(m)
	serviceMust(t, err)
	var scalar unumeral.Int
	t.Cleanup(func() {
		serviceMust(t, m.integer.Del(&scalar))
		serviceMust(t, pp.Close())
		serviceMust(t, content.Close())
	})
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("polynomial")
	m.monomials.config.Trace = tags
	serviceMust(t, full.IntegerContentPrimitiveParts(p, 0, &scalar, content, pp))
	if scalar != unumeral.NewInt(2) {
		t.Fatalf("integer content changed by trace: %v", scalar)
	}
	if tags.ShouldTrace("polynomial") {
		want := "-------- [polynomial] iccp src/math/polynomial/polynomial.cpp:3454 ---------\n" +
			"iccp x0\n2 x0^2\n------------------------------------------------\n" +
			"-------- [polynomial] iccp src/math/polynomial/polynomial.cpp:3493 ---------\n" +
			"degree of x0 at x0^2 is 2\n------------------------------------------------\n"
		if !strings.HasPrefix(output.String(), want) {
			t.Fatalf("ICCP entry and degree source order: got %q, want prefix %q", output.String(), want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("ordinary build emitted ICCP traces: %q", output.String())
	}
	tags.Disable("polynomial")
}

func TestPolynomialICCPContentAndDivisorSourceTraces(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	p := polynomialQueryTerms(t, m, []int32{6, 12}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{1, 2}}})
	content, err := NewPolynomialRef(m)
	serviceMust(t, err)
	pp, err := NewPolynomialRef(m)
	serviceMust(t, err)
	var scalar unumeral.Int
	t.Cleanup(func() {
		serviceMust(t, m.integer.Del(&scalar))
		serviceMust(t, pp.Close())
		serviceMust(t, content.Close())
	})
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("polynomial")
	m.monomials.config.Trace = tags
	serviceMust(t, full.IntegerContentPrimitiveParts(p, 0, &scalar, content, pp))
	if scalar != unumeral.NewInt(6) {
		t.Fatalf("integer content changed by trace: %v", scalar)
	}
	if tags.ShouldTrace("polynomial") {
		contentRecord := "-------- [polynomial] iccp src/math/polynomial/polynomial.cpp:3513 ---------\n" +
			"p: 12 x1^2 + 6 x0 x1\ni: 6\npp: 2 x1^2 + x0 x1\n------------------------------------------------\n"
		divisorRecord := "-------- [polynomial] iccp src/math/polynomial/polynomial.cpp:3528 ---------\n" +
			"pp: 2 x1^2 + x0 x1\nc: x1\n------------------------------------------------\n"
		contentAt := strings.Index(output.String(), contentRecord)
		divisorAt := strings.Index(output.String(), divisorRecord)
		if contentAt < 0 || divisorAt <= contentAt {
			t.Fatalf("ICCP content/divisor source records: content=%d divisor=%d output=%q", contentAt, divisorAt, output.String())
		}
	} else if output.Len() != 0 {
		t.Fatalf("ordinary build emitted ICCP traces: %q", output.String())
	}
	tags.Disable("polynomial")
}
