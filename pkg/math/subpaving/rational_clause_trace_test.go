// Source: src/math/subpaving/subpaving_t_def.h:802-841; config_mpq.
package subpaving

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestRationalClauseTraceSortedAfterOwnership(t *testing.T) {
	c, m, value, x, y := rationalClauseFixture(t)
	if err := m.SetInt32(value, 5); err != nil {
		t.Fatal(err)
	}
	lower, err := c.NewInequality(x, value, true, false)
	if err != nil {
		t.Fatal(err)
	}
	upper, err := c.NewInequality(y, value, false, true)
	if err != nil {
		t.Fatal(err)
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_clause")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	clause, err := c.AddClause([]*RationalInequality{upper, lower})
	if err != nil {
		t.Fatal(err)
	}
	if clause.atoms[0] != lower || clause.atoms[1] != upper || lower.refs != 1 || upper.refs != 1 || len(c.clauses) != 1 {
		t.Fatal("sorted clause ownership", lower.refs, upper.refs, len(c.clauses))
	}
	if tags.ShouldTrace("subpaving_clause") {
		if got := output.String(); !strings.Contains(got, "subpaving_t_def.h:841") || !strings.Contains(got, "new clause:\n5 <= x0 or x1 < 5\n") || output.flushes != 1 {
			t.Fatalf("source clause trace: %q; flushes %d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased clause trace emitted", output.String(), output.flushes)
	}
}

func TestRationalClauseTraceRefusalAndPrinterFailureKeepClause(t *testing.T) {
	c, m, value, x, y := rationalClauseFixture(t)
	if err := m.SetInt32(value, 9); err != nil {
		t.Fatal(err)
	}
	first, err := c.NewInequality(x, value, false, false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.NewInequality(y, value, true, true)
	if err != nil {
		t.Fatal(err)
	}
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("subpaving_clause")
	if err := c.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	clause, err := c.AddClause([]*RationalInequality{second, first})
	if err != nil || clause == nil || len(c.clauses) != 1 {
		t.Fatal("trace stream refusal changed clause publication", clause, err)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_clause")
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
	retained, err := c.AddClause([]*RationalInequality{second, first})
	if tags.ShouldTrace("subpaving_clause") {
		if !errors.Is(err, printerErr) || retained == nil || len(c.clauses) != 2 || first.refs != 2 || second.refs != 2 || !strings.Contains(output.String(), "new clause:\n") || !strings.Contains(output.String(), "candidate") {
			t.Fatal("source printer failure after clause publication", err, len(c.clauses), first.refs, second.refs, output.String())
		}
	} else if err != nil || retained == nil || len(c.clauses) != 2 {
		t.Fatal("native trace gate changed clause publication", err, len(c.clauses))
	}
}
