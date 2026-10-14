package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestFastFloatOwnedRuntimeJoinsSourceProcess(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	process := umemory.NewProcessAccounting()
	config := DefaultFastFloatConfig(arena.VectorConfig{Descriptors: arena.Config{Budget: budget}})
	config.InitialCapacity = 4
	config.Process = process
	fast, err := NewFastFloatManager(config)
	if err != nil {
		t.Fatal(err)
	}
	owned := fast.RuntimeOwner()
	if owned == nil || owned.ProcessOwner() != process {
		t.Fatal("fast-float raw runtime is outside its source process")
	}
	other, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, Process: process, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	a, err := owned.Allocate(8)
	if err != nil {
		t.Fatal(err)
	}
	b, err := other.Allocate(16)
	if err != nil || process.AllocationSize() != 40 {
		t.Fatal("cross-owner source allocation accounting", err, process.AllocationSize())
	}
	if err := owned.Deallocate(&a); err != nil {
		t.Fatal(err)
	}
	if err := other.Deallocate(&b); err != nil || process.AllocationSize() != 0 {
		t.Fatal("cross-owner source deallocation accounting", err, process.AllocationSize())
	}
	wrong := umemory.NewProcessAccounting()
	before := budget.Used()
	config.Memory, config.Process = other, wrong
	if m, err := NewFastFloatManager(config); m != nil || !errors.Is(err, arena.ErrConfig) || budget.Used() != before {
		t.Fatal("mismatched borrowed runtime and process changed storage", m, err, budget.Used(), before)
	}
	config.Memory, config.Process = other, process
	borrowed, err := NewFastFloatManager(config)
	if err != nil || borrowed.RuntimeOwner() != other {
		t.Fatal("matched borrowed runtime was rejected", err)
	}
	borrowed.Close()
	fast.Close()
	if err := other.Close(); err != nil || budget.Used() != 0 {
		t.Fatal("fast-float process teardown", err, budget.Used())
	}
}

func TestFastFloatOwnedRuntimeRejectsProcessABIWithoutStorage(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	process := umemory.NewProcessAccounting()
	first, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, Process: process, SizeTBits: 32})
	if err != nil {
		t.Fatal(err)
	}
	before := budget.Used()
	config := DefaultFastFloatConfig(arena.VectorConfig{Descriptors: arena.Config{Budget: budget}})
	config.InitialCapacity, config.Process = 4, process
	if m, err := NewFastFloatManager(config); m != nil || !errors.Is(err, arena.ErrConfig) || budget.Used() != before {
		t.Fatal("mixed process ABI retained MPFF constructor storage", m, err, budget.Used(), before)
	}
	if err := first.Close(); err != nil || budget.Used() != 0 {
		t.Fatal("process ABI fixture teardown", err, budget.Used())
	}
}
