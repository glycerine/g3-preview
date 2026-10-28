package subpaving

// Source: src/math/subpaving/subpaving_t_def.h:1170-1175,1703-1737; config_mpf.

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func arbitraryMayPropagateClauseFixture(t *testing.T) (*ArbitraryContext, *ArbitraryClause, *ArbitraryNode) {
	t.Helper()
	f := newArbitraryClauseFixture(t)
	c, m, x, y := f.context, f.numbers, f.x, f.y
	var two numeral.ArbitraryFloat
	if err := m.SetInt32(&two, 2); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Del(&two); err != nil {
			t.Error(err)
		}
	})
	xLower, err := c.NewInequality(x, &f.five, true, false)
	if err != nil {
		t.Fatal(err)
	}
	yUpper, err := c.NewInequality(y, &two, false, false)
	if err != nil {
		t.Fatal(err)
	}
	clause, err := c.AddClause([]*ArbitraryInequality{yUpper, xLower})
	if err != nil {
		t.Fatal(err)
	}
	unit, err := c.NewInequality(x, &f.four, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.AddUnitClause(unit, true); err != nil {
		t.Fatal(err)
	}
	root, err := c.AssertRootUnits()
	if err != nil {
		t.Fatal(err)
	}
	return c, clause, root
}

func TestArbitraryMayPropagateClauseEqualTimestampAndSkippedVisitTrace(t *testing.T) {
	c, clause, root := arbitraryMayPropagateClauseFixture(t)
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("may_propagate_bug")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateQueued(root); err != nil || root.upper[1] == nil || c.numVisited != 1 {
		t.Fatal("equal timestamp failed to propagate clause", err, c.numVisited)
	}
	if tags.ShouldTrace("may_propagate_bug") {
		got := output.String()
		if strings.Count(got, "subpaving_t_def.h:1172") != 2 ||
			!strings.Contains(got, "x0 <= 4 | 5 <= x0 or x1 <= 2\nresult: 0, 0, 0\n") ||
			!strings.Contains(got, "x1 <= 2 | 5 <= x0 or x1 <= 2\nresult: 0, 1, 2\n") ||
			output.flushes != 2 || clause.visited != 2 {
			t.Fatalf("strict diagnostic versus >= predicate: %q, flushes=%d, visited=%d", got, output.flushes, clause.visited)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased may_propagate TRACE emitted", output.String())
	}
}

func TestArbitraryMayPropagateDefinitionSkippedTimestampTrace(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	c := f.context
	defined, err := c.DefinePolynomial(&f.five, []numeral.ArbitraryFloat{f.four}, []uint32{f.x})
	if err != nil {
		t.Fatal(err)
	}
	c.polynomials[defined].visited = 3
	unit, err := c.NewInequality(f.x, &f.four, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.AddUnitClause(unit, true); err != nil {
		t.Fatal(err)
	}
	root, err := c.AssertRootUnits()
	if err != nil {
		t.Fatal(err)
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("may_propagate_bug")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateQueued(root); err != nil || root.lower[defined] != nil || c.numVisited != 0 {
		t.Fatal("newer definition should be skipped", err, c.numVisited)
	}
	if tags.ShouldTrace("may_propagate_bug") {
		if got := output.String(); !strings.Contains(got, "4 <= x0 | 5 + 4 x0\nresult: 0, 0, 3\n") || output.flushes != 1 {
			t.Fatalf("source definition timestamp TRACE: %q, flushes=%d", got, output.flushes)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased definition TRACE emitted", output.String())
	}
}

func TestArbitraryMayPropagateTracePrinterFailureAfterQueueConsumption(t *testing.T) {
	c, clause, root := arbitraryMayPropagateClauseFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("may_propagate_bug")
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
	err := c.PropagateQueued(root)
	if tags.ShouldTrace("may_propagate_bug") {
		if !errors.Is(err, printerErr) || c.queueHead != 1 || clause.visited != 0 || root.upper[1] != nil ||
			!strings.Contains(output.String(), "candidate") {
			t.Fatal("may_propagate diagnostic stopped before clause visit", err, c.queueHead, clause.visited, output.String())
		}
	} else if err != nil || root.upper[1] == nil {
		t.Fatal("native TRACE gate changed propagation", err)
	}
}
