// Source: src/math/polynomial/polynomial.cpp:2421-2435; src/util/trace.h:40-46.
package polynomial

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

var polynomialDeletionTraceObservation struct {
	output *bytes.Buffer
	want   string
	calls  int
}

func TestPolynomialDeletionTraceStreamRefusalStillDeletes(t *testing.T) {
	m := polynomialCellFixture(t)
	refusal := errors.New("trace output refused")
	out := &algebraicIntervalRefusingWriter{stop: 11, err: refusal}
	tags := diagnostic.NewTraceTagsWithWriter(out)
	tags.Enable("polynomial")
	m.monomials.config.Trace = tags
	p, err := m.MkVariable(0, 2)
	serviceMust(t, err)
	serviceMust(t, m.IncRef(p))
	serviceMust(t, m.DecRef(p))
	if tags.ShouldTrace("polynomial") {
		if out.writes != 11 || !strings.HasSuffix(out.String(), "deleting: ") {
			t.Fatalf("stream refusal must preserve the reached prefix: writes=%d output=%q", out.writes, out.String())
		}
	} else if out.writes != 0 {
		t.Fatalf("ordinary build emitted a deletion trace: %q", out.String())
	}
	tags.Disable("polynomial")
	serviceMust(t, m.CloseChecked())
}

func polynomialDeletionTraceCall(_ *PolynomialDeletionHandler, m *MultivariatePolynomialCore, p Polynomial) error {
	o := &polynomialDeletionTraceObservation
	o.calls++
	if o.output.String() != o.want {
		return errors.New("deletion callback preceded the source trace")
	}
	_, err := m.Info(p)
	return err
}

func TestPolynomialDeletionSourceTraceBeforeHandlerAndDuringClose(t *testing.T) {
	m := polynomialCellFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("polynomial")
	m.monomials.config.Trace = tags
	record := func(body string) string {
		return "-------- [polynomial] del src/math/polynomial/polynomial.cpp:2421 ---------\n" +
			"deleting: " + body + "\n------------------------------------------------\n"
	}
	want := ""
	if tags.ShouldTrace("polynomial") {
		want = record("x0^2")
	}
	polynomialDeletionTraceObservation.output = &output
	polynomialDeletionTraceObservation.want = want
	polynomialDeletionTraceObservation.calls = 0
	h, err := NewPolynomialDeletionHandler(m.integer.Budget(), polynomialDeletionTraceCall)
	serviceMust(t, err)
	serviceMust(t, m.AddDeletionHandler(h))
	p, err := m.MkVariable(0, 2)
	serviceMust(t, err)
	serviceMust(t, m.IncRef(p))
	serviceMust(t, m.DecRef(p))
	if polynomialDeletionTraceObservation.calls != 1 || output.String() != want {
		t.Fatalf("deletion trace/order: calls=%d output=%q", polynomialDeletionTraceObservation.calls, output.String())
	}
	serviceMust(t, m.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
	tags.Disable("polynomial")
	p, err = m.MkVariable(1, 1)
	serviceMust(t, err)
	serviceMust(t, m.IncRef(p))
	serviceMust(t, m.DecRef(p))
	if output.String() != want {
		t.Fatal("disabled deletion trace", output.String())
	}
	tags.Enable("polynomial")
	serviceMust(t, m.CloseChecked())
	if tags.ShouldTrace("polynomial") {
		want += record("0") + record("1")
	}
	if output.String() != want {
		t.Fatalf("teardown deletion trace: got %q, want %q", output.String(), want)
	}
	polynomialDeletionTraceObservation.output = nil
}
