package subpaving

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestArbitraryBoundTimestampSourceExceptionAfterPublication(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	c := f.context
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	c.timestamp = math.MaxUint64 - 1
	err = c.PublishBound(root, f.x, &f.four, true, false, true)
	if !errors.Is(err, ErrArbitraryContextTimestamp) || !errors.Is(err, ErrSubpavingFailure) {
		t.Fatal("source arbitrary bound timestamp exception", err)
	}
	if root.trail == nil || root.trail.timestamp != math.MaxUint64-1 || c.timestamp != math.MaxUint64 || len(c.queue) != 0 {
		t.Fatal("arbitrary bound publication must precede timestamp exception and queueing")
	}
}

func TestHardwareBoundTimestampSourceExceptionAfterPublication(t *testing.T) {
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewHardwareContext(m, &control.Reslimit{Limit: 1000}, 20, 1, 8192)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	}()
	x, err := c.NewVariable(false)
	if err != nil {
		t.Fatal(err)
	}
	root, err := c.CreateRootNode()
	if err != nil {
		t.Fatal(err)
	}
	var value numeral.HardwareFloat
	if err := m.SetInt32(&value, 4); err != nil {
		t.Fatal(err)
	}
	c.timestamp = math.MaxUint64 - 1
	err = c.PropagateBound(root, x, &value, true, false)
	if !errors.Is(err, ErrHardwareContextTimestamp) || !errors.Is(err, ErrSubpavingFailure) {
		t.Fatal("source hardware bound timestamp exception", err)
	}
	if root.trail == nil || root.trail.timestamp != math.MaxUint64-1 || c.timestamp != math.MaxUint64 || len(c.queue) != 0 {
		t.Fatal("hardware bound publication must precede timestamp exception and queueing")
	}
}
