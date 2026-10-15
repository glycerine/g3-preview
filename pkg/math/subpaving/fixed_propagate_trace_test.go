// Source: src/math/subpaving/subpaving_t_def.h:1703,1740-1751; config_mpfx.
package subpaving

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestFixedQueuedBoundPropagationTraceAndConsumedPrefix(t *testing.T) {
	m, c, v, root := fixedClauseFixture(t, []bool{false})
	for _, value := range []int64{3, 4} {
		if err := m.SetInt64(v, value); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateBound(root, 0, v, true, false); err != nil {
			t.Fatal(err)
		}
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_propagate")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateQueued(root); err != nil || len(c.queue) != 0 || c.queueHead != 0 {
		t.Fatal("queued propagation", err, len(c.queue), c.queueHead)
	}
	if tags.ShouldTrace("subpaving_propagate") {
		got := output.String()
		if strings.Count(got, "subpaving_t_def.h:1703") != 2 ||
			!strings.Contains(got, "propagate: 3 <= x0, timestamp: 0\n") ||
			!strings.Contains(got, "propagate: 4 <= x0, timestamp: 1\n") ||
			strings.Index(got, "timestamp: 0") > strings.Index(got, "timestamp: 1") || output.flushes != 2 {
			t.Fatalf("source queue TRACE: %q, flushes=%d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased queue TRACE emitted", output.String())
	}
}

func TestFixedQueuedBoundPropagationTraceRefusalAndPrinterStop(t *testing.T) {
	m, c, v, root := fixedClauseFixture(t, []bool{false})
	if err := m.SetInt64(v, 5); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(root, 0, v, true, false); err != nil {
		t.Fatal(err)
	}
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("subpaving_propagate")
	if err := c.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateQueued(root); err != nil || len(c.queue) != 0 {
		t.Fatal("borrowed trace refusal changed propagation", err)
	}

	dm, d, dv, otherRoot := fixedClauseFixture(t, []bool{false})
	if err := dm.SetInt64(dv, 6); err != nil {
		t.Fatal(err)
	}
	if err := d.PropagateBound(otherRoot, 0, dv, true, false); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_propagate")
	if err := d.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	printerErr := errors.New("borrowed variable printer stopped")
	if err := d.SetVariableDisplay(func(w io.Writer, variable uint32) error {
		_, _ = io.WriteString(w, "candidate")
		return printerErr
	}); err != nil {
		t.Fatal(err)
	}
	err := d.PropagateQueued(otherRoot)
	if tags.ShouldTrace("subpaving_propagate") {
		if !errors.Is(err, printerErr) || d.queueHead != 1 || len(d.queue) != 1 ||
			!strings.Contains(output.String(), "propagate: 6 <= candidate") {
			t.Fatal("TRACE stopped after dequeue before watchers", err, d.queueHead, len(d.queue), output.String())
		}
		if err := d.SetVariableDisplay(nil); err != nil {
			t.Fatal(err)
		}
		before := output.String()
		if err := d.PropagateQueued(otherRoot); err != nil || d.queueHead != 0 || len(d.queue) != 0 || output.String() != before {
			t.Fatal("source consumed prefix replayed", err, d.queueHead, output.String())
		}
	} else if err != nil || len(d.queue) != 0 {
		t.Fatal("native TRACE gate changed propagation", err)
	}
}
