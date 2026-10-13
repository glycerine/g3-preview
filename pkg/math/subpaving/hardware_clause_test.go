package subpaving

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestHardwareClauseStableWatchersAndReferences(t *testing.T) {
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewHardwareContext(m, &control.Reslimit{Limit: 100}, 0, 1, 8192)
	if err != nil {
		t.Fatal(err)
	}
	x, err := c.NewVariable(false)
	if err != nil {
		t.Fatal(err)
	}
	y, err := c.NewVariable(false)
	if err != nil {
		t.Fatal(err)
	}
	var five, eight numeral.HardwareFloat
	if err := m.SetInt32(&five, 5); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(&eight, 8); err != nil {
		t.Fatal(err)
	}
	ax, err := c.NewInequality(x, &five, true, false)
	if err != nil {
		t.Fatal(err)
	}
	ayLower, err := c.NewInequality(y, &five, true, false)
	if err != nil {
		t.Fatal(err)
	}
	ayUpper, err := c.NewInequality(y, &eight, false, false)
	if err != nil {
		t.Fatal(err)
	}
	clause, err := c.AddClause([]*HardwareInequality{ayUpper, ax, ayLower}, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(clause.atoms) != 3 || clause.atoms[0] != ax || clause.atoms[1] != ayUpper || clause.atoms[2] != ayLower ||
		len(c.watchOrder[x]) != 1 || len(c.watchOrder[y]) != 1 || ax.refs != 1 || ayLower.refs != 1 || ayUpper.refs != 1 {
		t.Fatal("stable variable sort, unique watchers, and one reference per atom")
	}
	single, err := c.AddClause([]*HardwareInequality{ax}, false, true)
	if err != nil || single != nil || len(c.unitClauses) != 1 || ax.refs != 2 {
		t.Fatal("singleton clause must become axiom unit", err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if ax.refs != 0 || ayLower.refs != 0 || ayUpper.refs != 0 || !clause.deleted {
		t.Fatal("context close must release unit and clause references")
	}
}

func TestHardwareClauseOpenTruthAndQueuedUnitPropagation(t *testing.T) {
	makeContext := func(t *testing.T) (*HardwareContext, *numeral.HardwareFloatNumeralManager, uint32, uint32) {
		t.Helper()
		m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
		if err != nil {
			t.Fatal(err)
		}
		c, err := NewHardwareContext(m, &control.Reslimit{Limit: 100}, 0, 1, 8192)
		if err != nil {
			t.Fatal(err)
		}
		x, err := c.NewVariable(false)
		if err != nil {
			t.Fatal(err)
		}
		y, err := c.NewVariable(false)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := c.Close(); err != nil {
				t.Error(err)
			}
		})
		return c, m, x, y
	}
	t.Run("open-equality-ties", func(t *testing.T) {
		c, m, x, _ := makeContext(t)
		var five numeral.HardwareFloat
		if err := m.SetInt32(&five, 5); err != nil {
			t.Fatal(err)
		}
		lowerClosed, _ := c.NewInequality(x, &five, true, false)
		lowerOpen, _ := c.NewInequality(x, &five, true, true)
		upperClosed, _ := c.NewInequality(x, &five, false, false)
		upperOpen, _ := c.NewInequality(x, &five, false, true)
		n, err := c.CreateRootNode()
		if err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateBound(n, x, &five, false, false); err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			a    *HardwareInequality
			want hardwareTruth
		}{{lowerClosed, hardwareUnknown}, {lowerOpen, hardwareFalse}, {upperClosed, hardwareTrue}, {upperOpen, hardwareUnknown}} {
			got, err := c.inequalityValue(tc.a, n)
			if err != nil || got != tc.want {
				t.Fatal("closed upper endpoint truth", got, tc.want, err)
			}
		}
		if err := c.PropagateBound(n, x, &five, false, true); err != nil {
			t.Fatal(err)
		}
		if got, err := c.inequalityValue(upperOpen, n); err != nil || got != hardwareTrue {
			t.Fatal("open upper endpoint makes strict upper atom true", got, err)
		}
	})
	t.Run("sole-unknown-through-queue", func(t *testing.T) {
		c, m, x, y := makeContext(t)
		var four, five numeral.HardwareFloat
		if err := m.SetInt32(&four, 4); err != nil {
			t.Fatal(err)
		}
		if err := m.SetInt32(&five, 5); err != nil {
			t.Fatal(err)
		}
		ax, _ := c.NewInequality(x, &five, true, false)
		ay, _ := c.NewInequality(y, &five, true, false)
		clause, err := c.AddClause([]*HardwareInequality{ay, ax}, false, true)
		if err != nil {
			t.Fatal(err)
		}
		n, err := c.CreateRootNode()
		if err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateClause(clause, n); err != nil || len(c.queue) != 0 {
			t.Fatal("two unknown atoms do not propagate", err)
		}
		if err := c.PropagateBound(n, x, &four, false, false); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateQueued(n); err != nil {
			t.Fatal(err)
		}
		if n.lower[y] == nil || m.ToFloat64(&n.lower[y].value) != 5 ||
			n.lower[y].reason != hardwareReasonClause || n.lower[y].reasonClause != clause ||
			len(c.queue) != 0 || n.conflictVar != ^uint32(0) {
			t.Fatal("sole unknown clause atom must publish its bound")
		}
	})
	t.Run("all-false-conflict", func(t *testing.T) {
		c, m, x, y := makeContext(t)
		var four, five numeral.HardwareFloat
		if err := m.SetInt32(&four, 4); err != nil {
			t.Fatal(err)
		}
		if err := m.SetInt32(&five, 5); err != nil {
			t.Fatal(err)
		}
		ax, _ := c.NewInequality(x, &five, true, false)
		ay, _ := c.NewInequality(y, &five, true, false)
		clause, err := c.AddClause([]*HardwareInequality{ay, ax}, false, true)
		if err != nil {
			t.Fatal(err)
		}
		n, err := c.CreateRootNode()
		if err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateBound(n, x, &four, false, false); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateBound(n, y, &four, false, false); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateQueued(n); err != nil {
			t.Fatal(err)
		}
		if n.conflictVar != x || n.lower[x] == nil || n.lower[x].reasonClause != clause || len(c.queue) != 0 {
			t.Fatal("all-false clause must publish first atom to expose conflict")
		}
	})
}
