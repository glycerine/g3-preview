package subpaving

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestHardwareContextRelevantNewBoundSourceBranches(t *testing.T) {
	makeContext := func(epsilon, maxPower uint32) (*HardwareContext, *numeral.HardwareFloatNumeralManager, *HardwareNode) {
		t.Helper()
		m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
		if err != nil {
			t.Fatal(err)
		}
		c, err := NewHardwareContext(m, &control.Reslimit{Limit: 1000}, epsilon, maxPower, 8192)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.NewVariable(false); err != nil {
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
		return c, m, n
	}
	value := func(m *numeral.HardwareFloatNumeralManager, f float64) *numeral.HardwareFloat {
		t.Helper()
		v := new(numeral.HardwareFloat)
		if err := m.SetFloat64(v, f); err != nil {
			t.Fatal(err)
		}
		return v
	}
	check := func(c *HardwareContext, n *HardwareNode, v *numeral.HardwareFloat, lower, open, want bool, label string) {
		t.Helper()
		got, err := c.RelevantNewBound(n, 0, v, lower, open)
		if err != nil || got != want {
			t.Fatalf("%s: relevant=%v want=%v err=%v", label, got, want, err)
		}
	}
	t.Run("zero-epsilon-and-max-bound", func(t *testing.T) {
		c, m, n := makeContext(0, 1)
		check(c, n, value(m, 11), true, false, false, "lower beyond max")
		check(c, n, value(m, 10), true, false, true, "lower at max")
		check(c, n, value(m, -11), false, false, false, "upper beyond negative max")
		check(c, n, value(m, -10), false, false, true, "upper at negative max")
		five := value(m, 5)
		if err := c.PropagateBound(n, 0, five, true, false); err != nil {
			t.Fatal(err)
		}
		check(c, n, five, true, false, false, "closed equal lower")
		check(c, n, five, true, true, true, "open equal lower")
		check(c, n, value(m, 4), true, false, false, "weaker lower")
		check(c, n, value(m, 6), true, false, true, "stronger lower")
		if err := c.PropagateBound(n, 0, value(m, 7), false, false); err != nil {
			t.Fatal(err)
		}
		check(c, n, value(m, 8), true, false, true, "conflict before max")
		check(c, n, value(m, 4), false, false, true, "reverse conflict")
		check(c, n, value(m, 7), false, false, false, "closed equal upper")
		check(c, n, value(m, 7), false, true, true, "open equal upper")
	})
	t.Run("epsilon-gap-and-candidate-publication", func(t *testing.T) {
		c, m, n := makeContext(20, 2)
		if err := c.PropagateBound(n, 0, value(m, 10), true, false); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateBound(n, 0, value(m, 20), false, false); err != nil {
			t.Fatal(err)
		}
		check(c, n, value(m, 10.25), true, false, false, "lower inside gap")
		check(c, n, value(m, 10.75), true, false, true, "lower beyond gap")
		check(c, n, value(m, 19.75), false, false, false, "upper inside gap")
		check(c, n, value(m, 19.25), false, false, true, "upper beyond gap")
		check(c, n, value(m, 20), true, true, true, "open equal conflict")
		before := len(c.queue)
		published, err := c.PropagateCandidate(n, 0, value(m, 10.25), true, false)
		if err != nil || published || len(c.queue) != before {
			t.Fatal("irrelevant candidate published", published, err)
		}
		published, err = c.PropagateCandidate(n, 0, value(m, 10.75), true, false)
		if err != nil || !published || len(c.queue) != before+1 || m.ToFloat64(&n.lower[0].value) != 10.75 {
			t.Fatal("relevant candidate publication", published, err)
		}
	})
	t.Run("arithmetic-failure", func(t *testing.T) {
		c, m, n := makeContext(1, 1)
		if err := c.PropagateBound(n, 0, value(m, -1e308), true, false); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateBound(n, 0, value(m, 1e308), false, false); err != nil {
			t.Fatal(err)
		}
		check(c, n, value(m, 0), true, false, false, "overflowing interval width")
		if !c.arithmeticFailed || len(c.queue) != 2 {
			t.Fatal("arithmetic failure must refuse without publishing")
		}
	})
}
