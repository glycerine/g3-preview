package subpaving

// Source: src/math/subpaving/subpaving_t_def.h:1376-1402; config_mpfx.

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func fixedClauseTracePropagationFixture(t *testing.T) (*FixedContext, *FixedClause, *FixedNode) {
	t.Helper()
	m, c, v, root := fixedClauseFixture(t, []bool{false, false})
	xLower := fixedClauseAtom(t, m, c, v, 0, 5, true, false)
	yUpper := fixedClauseAtom(t, m, c, v, 1, 2, false, false)
	clause, err := c.AddClause([]*FixedInequality{yUpper, xLower}, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt64(v, 4); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(root, 0, v, false, false); err != nil {
		t.Fatal(err)
	}
	return c, clause, root
}

func TestFixedClausePropagationTraceSourceOrder(t *testing.T) {
	c, clause, root := fixedClauseTracePropagationFixture(t)
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("propagate_clause")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateClause(clause, root); err != nil || root.upper[1] == nil || c.numVisited != 1 {
		t.Fatal("source clause propagation", err, c.numVisited)
	}
	if tags.ShouldTrace("propagate_clause") {
		got := output.String()
		if !strings.Contains(got, "subpaving_t_def.h:1376") || !strings.Contains(got, "propagate using:\n5 <= x0 or x1 <= 2\n") ||
			!strings.Contains(got, "subpaving_t_def.h:1400") || !strings.Contains(got, "propagating inequality: x1 <= 2\n") ||
			strings.Index(got, "subpaving_t_def.h:1376") > strings.Index(got, "subpaving_t_def.h:1400") || output.flushes != 2 {
			t.Fatalf("source clause propagation TRACE: %q, flushes=%d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased clause propagation TRACE", output.String())
	}
}

func TestFixedClausePropagationTraceFailureBoundaries(t *testing.T) {
	c, clause, root := fixedClauseTracePropagationFixture(t)
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("propagate_clause")
	if err := c.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateClause(clause, root); err != nil || root.upper[1] == nil {
		t.Fatal("borrowed trace refusal changed clause propagation", err)
	}

	d, otherClause, otherRoot := fixedClauseTracePropagationFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("propagate_clause")
	if err := d.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	printerErr := errors.New("borrowed variable printer stopped")
	e, firstClause, firstRoot := fixedClauseTracePropagationFixture(t)
	var firstOutput bytes.Buffer
	firstTags := diagnostic.NewTraceTagsWithWriter(&firstOutput)
	firstTags.Enable("propagate_clause")
	if err := e.BindTrace(firstTags); err != nil {
		t.Fatal(err)
	}
	if err := e.SetVariableDisplay(func(w io.Writer, variable uint32) error {
		_, _ = io.WriteString(w, "candidate")
		return printerErr
	}); err != nil {
		t.Fatal(err)
	}
	firstErr := e.PropagateClause(firstClause, firstRoot)
	if firstTags.ShouldTrace("propagate_clause") {
		if !errors.Is(firstErr, printerErr) || e.numVisited != 0 || firstRoot.upper[1] != nil ||
			!strings.Contains(firstOutput.String(), "propagate using:\n") {
			t.Fatal("stopped before clause visit", firstErr, e.numVisited, firstOutput.String())
		}
	} else if firstErr != nil || firstRoot.upper[1] == nil {
		t.Fatal("native gate changed first trace boundary", firstErr)
	}
	calls := 0
	if err := d.SetVariableDisplay(func(w io.Writer, variable uint32) error {
		calls++
		_, _ = io.WriteString(w, "candidate")
		if calls == 3 {
			return printerErr
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	err := d.PropagateClause(otherClause, otherRoot)
	if tags.ShouldTrace("propagate_clause") {
		if !errors.Is(err, printerErr) || d.numVisited != 1 || otherRoot.upper[1] != nil ||
			!strings.Contains(output.String(), "propagate using:\n") || !strings.Contains(output.String(), "propagating inequality: candidate") {
			t.Fatal("stopped after visit before bound publication", err, d.numVisited, output.String())
		}
	} else if err != nil || otherRoot.upper[1] == nil {
		t.Fatal("native gate changed clause propagation", err)
	}
}
