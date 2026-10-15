package subpaving

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestHardwareClauseAndUnitTracePublicationOrder(t *testing.T) {
	c, _, two, _ := hardwarePolynomialTraceFixture(t)
	lower, err := c.NewInequality(0, &two, true, false)
	if err != nil {
		t.Fatal(err)
	}
	upper, err := c.NewInequality(1, &two, false, true)
	if err != nil {
		t.Fatal(err)
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving")
	tags.Enable("subpaving_clause")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.AddUnitClause(lower, true); err != nil {
		t.Fatal(err)
	}
	clause, err := c.AddClause([]*HardwareInequality{upper, lower}, false, true)
	if err != nil || clause == nil || clause.atoms[0] != lower || clause.atoms[1] != upper || lower.refs != 2 || upper.refs != 1 || len(c.unitClauses) != 1 || len(c.clauses) != 1 {
		t.Fatal("unit and sorted clause ownership", err)
	}
	if tags.ShouldTrace("subpaving") {
		got := output.String()
		if !strings.Contains(got, "subpaving_t_def.h:866") || !strings.Contains(got, "2 <= x0\n") ||
			!strings.Contains(got, "subpaving_t_def.h:841") || !strings.Contains(got, "new clause:\n2 <= x0 or x1 < 2\n") ||
			strings.Index(got, "subpaving_t_def.h:866") > strings.Index(got, "subpaving_t_def.h:841") || output.flushes != 2 {
			t.Fatalf("HWF unit/clause TRACE: %q, flushes=%d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased HWF unit/clause TRACE emitted", output.String())
	}
}

func TestHardwareClauseAndUnitTraceFailureBoundaries(t *testing.T) {
	c, _, two, _ := hardwarePolynomialTraceFixture(t)
	first, err := c.NewInequality(0, &two, false, false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.NewInequality(1, &two, true, false)
	if err != nil {
		t.Fatal(err)
	}
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("subpaving")
	failed.Enable("subpaving_clause")
	if err := c.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if err := c.AddUnitClause(first, false); err != nil || first.refs != 1 || len(c.unitClauses) != 1 {
		t.Fatal("stream refusal changed unit publication", err)
	}
	clause, err := c.AddClause([]*HardwareInequality{second, first}, false, true)
	if err != nil || clause == nil || first.refs != 2 || second.refs != 1 || len(c.clauses) != 1 {
		t.Fatal("stream refusal changed clause publication", err)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving")
	tags.Enable("subpaving_clause")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	printerErr := errors.New("borrowed variable printer stopped")
	if err := c.SetVariableDisplay(func(w io.Writer, _ uint32) error {
		_, _ = io.WriteString(w, "candidate")
		return printerErr
	}); err != nil {
		t.Fatal(err)
	}
	err = c.AddUnitClause(first, true)
	if tags.ShouldTrace("subpaving") {
		if !errors.Is(err, printerErr) || first.refs != 2 || len(c.unitClauses) != 1 || !strings.Contains(output.String(), "candidate") {
			t.Fatal("HWF unit TRACE stopped before reference", err, first.refs, output.String())
		}
	} else if err != nil || first.refs != 3 || len(c.unitClauses) != 2 {
		t.Fatal("native gate changed HWF unit publication", err)
	}
	retained, err := c.AddClause([]*HardwareInequality{second, first}, false, true)
	if tags.ShouldTrace("subpaving_clause") {
		wantFirst := uint32(3)
		if !tags.ShouldTrace("subpaving") {
			wantFirst = 2
		}
		if !errors.Is(err, printerErr) || retained == nil || len(c.clauses) != 2 || first.refs != wantFirst || second.refs != 2 || !strings.Contains(output.String(), "new clause:\n") {
			t.Fatal("HWF clause TRACE stopped after ownership", err, first.refs, second.refs, output.String())
		}
	} else if err != nil || retained == nil || len(c.clauses) != 2 {
		t.Fatal("native gate changed HWF clause publication", err)
	}
	if err := c.SetVariableDisplay(nil); err != nil {
		t.Fatal(err)
	}
}
