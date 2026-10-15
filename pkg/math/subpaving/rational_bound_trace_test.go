// Source: src/math/subpaving/subpaving_t_def.h:525-570; config_mpq.
package subpaving

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestRationalBoundTracePublicationAndConflictOrder(t *testing.T) {
	c, m, value, x, _ := rationalClauseFixture(t)
	n, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_mk_bound")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(value, 3); err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(n, x, value, true, false); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(value, 2); err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(n, x, value, false, false); err != nil {
		t.Fatal(err)
	}
	if n.conflictVar != x || c.numConflicts != 1 || c.timestamp != 2 {
		t.Fatal("source conflict publication", n.conflictVar, c.numConflicts, c.timestamp)
	}
	if tags.ShouldTrace("subpaving_mk_bound") {
		got := output.String()
		if !strings.Contains(got, "subpaving_t_def.h:563") || !strings.Contains(got, "mk_bound: 3 <= x0\ntimestamp: 0\n") || !strings.Contains(got, "mk_bound: x0 <= 2\ntimestamp: 1\n") || !strings.Contains(got, "subpaving_t_def.h:565") || !strings.Contains(got, "conflict\n3 <= x0 x0 <= 2\n") || output.flushes != 3 {
			t.Fatalf("source bound/conflict trace: %q; flushes %d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased source bound trace emitted", output.String(), output.flushes)
	}
}

func TestRationalBoundTraceStreamAndPrinterFailurePrefix(t *testing.T) {
	c, m, value, x, _ := rationalClauseFixture(t)
	n, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(value, 4); err != nil {
		t.Fatal(err)
	}
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("subpaving_mk_bound")
	if err := c.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(n, x, value, true, false); err != nil || c.timestamp != 1 {
		t.Fatal("stream refusal changed bound publication", err, c.timestamp)
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_mk_bound")
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
	err = c.PublishBound(n, x, value, false, false)
	if tags.ShouldTrace("subpaving_mk_bound") {
		if !errors.Is(err, printerErr) || n.trail == nil || n.trail.lower || c.timestamp != 1 || n.conflictVar == x || !strings.Contains(output.String(), "mk_bound: candidate") || output.flushes != 0 {
			t.Fatal("source stopped after bound push before conflict check", err, c.timestamp, n.conflictVar, output.String(), output.flushes)
		}
	} else if err != nil || c.timestamp != 2 {
		t.Fatal("native trace gate changed publication", err, c.timestamp)
	}
}
