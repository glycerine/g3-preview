package subpaving

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryBoundTracePublicationAndConflictOrder(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	c := f.context
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_mk_bound")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(root, f.x, &f.five, true, false, false); err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(root, f.x, &f.four, false, false, false); err != nil {
		t.Fatal(err)
	}
	if root.conflictVar != f.x || c.numConflicts != 1 || c.timestamp != 2 || len(c.queue) != 2 {
		t.Fatal("source MPF bound conflict publication", root.conflictVar, c.timestamp, len(c.queue))
	}
	if tags.ShouldTrace("subpaving_mk_bound") {
		got := output.String()
		for _, part := range []string{"subpaving_t_def.h:563", "mk_bound: 5 <= x0\ntimestamp: 0\n", "mk_bound: x0 <= 4\ntimestamp: 1\n", "subpaving_t_def.h:565", "conflict\n5 <= x0 x0 <= 4\n"} {
			if !strings.Contains(got, part) {
				t.Fatalf("missing MPF bound TRACE %q in %q", part, got)
			}
		}
		if output.flushes != 3 || strings.Index(got, "mk_bound: x0 <= 4") > strings.Index(got, "conflict\n") {
			t.Fatalf("source MPF bound TRACE order: %q, flushes=%d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased MPF bound TRACE emitted", output.String())
	}
}

func TestArbitraryBoundTraceStreamAndPrinterFailurePrefix(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	c := f.context
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("subpaving_mk_bound")
	if err := c.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(root, f.x, &f.four, true, false, false); err != nil || c.timestamp != 1 || len(c.queue) != 1 {
		t.Fatal("stream refusal changed MPF bound publication", err, c.timestamp)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_mk_bound")
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
	err = c.PublishBound(root, f.x, &f.five, false, false, false)
	if tags.ShouldTrace("subpaving_mk_bound") {
		if !errors.Is(err, printerErr) || root.trail == nil || root.trail.lower || c.timestamp != 1 || root.conflictVar != ^uint32(0) || len(c.queue) != 1 ||
			!strings.Contains(output.String(), "mk_bound: candidate") {
			t.Fatal("MPF trace stopped after bound push before conflict/queue", err, c.timestamp, len(c.queue), output.String())
		}
	} else if err != nil || c.timestamp != 2 || len(c.queue) != 2 {
		t.Fatal("native TRACE gate changed MPF publication", err, c.timestamp)
	}
}
