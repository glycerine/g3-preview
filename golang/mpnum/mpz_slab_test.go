// Licensed under the MIT license.
// Concrete numeric-cell storage adaptation tests.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
	"testing"
)

func TestNumericCellSlabConcreteLifecycle(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	s, e := newNumericCellSlab(arena.Config{PageSize: 2, SlotLimit: 5, Budget: budget})
	if e != nil {
		t.Fatal(e)
	}
	slots := make([]arena.Slot, 5)
	gens := make([]arena.Generation, 5)
	for i := range slots {
		slots[i], gens[i], e = s.Allocate()
		if e != nil {
			t.Fatal(e)
		}
		if e = s.Write(slots[i], numericCell{size: uint32(i + 1), capacity: 6}); e != nil {
			t.Fatal(e)
		}
	}
	if _, _, e = s.Allocate(); !errors.Is(e, arena.ErrSlotLimit) {
		t.Fatal(e)
	}
	for i, slot := range slots {
		c, e := s.Read(slot)
		if e != nil || c.size != uint32(i+1) || !s.IsLive(slot, gens[i]) {
			t.Fatal(c, e)
		}
	}
	if e = s.Recycle(slots[1]); e != nil {
		t.Fatal(e)
	}
	if s.IsLive(slots[1], gens[1]) {
		t.Fatal("dead generation live")
	}
	slot, generation, e := s.Allocate()
	if e != nil || slot != slots[1] || generation != gens[1]+1 {
		t.Fatal(slot, generation, e)
	}
	if c, e := s.Read(slot); e != nil || c != (numericCell{}) {
		t.Fatal("recycled cell not reset", c, e)
	}
	s.Reset()
	if s.Stats().Live != 0 || s.Stats().RecordBytes != 0 {
		t.Fatal(s.Stats())
	}
	slot, generation, e = s.Allocate()
	if e != nil || !s.IsLive(slot, generation) {
		t.Fatal(e)
	}
	s.Close()
	s.Close()
	if budget.Used() != 0 {
		t.Fatal("close retained budget", budget.Used())
	}
	if _, _, e = s.Allocate(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestNumericCellSlabGenerationRetirement(t *testing.T) {
	s, e := newNumericCellSlab(arena.Config{PageSize: 1, SlotLimit: 1})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	slot, _, e := s.Allocate()
	if e != nil {
		t.Fatal(e)
	}
	s.state(slot).generation = arena.Generation(math.MaxUint32)
	if e = s.Recycle(slot); e != nil {
		t.Fatal(e)
	}
	if s.Stats().Retired != 1 {
		t.Fatal(s.Stats())
	}
	if _, _, e = s.Allocate(); !errors.Is(e, arena.ErrSlotLimit) {
		t.Fatal(e)
	}
}

func TestNumericCellSlabFailureAndReuseAllocations(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	s, e := newNumericCellSlab(arena.Config{PageSize: 1, Budget: budget})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	budget.FailAfter(0)
	if _, _, e = s.Allocate(); !errors.Is(e, arena.ErrInjectedFailure) || budget.Used() != s.Stats().OwnerBytes || s.Stats().Live != 0 {
		t.Fatal(e, s.Stats())
	}
	budget.FailAfter(-1)
	slot, _, e := s.Allocate()
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Recycle(slot); e != nil {
		t.Fatal(e)
	}
	var allocationError error
	allocations := testing.AllocsPerRun(100, func() {
		slot, _, allocationError = s.Allocate()
		if allocationError == nil {
			allocationError = s.Write(slot, numericCell{size: 1, capacity: 6})
		}
		if allocationError == nil {
			allocationError = s.Recycle(slot)
		}
	})
	if allocationError != nil || allocations != 0 {
		t.Fatal("recycling allocates", allocations, allocationError)
	}
}

func TestIntegerConcreteStorageWarmArithmeticAllocations(t *testing.T) {
	m, _ := integerManager(t)
	var a, b, out Int
	if e := m.SetInt64(&a, 1<<40); e != nil {
		t.Fatal(e)
	}
	if e := m.SetInt64(&b, 1<<39); e != nil {
		t.Fatal(e)
	}
	if e := m.SetInt64(&out, 1<<40); e != nil {
		t.Fatal(e)
	}
	var operationError error
	allocations := testing.AllocsPerRun(100, func() {
		operationError = m.Add(&a, &b, &out)
		if operationError == nil {
			operationError = m.Sub(&a, &b, &out)
		}
	})
	if operationError != nil || allocations != 0 {
		t.Fatal("warm Add/Sub allocates", allocations, operationError)
	}
	if value, e := m.Int64(&out); e != nil || value != 1<<39 {
		t.Fatal(value, e)
	}
}

func TestNumericCellSlabDirectoryPeakBudget(t *testing.T) {
	// Budget permits the final two-page layout but not overlapping old and new
	// directory arrays during growth. This preserves the original peak charge.
	pageBytes := uint64(reflect.TypeOf(numericCell{}).Size() + reflect.TypeOf(numericCellSlotState{}).Size())
	directoryEntry := uint64(reflect.TypeOf(numericCellPage{}).Size())
	budget := arena.NewBudget(uint64(reflect.TypeOf(numericCellSlab{}).Size()) + 2*pageBytes + 2*directoryEntry)
	s, e := newNumericCellSlab(arena.Config{PageSize: 1, Budget: budget})
	if e != nil {
		t.Fatal(e)
	}
	slot, generation, e := s.Allocate()
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Write(slot, numericCell{size: 7, capacity: 9}); e != nil {
		t.Fatal(e)
	}
	before := s.Stats()
	used := budget.Used()
	if _, _, e = s.Allocate(); !errors.Is(e, arena.ErrBudget) {
		t.Fatal("growth did not charge old+new directory", e)
	}
	if s.Stats() != before || budget.Used() != used || !s.IsLive(slot, generation) {
		t.Fatal("failed growth changed storage")
	}
	if c, e := s.Read(slot); e != nil || c.size != 7 {
		t.Fatal(c, e)
	}
	s.Close()
	if budget.Used() != 0 {
		t.Fatal(budget.Used())
	}
}
