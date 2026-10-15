// Source: src/math/subpaving/subpaving_t_def.h:243-246; config_hwf.
package subpaving

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func hardwareSplitTraceFixture(t *testing.T) (*HardwareContext, *HardwareNode, uint32) {
	t.Helper()
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewHardwareContext(m, &control.Reslimit{Limit: 1000}, 0, 1, 8192)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	x, err := c.NewVariable(false)
	if err != nil {
		t.Fatal(err)
	}
	root, err := c.CreateRootNode()
	if err != nil {
		t.Fatal(err)
	}
	var five numeral.HardwareFloat
	if err := m.SetInt32(&five, 5); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(root, x, &five, false, false); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateQueued(root); err != nil {
		t.Fatal(err)
	}
	return c, root, x
}

func closeHardwareSplitChildren(t *testing.T, left, right *HardwareNode) {
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

func TestHardwareMidpointSplitSourceBoundsTrace(t *testing.T) {
	c, root, x := hardwareSplitTraceFixture(t)
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_int_split")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	left, right, err := c.SplitMidpoint(root, x, true, 2)
	closeHardwareSplitChildren(t, left, right)
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
		t.Fatalf("source HWF child-bound trace: %q, flushes=%d", got, output.flushes)
	}
}

func TestHardwareMidpointSplitTraceRefusalKeepsChildren(t *testing.T) {
	c, root, x := hardwareSplitTraceFixture(t)
	tags := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	tags.Enable("subpaving_int_split")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	left, right, err := c.SplitMidpoint(root, x, true, 2)
	closeHardwareSplitChildren(t, left, right)
	if err != nil || left == nil || right == nil || left.upper[x] == nil || right.lower[x] == nil || root.children != 2 {
		t.Fatal("borrowed HWF split stream changed child publication", err)
	}
}

func TestHardwareMidpointSplitTracePrinterFailureKeepsChildren(t *testing.T) {
	c, root, x := hardwareSplitTraceFixture(t)
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
	closeHardwareSplitChildren(t, left, right)
	if left == nil || right == nil || root.children != 2 {
		t.Fatal("split children preceded HWF trace", err)
	}
	if tags.ShouldTrace("subpaving_int_split") {
		if !errors.Is(err, refusal) || !strings.Contains(output.String(), "LEFT:\n") || strings.Contains(output.String(), "RIGHT:\n") || output.flushes != 0 {
			t.Fatalf("HWF printer failure crossed source trace boundary: %q, flushes=%d, err=%v", output.String(), output.flushes, err)
		}
	} else if err != nil || output.Len() != 0 {
		t.Fatal("erased trace ran HWF variable printer", err, output.String())
	}
}
