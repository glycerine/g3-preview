package subpaving

// Source: src/math/subpaving/subpaving_t_def.h:243-246; config_mpfx.

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func fixedSplitTraceFixture(t *testing.T) (*FixedContext, *FixedNode) {
	t.Helper()
	m, err := numeral.NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	value, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewFixedContext(m, &control.Reslimit{Limit: 1000}, []bool{false}, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, e := range []error{c.Close(), value.Close(), m.Close()} {
			if e != nil {
				t.Error(e)
			}
		}
	})
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt64(value, 10); err != nil {
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

func closeFixedSplitChildren(t *testing.T, left, right *FixedNode) {
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

func TestFixedMidpointSplitSourceBoundsTrace(t *testing.T) {
	c, root := fixedSplitTraceFixture(t)
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_int_split")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	left, right, err := c.SplitMidpoint(root, 0, true, 2)
	closeFixedSplitChildren(t, left, right)
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
	if strings.Count(got, "subpaving_t_def.h:243") != 1 || !strings.Contains(got, "LEFT:\nx0 < 8\n\nRIGHT:\n8 <= x0 x0 <= 10\n") || output.flushes != 1 {
		t.Fatalf("source MPFX child-bound trace: %q, flushes=%d", got, output.flushes)
	}
}

func TestFixedMidpointSplitTraceRefusalKeepsChildren(t *testing.T) {
	c, root := fixedSplitTraceFixture(t)
	tags := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	tags.Enable("subpaving_int_split")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	left, right, err := c.SplitMidpoint(root, 0, true, 2)
	closeFixedSplitChildren(t, left, right)
	if err != nil || left == nil || right == nil || left.upper[0] == nil || right.lower[0] == nil || root.children != 2 {
		t.Fatal("borrowed MPFX split stream changed child publication", err)
	}
}

func TestFixedMidpointSplitTracePrinterFailureKeepsChildren(t *testing.T) {
	c, root := fixedSplitTraceFixture(t)
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
	closeFixedSplitChildren(t, left, right)
	if left == nil || right == nil || root.children != 2 {
		t.Fatal("split children preceded MPFX trace", err)
	}
	if tags.ShouldTrace("subpaving_int_split") {
		if !errors.Is(err, refusal) || !strings.Contains(output.String(), "LEFT:\n") || strings.Contains(output.String(), "RIGHT:\n") || output.flushes != 0 {
			t.Fatalf("MPFX printer failure crossed source trace boundary: %q, flushes=%d, err=%v", output.String(), output.flushes, err)
		}
	} else if err != nil || output.Len() != 0 {
		t.Fatal("erased trace ran MPFX variable printer", err, output.String())
	}
}
