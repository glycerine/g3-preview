package subpaving

// Source: src/math/subpaving/subpaving_t_def.h:802-868; config_mpfx.

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestFixedClauseAndUnitTracePublicationOrder(t *testing.T) {
	m, c, v, _ := fixedClauseFixture(t, []bool{false, false})
	lower := fixedClauseAtom(t, m, c, v, 0, 5, true, false)
	upper := fixedClauseAtom(t, m, c, v, 1, 5, false, true)
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
	clause, err := c.AddClause([]*FixedInequality{upper, lower}, false, true)
	if err != nil || clause == nil || clause.atoms[0] != lower || clause.atoms[1] != upper || lower.refs != 2 || upper.refs != 1 || len(c.unitClauses) != 1 {
		t.Fatal("source unit and sorted clause ownership", err)
	}
	if tags.ShouldTrace("subpaving") {
		got := output.String()
		if !strings.Contains(got, "subpaving_t_def.h:866") || !strings.Contains(got, "5 <= x0\n") ||
			!strings.Contains(got, "subpaving_t_def.h:841") || !strings.Contains(got, "new clause:\n5 <= x0 or x1 < 5\n") ||
			strings.Index(got, "subpaving_t_def.h:866") > strings.Index(got, "subpaving_t_def.h:841") || output.flushes != 2 {
			t.Fatalf("source unit/clause TRACE: %q, flushes=%d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased unit/clause TRACE emitted", output.String())
	}
}

func TestFixedClauseAndUnitTraceFailureBoundaries(t *testing.T) {
	m, c, v, _ := fixedClauseFixture(t, []bool{false, false})
	first := fixedClauseAtom(t, m, c, v, 0, 7, false, false)
	second := fixedClauseAtom(t, m, c, v, 1, 7, true, false)
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("subpaving")
	failed.Enable("subpaving_clause")
	if err := c.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if err := c.AddUnitClause(first, false); err != nil || first.refs != 1 || len(c.unitClauses) != 1 {
		t.Fatal("borrowed stream refusal changed unit", err)
	}
	clause, err := c.AddClause([]*FixedInequality{second, first}, false, true)
	if err != nil || clause == nil || first.refs != 2 || second.refs != 1 || len(c.clauses) != 1 {
		t.Fatal("borrowed stream refusal changed clause", err)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving")
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
	err = c.AddUnitClause(first, true)
	if tags.ShouldTrace("subpaving") {
		if !errors.Is(err, printerErr) || first.refs != 2 || len(c.unitClauses) != 1 || !strings.Contains(output.String(), "candidate") {
			t.Fatal("unit TRACE stopped before reference", err, first.refs, len(c.unitClauses), output.String())
		}
	} else if err != nil || first.refs != 3 || len(c.unitClauses) != 2 {
		t.Fatal("native gate changed unit publication", err)
	}
	retained, err := c.AddClause([]*FixedInequality{second, first}, false, true)
	if tags.ShouldTrace("subpaving_clause") {
		if !errors.Is(err, printerErr) || retained == nil || len(c.clauses) != 2 || first.refs != 3 || second.refs != 2 || !strings.Contains(output.String(), "new clause:\n") {
			t.Fatal("clause TRACE stopped after ownership", err, len(c.clauses), first.refs, second.refs, output.String())
		}
	} else if err != nil || retained == nil || len(c.clauses) != 2 {
		t.Fatal("native gate changed clause publication", err)
	}
}
