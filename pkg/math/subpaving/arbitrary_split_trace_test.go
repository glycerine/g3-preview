package subpaving

// Source: src/math/subpaving/subpaving_t_def.h:243-246; config_mpf.

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func arbitrarySplitTraceFixture(t *testing.T) (*ArbitraryContext, *ArbitraryNode, uint32) {
	t.Helper()
	f := newArbitraryClauseFixture(t)
	root, err := f.context.Root()
	if err != nil {
		t.Fatal(err)
	}
	if err := f.context.PublishBound(root, f.x, &f.five, false, false, true); err != nil {
		t.Fatal(err)
	}
	if err := f.context.PropagateQueued(root); err != nil {
		t.Fatal(err)
	}
	return f.context, root, f.x
}

func TestArbitraryMidpointSplitSourceBoundsTrace(t *testing.T) {
	c, root, x := arbitrarySplitTraceFixture(t)
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_int_split")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	left, right, err := c.SplitMidpoint(root, x, true, 2)
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
	if strings.Count(got, "subpaving_t_def.h:243") != 1 || !strings.Contains(got, "LEFT:\nx0 < 3\n\nRIGHT:\n3 <= x0 x0 <= 5\n") || output.flushes != 1 {
		t.Fatalf("source MPF child-bound trace: %q, flushes=%d", got, output.flushes)
	}
}

func TestArbitraryMidpointSplitTraceRefusalKeepsChildren(t *testing.T) {
	c, root, x := arbitrarySplitTraceFixture(t)
	tags := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	tags.Enable("subpaving_int_split")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	left, right, err := c.SplitMidpoint(root, x, true, 2)
	if err != nil || left == nil || right == nil || left.upper[x] == nil || right.lower[x] == nil || root.children != 2 {
		t.Fatal("borrowed MPF split stream changed child publication", err)
	}
}

func TestArbitraryMidpointSplitTracePrinterFailureKeepsChildren(t *testing.T) {
	c, root, x := arbitrarySplitTraceFixture(t)
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
	left, right, err := c.SplitMidpoint(root, x, true, 2)
	if left == nil || right == nil || root.children != 2 {
		t.Fatal("split children preceded MPF trace", err)
	}
	if tags.ShouldTrace("subpaving_int_split") {
		if !errors.Is(err, refusal) || !strings.Contains(output.String(), "LEFT:\n") || strings.Contains(output.String(), "RIGHT:\n") || output.flushes != 0 {
			t.Fatalf("MPF printer failure crossed source trace boundary: %q, flushes=%d, err=%v", output.String(), output.flushes, err)
		}
	} else if err != nil || output.Len() != 0 {
		t.Fatal("erased trace ran MPF variable printer", err, output.String())
	}
}
