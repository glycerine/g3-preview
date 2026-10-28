package subpaving

// Source: src/math/subpaving/subpaving_t_def.h:865-868; config_mpq.

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestRationalUnitTracePrecedesReferenceAndPublication(t *testing.T) {
	c, m, value, x, _ := rationalClauseFixture(t)
	if err := m.SetInt32(value, 7); err != nil {
		t.Fatal(err)
	}
	atom, err := c.NewInequality(x, value, true, false)
	if err != nil {
		t.Fatal(err)
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.AddUnitClause(atom, true); err != nil {
		t.Fatal(err)
	}
	if atom.refs != 1 || len(c.unitClauses) != 1 || c.unitClauses[0].atom != atom {
		t.Fatal("unit reference/publication", atom.refs, len(c.unitClauses))
	}
	if tags.ShouldTrace("subpaving") {
		if got := output.String(); !strings.Contains(got, "subpaving_t_def.h:866") || !strings.Contains(got, "7 <= x0\n") || output.flushes != 1 {
			t.Fatalf("source unit trace: %q; flushes %d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased source trace emitted", output.String(), output.flushes)
	}
}

func TestRationalUnitTraceFailureBoundaries(t *testing.T) {
	c, m, value, x, _ := rationalClauseFixture(t)
	if err := m.SetInt32(value, 11); err != nil {
		t.Fatal(err)
	}
	atom, err := c.NewInequality(x, value, false, true)
	if err != nil {
		t.Fatal(err)
	}
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("subpaving")
	if err := c.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if err := c.AddUnitClause(atom, false); err != nil || atom.refs != 1 || len(c.unitClauses) != 1 {
		t.Fatal("stream refusal changed unit publication", err, atom.refs, len(c.unitClauses))
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	printerErr := errors.New("borrowed variable printer stopped")
	if err := c.SetVariableDisplay(func(w io.Writer, variable uint32) error {
		_, _ = io.WriteString(w, "candidate")
		return printerErr
	}); err != nil {
		t.Fatal(err)
	}
	err = c.AddUnitClause(atom, true)
	if tags.ShouldTrace("subpaving") {
		if !errors.Is(err, printerErr) || atom.refs != 1 || len(c.unitClauses) != 1 || !strings.Contains(output.String(), "candidate") {
			t.Fatal("printer failure prefix or pre-publication boundary", err, atom.refs, len(c.unitClauses), output.String())
		}
	} else if err != nil || atom.refs != 2 || len(c.unitClauses) != 2 {
		t.Fatal("native trace gate changed unit publication", err, atom.refs, len(c.unitClauses))
	}
}
