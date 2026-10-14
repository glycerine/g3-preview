// Source: src/util/memory_manager.cpp:33-44,90-95,124-149,274-305.
package memory

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestProcessAccountingKeepsOneSourceSizeTProfile(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		p := NewProcessAccounting()
		b := arena.NewBudget(1 << 20)
		first, err := NewRuntime(RuntimeConfig{Budget: b, Process: p, SizeTBits: bits})
		if err != nil {
			t.Fatal(bits, err)
		}
		otherBits := uint8(96 - bits)
		before := b.Used()
		if mixed, err := NewRuntime(RuntimeConfig{Budget: b, Process: p, SizeTBits: otherBits}); mixed != nil || !errors.Is(err, arena.ErrConfig) || b.Used() != before {
			t.Fatal("mixed process ABI changed owner storage", bits, mixed, err, b.Used(), before)
		}
		second, err := NewRuntime(RuntimeConfig{Budget: b, Process: p, SizeTBits: bits})
		if err != nil {
			t.Fatal(bits, err)
		}
		h, err := second.Allocate(0)
		if err != nil || p.AllocationCount() != 1 {
			t.Fatal("matched runtime did not share source counters", bits, h, err, p.AllocationCount())
		}
		if err := second.Deallocate(&h); err != nil {
			t.Fatal(err)
		}
		if err := second.Close(); err != nil {
			t.Fatal(err)
		}
		if err := first.Close(); err != nil {
			t.Fatal(err)
		}
		if mixed, err := NewRuntime(RuntimeConfig{Budget: b, Process: p, SizeTBits: otherBits}); mixed != nil || !errors.Is(err, arena.ErrConfig) {
			t.Fatal("ended runtime changed process ABI", bits, mixed, err)
		}
		if b.Used() != 0 {
			t.Fatal("source process runtime storage", b.Used())
		}
	}
}

func TestProcessProfilePublishesOnlyAfterSuccessfulRuntimeConstruction(t *testing.T) {
	p := NewProcessAccounting()
	b := arena.NewBudget(1 << 20)
	b.FailAfter(0)
	if runtime, err := NewRuntime(RuntimeConfig{Budget: b, Process: p, SizeTBits: 32}); runtime != nil || !errors.Is(err, arena.ErrInjectedFailure) || p.sizeTBits != 0 || b.Used() != 0 {
		t.Fatal("failed constructor fixed process ABI", runtime, err, p.sizeTBits, b.Used())
	}
	b.FailAfter(-1)
	runtime, err := NewRuntime(RuntimeConfig{Budget: b, Process: p, SizeTBits: 64})
	if err != nil || p.sizeTBits != 64 {
		t.Fatal("successful constructor did not publish process ABI", runtime, err, p.sizeTBits)
	}
	if err := runtime.Close(); err != nil || b.Used() != 0 {
		t.Fatal("runtime teardown", err, b.Used())
	}
}
