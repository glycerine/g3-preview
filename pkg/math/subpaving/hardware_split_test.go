package subpaving

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestHardwareMidpointSplitterRetainedBranchesAndRefusal(t *testing.T) {
	makeContext := func(t *testing.T, integer bool) (*HardwareContext, *numeral.HardwareFloatNumeralManager, *HardwareNode, uint32) {
		t.Helper()
		m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
		if err != nil {
			t.Fatal(err)
		}
		c, err := NewHardwareContext(m, &control.Reslimit{Limit: 1000}, 0, 1, 8192)
		if err != nil {
			t.Fatal(err)
		}
		x, err := c.NewVariable(integer)
		if err != nil {
			t.Fatal(err)
		}
		n, err := c.CreateRootNode()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := c.Close(); err != nil {
				t.Error(err)
			}
		})
		return c, m, n, x
	}
	for _, tc := range []struct {
		name                        string
		integer                     bool
		lower, upper                *float64
		delta                       uint32
		wantLeft, wantRight         float64
		wantLeftOpen, wantRightOpen bool
	}{
		{name: "unbounded", delta: 1, wantLeft: 0, wantRight: 0, wantLeftOpen: true},
		{name: "upper-only", upper: floatPointer(5), delta: 2, wantLeft: 3, wantRight: 3, wantLeftOpen: true},
		{name: "lower-only", lower: floatPointer(5), delta: 2, wantLeft: 7, wantRight: 7, wantLeftOpen: true},
		{name: "finite", lower: floatPointer(0), upper: floatPointer(4), delta: 1, wantLeft: 2, wantRight: 2, wantLeftOpen: true},
		{name: "integer-finite", integer: true, lower: floatPointer(0), upper: floatPointer(3), delta: 1, wantLeft: 1, wantRight: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, m, root, x := makeContext(t, tc.integer)
			for _, b := range []struct {
				value *float64
				lower bool
			}{{tc.lower, true}, {tc.upper, false}} {
				if b.value == nil {
					continue
				}
				var v numeral.HardwareFloat
				if err := m.SetFloat64(&v, *b.value); err != nil {
					t.Fatal(err)
				}
				if err := c.PropagateBound(root, x, &v, b.lower, false); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.PropagateQueued(root); err != nil {
				t.Fatal(err)
			}
			left, right, err := c.SplitMidpoint(root, x, true, tc.delta)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := right.Close(); err != nil {
					t.Error(err)
				}
				if err := left.Close(); err != nil {
					t.Error(err)
				}
			})
			if left.id != 1 || right.id != 2 || left.upper[x] == nil || right.lower[x] == nil ||
				m.ToFloat64(&left.upper[x].value) != tc.wantLeft || m.ToFloat64(&right.lower[x].value) != tc.wantRight ||
				left.upper[x].open != tc.wantLeftOpen || right.lower[x].open != tc.wantRightOpen ||
				left.upper[x].reason != hardwareReasonAxiom || right.lower[x].reason != hardwareReasonAxiom || len(c.queue) != 0 {
				t.Fatal("source midpoint child decisions")
			}
		})
	}
	t.Run("adjacent-hardware-values", func(t *testing.T) {
		c, m, root, x := makeContext(t, false)
		var lower, upper numeral.HardwareFloat
		if err := m.SetFloat64(&lower, 1); err != nil {
			t.Fatal(err)
		}
		if err := m.SetFloat64(&upper, math.Nextafter(1, 2)); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateBound(root, x, &lower, true, false); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateBound(root, x, &upper, false, false); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateQueued(root); err != nil {
			t.Fatal(err)
		}
		left, right, err := c.SplitMidpoint(root, x, true, 1)
		if !errors.Is(err, ErrHardwareSplitNoInterior) || !errors.Is(err, ErrSubpavingFailure) || left == nil || right == nil ||
			left.trail != root.trail || right.trail != root.trail || root.children != 2 {
			t.Fatal("source creates both children before refusing non-interior midpoint", err)
		}
		if err := right.Close(); err != nil {
			t.Fatal(err)
		}
		if err := left.Close(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("opposite-open-flags", func(t *testing.T) {
		c, m, root, x := makeContext(t, false)
		left, right, err := c.SplitMidpoint(root, x, false, 1)
		if err != nil {
			t.Fatal(err)
		}
		if left.upper[x].open || !right.lower[x].open || m.ToFloat64(&left.upper[x].value) != 0 {
			t.Fatal("left and right source openness must be complementary")
		}
		if err := right.Close(); err != nil {
			t.Fatal(err)
		}
		if err := left.Close(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("overflow-after-child-creation", func(t *testing.T) {
		c, m, root, x := makeContext(t, false)
		var lower, upper numeral.HardwareFloat
		for _, entry := range []struct {
			out   *numeral.HardwareFloat
			value float64
		}{{&lower, 1e308}, {&upper, 1.7e308}} {
			if err := m.SetFloat64(entry.out, entry.value); err != nil {
				t.Fatal(err)
			}
		}
		if err := c.PropagateBound(root, x, &lower, true, false); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateBound(root, x, &upper, false, false); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateQueued(root); err != nil {
			t.Fatal(err)
		}
		left, right, err := c.SplitMidpoint(root, x, true, 1)
		if !errors.Is(err, numeral.ErrHardwareFloatNonRegular) || left == nil || right == nil || root.children != 2 {
			t.Fatal("source creates children before overflowing midpoint addition", err)
		}
		if err := right.Close(); err != nil {
			t.Fatal(err)
		}
		if err := left.Close(); err != nil {
			t.Fatal(err)
		}
	})
}

func floatPointer(value float64) *float64 { return &value }
