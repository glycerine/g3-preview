package subpaving

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func hardwareClausePropagationTraceFixture(t *testing.T) (*HardwareContext, *HardwareClause, *HardwareNode) {
	t.Helper()
	c, m, two, _ := hardwarePolynomialTraceFixture(t)
	var three numeral.HardwareFloat
	if err := m.SetInt32(&three, 3); err != nil {
		t.Fatal(err)
	}
	first, err := c.NewInequality(0, &three, true, false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.NewInequality(1, &two, true, false)
	if err != nil {
		t.Fatal(err)
	}
	clause, err := c.AddClause([]*HardwareInequality{second, first}, false, true)
	if err != nil {
		t.Fatal(err)
	}
	root, err := c.CreateRootNode()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(root, 0, &two, false, false); err != nil {
		t.Fatal(err)
	}
	return c, clause, root
}

func TestHardwareClausePropagationTraceSourceOrder(t *testing.T) {
	c, clause, root := hardwareClausePropagationTraceFixture(t)
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("propagate_clause")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateClause(clause, root); err != nil || root.lower[1] == nil || c.numVisited != 1 {
		t.Fatal("HWF clause propagation", err)
	}
	if tags.ShouldTrace("propagate_clause") {
		got := output.String()
		if !strings.Contains(got, "subpaving_t_def.h:1376") || !strings.Contains(got, "propagate using:\n3 <= x0 or 2 <= x1\n") ||
			!strings.Contains(got, "subpaving_t_def.h:1400") || !strings.Contains(got, "propagating inequality: 2 <= x1\n") ||
			strings.Index(got, "subpaving_t_def.h:1376") > strings.Index(got, "subpaving_t_def.h:1400") || output.flushes != 2 {
			t.Fatalf("HWF clause propagation TRACE: %q, flushes=%d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased HWF clause propagation TRACE emitted", output.String())
	}
}

func TestHardwareClausePropagationTraceRefusalAndPrinterPrefixes(t *testing.T) {
	c, clause, root := hardwareClausePropagationTraceFixture(t)
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("propagate_clause")
	if err := c.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateClause(clause, root); err != nil || root.lower[1] == nil {
		t.Fatal("borrowed stream refusal changed clause propagation", err)
	}

	for _, stopAt := range []int{1, 3} {
		d, otherClause, otherRoot := hardwareClausePropagationTraceFixture(t)
		var output bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		tags.Enable("propagate_clause")
		if err := d.BindTrace(tags); err != nil {
			t.Fatal(err)
		}
		printerErr := errors.New("borrowed variable printer stopped")
		calls := 0
		if err := d.SetVariableDisplay(func(w io.Writer, _ uint32) error {
			calls++
			_, _ = io.WriteString(w, "candidate")
			if calls == stopAt {
				return printerErr
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		err := d.PropagateClause(otherClause, otherRoot)
		if tags.ShouldTrace("propagate_clause") {
			wantVisited := uint64(0)
			if stopAt == 3 {
				wantVisited = 1
			}
			if !errors.Is(err, printerErr) || d.numVisited != wantVisited || otherRoot.lower[1] != nil || !strings.Contains(output.String(), "candidate") {
				t.Fatal("HWF TRACE printer prefix", stopAt, err, d.numVisited, output.String())
			}
		} else if err != nil || otherRoot.lower[1] == nil {
			t.Fatal("native gate changed HWF clause propagation", stopAt, err)
		}
		if err := d.SetVariableDisplay(nil); err != nil {
			t.Fatal(err)
		}
	}
}
