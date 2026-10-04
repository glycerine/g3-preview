// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestRationalServiceStatsCountsActualNumericPayloads(t *testing.T) {
	s, _ := serviceNew(t)
	before, e := s.Stats()
	serviceMust(t, e)
	v, e := s.NewValue()
	serviceMust(t, e)
	defer s.Del(&v)
	serviceMust(t, s.SetString(&v, "123456789012345678901234567890123456789/987654321098765432109876543210987654319"))
	after, e := s.Stats()
	serviceMust(t, e)
	if after.LiveCells != before.LiveCells+2 || after.LivePayloads != before.LivePayloads+2 {
		t.Fatal("actual numeric object copies must appear in payload statistics", before, after)
	}
	z := s.manager.z
	seen := make(map[memory.ObjectRef]bool)
	var objects uint32
	for n := uint32(1); n <= z.cells.Stats().Slots; n++ {
		c, err := z.cells.Read(arena.Slot(n))
		if err != nil {
			continue
		}
		if c.released {
			continue
		}
		// A generation-validated object location is the physical payload authority.
		if _, _, err := z.allocator.RawLocation(c.backing.object); err != nil {
			t.Fatal(err)
		}
		// Each cell has a distinct generation-bearing allocator token.
		id := c.backing.object
		if seen[id] {
			t.Fatal("duplicate physical payload", id)
		}
		seen[id] = true
		objects++
	}
	if objects != after.LiveCells || after.LiveNumericObjects != objects || after.PendingCellObjects != 0 || after.RetainedCellDescriptors != 0 || after.NaturalPayloads != z.words.Stats().Live || after.LivePayloads != objects+z.words.Stats().Live {
		t.Fatal("payload graph must include actual cells and genuine Natural buffers", objects, after)
	}
}

func TestRationalServiceStatsIncludesUnpublishedPhysicalCell(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	config := serviceConfig(b)
	config.Integers.Cells.PageSize = 8
	config.Integers.Cells.SlotLimit = 7
	config.Integers.Profile = memory.SmallAllocatorProfile{PointerBits: 64}
	s, e := NewRationalService(config)
	serviceMust(t, e)
	v, e := s.NewValue()
	serviceMust(t, e)
	before, e := s.Stats()
	serviceMust(t, e)
	e = s.SetUint64(&v, 1<<40)
	if !errors.Is(e, arena.ErrSlotLimit) {
		t.Fatal(e)
	}
	after, err := s.Stats()
	serviceMust(t, err)
	z := s.manager.z
	if z.pendingCell.object == (memory.ObjectRef{}) {
		t.Fatal("missing completed physical allocation")
	}
	if _, _, err := z.allocator.RawLocation(z.pendingCell.object); err != nil {
		t.Fatal(err)
	}
	if after.LiveCells != before.LiveCells || after.LiveNumericObjects != before.LiveNumericObjects || after.PendingCellObjects != 1 || after.LivePayloads != before.LivePayloads+1 || after.NaturalPayloads != before.NaturalPayloads {
		t.Fatal("pending object must remain visible before cleanup", before, after)
	}
	serviceMust(t, s.Del(&v))
	serviceMust(t, s.Close())
	if b.Used() != 0 {
		t.Fatal("retained charges", b.Used())
	}
}
