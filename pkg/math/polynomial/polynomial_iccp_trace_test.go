// Source: src/math/polynomial/polynomial.cpp:3454-3468; src/util/trace.h:40-46.
package polynomial

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
