package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestArbitraryChildRecentBoundReplayAndPolynomialWatcher(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	c := f.context
	y, err := c.DefinePolynomial(&f.five, []numeral.ArbitraryFloat{f.four}, []uint32{f.x})
	if err != nil {
		t.Fatal(err)
	}
	makeValue := func(value int32) numeral.ArbitraryFloat {
		t.Helper()
		var v numeral.ArbitraryFloat
		if err := f.numbers.SetInt32(&v, value); err != nil {
			t.Fatal(err)
		}
		return v
	}
	for _, item := range []struct {
		value int32
		lower bool
	}{{1, true}, {4, false}} {
		v := makeValue(item.value)
		a, err := c.NewInequality(f.x, &v, item.lower, false)
		if err != nil {
			t.Fatal(err)
		}
		if err := c.AddUnitClause(a, true); err != nil {
			t.Fatal(err)
		}
		if err := f.numbers.Del(&v); err != nil {
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
	if child.parent != root || child.depth != 1 || child.id != 1 || child.trail != root.trail ||
		child.lower[f.x] != root.lower[f.x] || child.upper[y] != root.upper[y] {
		t.Fatal("child must borrow parent trail and copy bound views")
	}
	decide := func(value int32, lower bool) *ArbitraryBound {
		t.Helper()
		v := makeValue(value)
		if err := c.DecideBound(child, f.x, &v, lower, false); err != nil {
			t.Fatal(err)
		}
		if err := f.numbers.Del(&v); err != nil {
			t.Fatal(err)
		}
		return child.trail
	}
	stale := decide(2, true)
	currentLower := decide(3, true)
	currentUpper := decide(3, false)
	if len(c.queue) != 0 || child.lower[f.x] != currentLower || root.lower[f.x] == child.lower[f.x] || root.upper[f.x] == child.upper[f.x] {
		t.Fatal("decisions must remain child-local and unqueued")
	}
	before := c.timestamp
	if err := c.AddRecentBounds(root); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("root has no parent-local bounds to replay", err)
	}
	if err := c.AddRecentBounds(child); err != nil {
		t.Fatal(err)
	}
	if len(c.queue) != 2 || c.queue[0] != currentUpper || c.queue[1] != currentLower ||
		currentUpper.timestamp != before || currentLower.timestamp != before || stale.timestamp == before || c.timestamp != before {
		t.Fatal("newest-first replay must skip superseded lower bound without advancing timestamp")
	}
	if err := c.AddRecentBounds(child); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("replay requires an empty queue", err)
	}
	if err := c.PropagateQueued(child); err != nil {
		t.Fatal(err)
	}
	core, err := f.numbers.BorrowCore()
	if err != nil {
		t.Fatal(err)
	}
	value := func(b *ArbitraryBound) float64 {
		t.Helper()
		if b == nil {
			t.Fatal("missing polynomial bound")
		}
		v, err := core.ToDouble(&b.value)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	if value(child.lower[y]) != 12 || value(child.upper[y]) != 12 ||
		value(root.lower[y]) != 4 || value(root.upper[y]) != 16 || len(c.queue) != 0 || c.queueHead != 0 {
		t.Fatal("replayed decisions must update child polynomial bounds without changing parent")
	}
}
