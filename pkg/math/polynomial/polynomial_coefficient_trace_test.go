package polynomial

// Source: src/math/polynomial/polynomial.cpp:3264-3276; src/util/trace.h:40-46.

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialCoefficientSourceTraceAndStreamRefusal(t *testing.T) {
	m := polynomialCellFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("coeff_bug")
	m.monomials.config.Trace = tags
	p, err := m.MkVariable(0, 2)
	serviceMust(t, err)
	serviceMust(t, m.IncRef(p))
	defer func() { serviceMust(t, m.DecRef(p)) }()
	record := func(k uint32) string {
		return "-------- [coeff_bug] coeff src/math/polynomial/polynomial.cpp:3267 ---------\n" +
			"p: x0^2\nx: 0, k: " + strconv.FormatUint(uint64(k), 10) + "\n------------------------------------------------\n"
	}
	r, err := m.Coeff(p, 0, 2)
	serviceMust(t, err)
	serviceMust(t, m.IncRef(r))
	serviceMust(t, m.DecRef(r))
	want := ""
	if tags.ShouldTrace("coeff_bug") {
		want = record(2)
	}
	if output.String() != want {
		t.Fatalf("coefficient source trace: got %q, want %q", output.String(), want)
	}
	reduct, err := NewPolynomialRef(m)
	serviceMust(t, err)
	r, err = m.CoeffWithReduct(p, 0, 1, reduct)
	serviceMust(t, err)
	serviceMust(t, m.IncRef(r))
	serviceMust(t, m.DecRef(r))
	serviceMust(t, reduct.Close())
	if output.String() != want {
		t.Fatalf("reduct overload must not trace: got %q, want %q", output.String(), want)
	}
	tags.Disable("coeff_bug")
	r, err = m.Coeff(p, 0, 1)
	serviceMust(t, err)
	serviceMust(t, m.IncRef(r))
	serviceMust(t, m.DecRef(r))
	if output.String() != want {
		t.Fatalf("disabled coefficient trace: %q", output.String())
	}
	if tags.ShouldTrace("coeff_bug") {
		t.Fatal("tag remained enabled")
	}

	refusal := errors.New("coefficient trace refused")
	out := &algebraicIntervalRefusingWriter{stop: 11, err: refusal}
	tags = diagnostic.NewTraceTagsWithWriter(out)
	tags.Enable("coeff_bug")
	m.monomials.config.Trace = tags
	r, err = m.Coeff(p, 0, 2)
	serviceMust(t, err)
	serviceMust(t, m.IncRef(r))
	serviceMust(t, m.DecRef(r))
	if tags.ShouldTrace("coeff_bug") {
		if out.writes != 11 || !strings.HasSuffix(out.String(), "p: ") {
			t.Fatalf("refused coefficient trace prefix: writes=%d output=%q", out.writes, out.String())
		}
	} else if out.writes != 0 {
		t.Fatalf("ordinary build emitted coefficient trace: %q", out.String())
	}
	tags.Disable("coeff_bug")
}
