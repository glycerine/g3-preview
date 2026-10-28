package subpaving

// Source: src/math/subpaving/subpaving_t_def.h:243-246; config_mpq.

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestRationalMidpointSplitSourceBoundsTrace(t *testing.T) {
	c, _, _, x, _ := rationalClauseFixture(t)
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_int_split")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	left, right, err := c.SplitMidpoint(root, x, true, 1)
	if err != nil || left == nil || right == nil {
		t.Fatal("source split before trace", err)
	}
	if !tags.ShouldTrace("subpaving_int_split") {
		if output.Len() != 0 || output.flushes != 0 {
			t.Fatal("erased split trace emitted", output.String())
		}
		return
	}
	got := output.String()
	if strings.Count(got, "subpaving_t_def.h:243") != 1 || !strings.Contains(got, "LEFT:\nx0 < 0\n\nRIGHT:\n0 <= x0 \n") || output.flushes != 1 {
		t.Fatalf("source child-bound trace: %q, flushes=%d", got, output.flushes)
	}
}

func TestRationalMidpointSplitTraceRefusalKeepsChildren(t *testing.T) {
	c, _, _, x, _ := rationalClauseFixture(t)
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	tags := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	tags.Enable("subpaving_int_split")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	left, right, err := c.SplitMidpoint(root, x, true, 1)
	if err != nil || left == nil || right == nil || left.upper[x] == nil || right.lower[x] == nil || root.children != 2 {
		t.Fatal("borrowed split trace stream changed child publication", err)
	}
}

func TestRationalMidpointSplitTracePrinterFailureKeepsChildren(t *testing.T) {
	c, _, _, x, _ := rationalClauseFixture(t)
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_int_split")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	refusal := errors.New("variable printer refused")
	if err := c.SetVariableDisplay(func(io.Writer, uint32) error { return refusal }); err != nil {
		t.Fatal(err)
	}
	left, right, err := c.SplitMidpoint(root, x, true, 1)
	if left == nil || right == nil || root.children != 2 {
		t.Fatal("split child construction preceded trace", err)
	}
	if tags.ShouldTrace("subpaving_int_split") {
		if !errors.Is(err, refusal) || !strings.Contains(output.String(), "LEFT:\n") || strings.Contains(output.String(), "RIGHT:\n") || output.flushes != 0 {
			t.Fatalf("printer failure crossed source trace boundary: %q, flushes=%d, err=%v", output.String(), output.flushes, err)
		}
	} else if err != nil || output.Len() != 0 {
		t.Fatal("erased trace ran variable printer", err, output.String())
	}
}
