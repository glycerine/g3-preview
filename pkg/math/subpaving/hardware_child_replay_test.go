package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestHardwareChildRecentBoundReplayAndPolynomialWatcher(t *testing.T) {
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewHardwareContext(m, &control.Reslimit{Limit: 1000}, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	x, err := c.NewVariable(false)
	if err != nil {
		t.Fatal(err)
	}
	values := make([]numeral.HardwareFloat, 5)
	for i := range values {
		if err := m.SetInt32(&values[i], int32(i)); err != nil {
			t.Fatal(err)
		}
	}
	y, err := c.DefinePolynomial(&values[0], []HardwarePolynomialTerm{{Variable: x, Coefficient: &values[2]}})
	if err != nil {
		t.Fatal(err)
	}
	for _, unit := range []struct {
		value *numeral.HardwareFloat
		lower bool
	}{{&values[0], true}, {&values[4], false}} {
		a, err := c.NewInequality(x, unit.value, unit.lower, false)
		if err != nil {
			t.Fatal(err)
		}
		if err := c.AddUnitClause(a, true); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.InitializeRoot(); err != nil {
		t.Fatal(err)
	}
	root := c.root
	child, err := c.NewChildNode(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := child.Close(); err != nil {
			t.Error(err)
		}
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := c.DecideBound(child, x, &values[1], true, false); err != nil {
		t.Fatal(err)
	}
	stale := child.trail
	if err := c.DecideBound(child, x, &values[2], true, false); err != nil {
		t.Fatal(err)
	}
	currentLower := child.trail
	if err := c.DecideBound(child, x, &values[3], false, false); err != nil {
		t.Fatal(err)
	}
	currentUpper := child.trail
	if len(c.queue) != 0 || m.ToFloat64(&root.lower[x].value) != 0 || m.ToFloat64(&root.upper[x].value) != 4 {
		t.Fatal("decisions must stay child-local and unqueued")
	}
	before := c.timestamp
	if err := c.AddRecentBounds(root); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("root cannot replay child bounds", err)
	}
	if err := c.AddRecentBounds(child); err != nil {
		t.Fatal(err)
	}
	if len(c.queue) != 2 || c.queue[0] != currentUpper || c.queue[1] != currentLower ||
		currentUpper.timestamp != before || currentLower.timestamp != before || stale.timestamp == before || c.timestamp != before {
		t.Fatal("newest-first replay must skip superseded lower decision without advancing timestamp")
	}
	if err := c.AddRecentBounds(child); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("replay requires empty queue", err)
	}
	if err := c.PropagateQueued(child); err != nil {
		t.Fatal(err)
	}
	if m.ToFloat64(&child.lower[y].value) != 4 || m.ToFloat64(&child.upper[y].value) != 6 ||
		m.ToFloat64(&root.lower[y].value) != 0 || m.ToFloat64(&root.upper[y].value) != 8 ||
		len(c.queue) != 0 || c.queueHead != 0 {
		t.Fatal("child decisions must update watched polynomial without changing parent")
	}
}
