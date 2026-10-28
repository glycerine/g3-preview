package polynomial

// Source: src/math/polynomial/polynomial.cpp:2833-2901; src/util/trace.h:40-46.

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialNewtonSourceTraceSequenceAndRefusal(t *testing.T) {
	m := polynomialCellFixture(t)
	serviceMust(t, m.SetZPUint64(97))
	n, err := NewPolynomialNewtonInterpolator(m)
	serviceMust(t, err)
	defer func() { serviceMust(t, n.Close()) }()
	r, err := NewPolynomialRef(m)
	serviceMust(t, err)
	defer func() { serviceMust(t, r.Close()) }()
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("newton")
	m.monomials.config.Trace = tags
	zero, one := unumeral.NewInt(0), unumeral.NewInt(1)
	serviceMust(t, n.Add(&zero, m.unit))
	serviceMust(t, n.Add(&one, m.unit))
	serviceMust(t, n.Mk(0, r))
	result, err := r.Get()
	serviceMust(t, err)
	same, err := m.Equal(result, m.unit)
	serviceMust(t, err)
	if !same {
		t.Fatal("trace changed Newton result")
	}
	record := func(function string, line int, body string) string {
		return "-------- [newton] " + function + " src/math/polynomial/polynomial.cpp:" +
			strconv.Itoa(line) + " ---------\n" + body + "\n------------------------------------------------\n"
	}
	want := ""
	if tags.ShouldTrace("newton") {
		want = record("add", 2833, "0 -> 1") +
			record("add", 2833, "1 -> 1") +
			record("add", 2851, "invs[1]: 1") +
			record("add", 2871, "vs[1]: 0") +
			record("mk", 2890, "u: 0") +
			record("mk", 2896, "(x - inputs[k]): x0") +
			record("mk", 2901, "result u: 1")
	}
	if output.String() != want {
		t.Fatalf("Newton source trace sequence: got %q, want %q", output.String(), want)
	}
	tags.Disable("newton")
	serviceMust(t, n.Reset())
	serviceMust(t, n.Add(&zero, m.unit))
	serviceMust(t, n.Mk(0, r))
	if output.String() != want {
		t.Fatalf("disabled Newton trace: %q", output.String())
	}

	serviceMust(t, n.Reset())
	refusal := errors.New("Newton trace refused")
	out := &algebraicIntervalRefusingWriter{stop: 10, err: refusal}
	tags = diagnostic.NewTraceTagsWithWriter(out)
	tags.Enable("newton")
	m.monomials.config.Trace = tags
	serviceMust(t, n.Add(&zero, m.unit))
	serviceMust(t, n.Add(&one, m.unit))
	serviceMust(t, n.Mk(0, r))
	if tags.ShouldTrace("newton") {
		if out.writes != 10 || !strings.HasSuffix(out.String(), "---------\n") {
			t.Fatalf("refused Newton trace prefix: writes=%d output=%q", out.writes, out.String())
		}
	} else if out.writes != 0 {
		t.Fatalf("ordinary build emitted Newton trace: %q", out.String())
	}
	tags.Disable("newton")
}
