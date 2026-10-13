package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/messages"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func fastProcessFixture(t *testing.T) (*FastContext, *numeral.FastFloatManager, *memory.Runtime) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	p := memory.NewProcessAccounting()
	r, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: p, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	other, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: p, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	cfg := numeral.DefaultFastFloatConfig(arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	cfg.InitialCapacity, cfg.Memory = 4, r
	m, err := numeral.NewFastFloatManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewFastContext(m, &control.Reslimit{Limit: 100}, []bool{false}, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	if c.process != p {
		t.Fatal("context did not borrow the numeral runtime's process")
	}
	t.Cleanup(func() {
		for _, err := range []error{c.Close(), m.CloseChecked(), other.Close(), r.Close()} {
			if err != nil {
				t.Error(err)
			}
		}
		if b.Used() != 0 {
			t.Error("retained process fixture budget", b.Used())
		}
	})
	return c, m, other
}

func TestFastCheckpointReadsSharedProcessAfterResourceIncrement(t *testing.T) {
	c, _, other := fastProcessFixture(t)
	c.maxDepth = 0
	c.maxMemory = c.process.AllocationSize()
	extra, err := other.Allocate(1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if extra != (memory.Block{}) {
			if err := other.Deallocate(&extra); err != nil {
				t.Error(err)
			}
		}
	}()
	c.limit.Limit, c.limit.Steps = 1, 1
	if err := c.Search(); !errors.Is(err, ErrFastContextCanceled) || c.limit.Steps != 2 {
		t.Fatal("resource limit must be first", err, c.limit.Steps)
	}
	c.limit.Limit = 100
	if err := c.Search(); !errors.Is(err, ErrSubpavingMaxMemory) || err.Error() != messages.MaxMemory || c.limit.Steps != 3 || len(c.leafQueue) != 1 || c.searchNode != nil {
		t.Fatal("process memory checkpoint before leaf selection", err, c.limit.Steps, len(c.leafQueue), c.searchNode)
	}
	if err := other.Deallocate(&extra); err != nil {
		t.Fatal(err)
	}
	if c.process.AllocationSize() != c.maxMemory {
		t.Fatal("process size did not return to exact limit")
	}
	if err := c.Search(); err != nil || c.searchNode != nil || len(c.leafQueue) != 0 {
		t.Fatal("equal size must pass and resume root", err, c.searchNode, len(c.leafQueue))
	}
}

func TestFastCheckpointReadsUnsharedRuntime(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	cfg := numeral.DefaultFastFloatConfig(arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	cfg.InitialCapacity, cfg.Memory = 4, r
	m, err := numeral.NewFastFloatManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewFastContext(m, &control.Reslimit{Limit: 100}, []bool{false}, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, err := range []error{c.Close(), m.CloseChecked(), r.Close()} {
			if err != nil {
				t.Error(err)
			}
		}
		if b.Used() != 0 {
			t.Error("unshared fast context owners", b.Used())
		}
	})
	if c.process != nil || c.runtime != r {
		t.Fatal("unexpected fast-float accounting owners", c.process, c.runtime)
	}
	if err := c.InitializeRoot(); err != nil {
		t.Fatal(err)
	}
	baseline, err := r.AllocationSize()
	if err != nil {
		t.Fatal(err)
	}
	c.maxMemory = baseline
	extra, err := r.Allocate(1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if extra != (memory.Block{}) {
			if err := r.Deallocate(&extra); err != nil {
				t.Error(err)
			}
		}
	}()
	if err := c.Search(); !errors.Is(err, ErrSubpavingMaxMemory) || c.searchNode != nil || len(c.leafQueue) != 1 || c.limit.Steps != 1 {
		t.Fatal("unshared fast-float checkpoint skipped runtime bytes", err, c.searchNode, len(c.leafQueue), c.limit.Steps)
	}
	if err := r.Deallocate(&extra); err != nil {
		t.Fatal(err)
	}
	if used, err := r.AllocationSize(); err != nil || used != baseline {
		t.Fatal("fast-float runtime did not return to checkpoint limit", used, err, baseline)
	}
	if err := sourceCheckpoint(c.limit, c.runtime, c.process, c.maxMemory, ErrFastContextCanceled); err != nil {
		t.Fatal("equal runtime allocation must pass checkpoint", err)
	}
}

func TestFastUnitAndQueueCheckpointsReadSharedProcess(t *testing.T) {
	c, m, other := fastProcessFixture(t)
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := numeral.NewScopedFastFloat(m)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := scalar.Close(); err != nil {
			t.Error(err)
		}
	}()
	v, err := scalar.Get()
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(v, 1); err != nil {
		t.Fatal(err)
	}
	unit, err := c.NewInequality(0, v, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.AddUnitClause(unit, true); err != nil {
		t.Fatal(err)
	}
	c.maxMemory = c.process.AllocationSize()
	extra, err := other.Allocate(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.AssertUnitClauses(root); !errors.Is(err, ErrSubpavingMaxMemory) || c.unitHead != 0 || root.lower[0] != nil {
		t.Fatal("unit checkpoint before publication", err, c.unitHead)
	}
	if err := other.Deallocate(&extra); err != nil {
		t.Fatal(err)
	}
	if err := c.AssertUnitClauses(root); err != nil {
		t.Fatal(err)
	}
	if len(c.queue) == 0 {
		t.Fatal("unit bound did not enqueue")
	}
	c.maxMemory = c.process.AllocationSize()
	extra, err = other.Allocate(1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := other.Deallocate(&extra); err != nil {
			t.Error(err)
		}
	}()
	if err := c.PropagateQueued(root); !errors.Is(err, ErrSubpavingMaxMemory) || c.queueHead != 0 || len(c.queue) == 0 {
		t.Fatal("queue checkpoint before dequeue", err, c.queueHead, len(c.queue))
	}
}
