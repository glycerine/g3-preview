package subpaving

// Source: src/math/subpaving/subpaving_t_def.h:243-246; config_mpff.

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func fastSplitTraceFixture(t *testing.T) (*FastContext, *FastNode) {
	t.Helper()
	m, c, value, root := fastMonomialFixture(t, []bool{false})
	if err := m.SetInt32(value, 5); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(root, 0, value, false, false); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateQueued(root); err != nil {
		t.Fatal(err)
	}
	return c, root
}

func closeFastSplitChildren(t *testing.T, left, right *FastNode) {
	t.Helper()
	t.Cleanup(func() {
		if right != nil {
			if err := right.Close(); err != nil {
				t.Error(err)
			}
		}
		if left != nil {
			if err := left.Close(); err != nil {
				t.Error(err)
			}
		}
	})
}

func TestFastMidpointSplitSourceBoundsTrace(t *testing.T) {
	c, root := fastSplitTraceFixture(t)
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_int_split")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	left, right, err := c.SplitMidpoint(root, 0, true, 2)
	closeFastSplitChildren(t, left, right)
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
		t.Fatalf("source MPFF child-bound trace: %q, flushes=%d", got, output.flushes)
	}
}

func TestFastMidpointSplitTraceRefusalKeepsChildren(t *testing.T) {
	c, root := fastSplitTraceFixture(t)
	tags := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	tags.Enable("subpaving_int_split")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	left, right, err := c.SplitMidpoint(root, 0, true, 2)
	closeFastSplitChildren(t, left, right)
	if err != nil || left == nil || right == nil || left.upper[0] == nil || right.lower[0] == nil || root.children != 2 {
		t.Fatal("borrowed MPFF split stream changed child publication", err)
	}
}

func TestFastMidpointSplitTracePrinterFailureKeepsChildren(t *testing.T) {
	c, root := fastSplitTraceFixture(t)
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
	left, right, err := c.SplitMidpoint(root, 0, true, 2)
	closeFastSplitChildren(t, left, right)
	if left == nil || right == nil || root.children != 2 {
		t.Fatal("split children preceded MPFF trace", err)
	}
	if tags.ShouldTrace("subpaving_int_split") {
		if !errors.Is(err, refusal) || !strings.Contains(output.String(), "LEFT:\n") || strings.Contains(output.String(), "RIGHT:\n") || output.flushes != 0 {
			t.Fatalf("MPFF printer failure crossed source trace boundary: %q, flushes=%d, err=%v", output.String(), output.flushes, err)
		}
	} else if err != nil || output.Len() != 0 {
		t.Fatal("erased trace ran MPFF variable printer", err, output.String())
	}
}
