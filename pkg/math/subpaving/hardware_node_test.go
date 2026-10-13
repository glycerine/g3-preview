package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestHardwareNodeRetainedBoundViewsAndChildTrail(t *testing.T) {
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	root, err := NewHardwareNode(m, 2, nil, 7)
	if err != nil {
		t.Fatal(err)
	}
	var one, two numeral.HardwareFloat
	if err := m.SetInt32(&one, 1); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(&two, 2); err != nil {
		t.Fatal(err)
	}
	if err := root.PushBound(0, &one, true, false); err != nil {
		t.Fatal(err)
	}
	child, err := NewHardwareNode(m, 2, root, 8)
	if err != nil {
		t.Fatal(err)
	}
	if child.id != 8 || child.depth != 1 || child.trail != root.trail || child.lower[0] != root.lower[0] || child.upper[0] != nil {
		t.Fatal("child must inherit the source root's bound views and trail")
	}
	if err := root.Close(); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("live child must retain parent", err)
	}
	if err := child.PushBound(0, &two, true, true); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(&two, 9); err != nil {
		t.Fatal(err)
	}
	if child.lower[0] == root.lower[0] || child.trail.prev != root.trail ||
		m.ToFloat64(&child.lower[0].value) != 2 || m.ToFloat64(&root.lower[0].value) != 1 ||
		!child.lower[0].open || root.lower[0].open {
		t.Fatal("child replacement must preserve the parent view and copy its numeral")
	}
	if err := child.PushBound(2, &one, false, false); !errors.Is(err, arena.ErrInvalidSlot) || len(child.owned) != 1 {
		t.Fatal("out-of-range child bound", err)
	}
	if err := child.Close(); err != nil {
		t.Fatal(err)
	}
	if err := root.Close(); err != nil {
		t.Fatal(err)
	}
	if err := child.PushBound(0, &one, false, false); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed child", err)
	}
	if err := root.Close(); err != nil {
		t.Fatal("idempotent close", err)
	}
}
