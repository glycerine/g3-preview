package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestIntegerOwnedRuntimeJoinsSourceProcess(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	process := umemory.NewProcessAccounting()
	config := IntegerConfig{Process: process, Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}
	m, err := NewIntegerManager(config)
	if err != nil {
		t.Fatal(err)
	}
	if m.RuntimeOwner() == nil || m.RuntimeOwner().ProcessOwner() != process || m.Natural.RuntimeOwner() != m.RuntimeOwner() {
		t.Fatal("integer and natural owners do not share the source process")
	}
	other, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, Process: process, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	before := process.AllocationSize()
	h, err := other.Allocate(16)
	if err != nil || process.AllocationSize() != before+24 {
		t.Fatal("integer process missed the second runtime allocation", err, process.AllocationSize(), before)
	}
	if err := other.Deallocate(&h); err != nil || process.AllocationSize() != before {
		t.Fatal("integer process missed the second runtime release", err, process.AllocationSize(), before)
	}
	config.Memory, config.Process = other, umemory.NewProcessAccounting()
	used := budget.Used()
	if foreign, err := NewIntegerManager(config); foreign != nil || !errors.Is(err, arena.ErrConfig) || budget.Used() != used {
		t.Fatal("mismatched borrowed runtime and process changed storage", foreign, err, budget.Used(), used)
	}
	if err := m.CloseChecked(); err != nil {
		t.Fatal(err)
	}
	if err := other.Close(); err != nil || budget.Used() != 0 {
		t.Fatal("integer process teardown", err, budget.Used())
	}
}

func TestIntegerOwnedRuntimeRejectsProcessABIWithoutStorage(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	process := umemory.NewProcessAccounting()
	first, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, Process: process, SizeTBits: 32})
	if err != nil {
		t.Fatal(err)
	}
	before := budget.Used()
	config := IntegerConfig{Process: process, Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}
	if m, err := NewIntegerManager(config); m != nil || !errors.Is(err, arena.ErrConfig) || budget.Used() != before {
		t.Fatal("mixed process ABI retained integer constructor storage", m, err, budget.Used(), before)
	}
	if err := first.Close(); err != nil || budget.Used() != 0 {
		t.Fatal("process ABI fixture teardown", err, budget.Used())
	}
}
