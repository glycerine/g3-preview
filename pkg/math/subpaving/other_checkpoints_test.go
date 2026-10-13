package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestArbitraryCheckpointBorrowsProcessFromCoreRuntime(t *testing.T) {
	p := memory.NewProcessAccounting()
	f := newArbitraryClauseFixtureWithPolicy(t, 0, false, p)
	c := f.context
	if c.process != p {
		t.Fatal("MPF context did not borrow process")
	}
	b := arena.NewBudget(1024)
	other, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: p, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := other.Close(); err != nil {
			t.Error(err)
		}
	})
	c.maxMemory = p.AllocationSize()
	extra, err := other.Allocate(1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := other.Deallocate(&extra); err != nil {
			t.Error(err)
		}
	}()
	if err := c.Search(); !errors.Is(err, ErrSubpavingMaxMemory) || c.searchNode != nil || c.leafHead != c.root || c.limit.Steps != 1 {
		t.Fatal("MPF process checkpoint before root selection", err, c.searchNode, c.leafHead, c.limit.Steps)
	}
}

func TestHardwareCheckpointUsesBoundProcess(t *testing.T) {
	p := memory.NewProcessAccounting()
	b := arena.NewBudget(1024)
	r, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: p, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewHardwareContext(m, &control.Reslimit{Limit: 100}, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
		if err := r.Close(); err != nil {
			t.Error(err)
		}
	})
	copy := *p
	if err := c.BindProcessAccounting(&copy); !errors.Is(err, arena.ErrConfig) {
		t.Fatal("copied process must be refused", err)
	}
	if err := c.BindProcessAccounting(p); err != nil {
		t.Fatal(err)
	}
	c.maxMemory = p.AllocationSize()
	extra, err := r.Allocate(1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Deallocate(&extra); err != nil {
			t.Error(err)
		}
	}()
	if err := c.Search(); !errors.Is(err, ErrSubpavingMaxMemory) || c.searchNode != nil || c.leafHead != c.root || c.limit.Steps != 1 {
		t.Fatal("HWF process checkpoint before root selection", err, c.searchNode, c.leafHead, c.limit.Steps)
	}
}
