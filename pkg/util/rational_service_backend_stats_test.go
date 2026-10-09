// Source: src/util/rational.cpp, src/util/mpz.h.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
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
	z := s.Manager.Z
	seen := make(map[ObjectRef]bool)
	var objects uint32
	for n := uint32(1); n <= z.Cells.Stats().Slots; n++ {
		c, err := z.Cells.Read(arena.Slot(n))
		if err != nil {
			continue
		}
		if c.Released {
			continue
		}
		// A generation-validated object location is the physical payload authority.
		if _, _, err := z.Allocator.RawLocation(c.Backing.Object); err != nil {
			t.Fatal(err)
		}
		// Each cell has a distinct generation-bearing allocator token.
		id := c.Backing.Object
		if seen[id] {
			t.Fatal("duplicate physical payload", id)
		}
		seen[id] = true
		objects++
	}
	if objects != after.LiveCells || after.LiveNumericObjects != objects || after.PendingCellObjects != 0 || after.RetainedCellDescriptors != 0 || after.NaturalPayloads != 0 || after.LivePayloads != objects {
		t.Fatal("payload graph must include actual cells and genuine Natural buffers", objects, after)
	}
}

func TestRationalServiceStatsIncludesUnpublishedPhysicalCell(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	config := serviceConfig(b)
	config.Integers.Cells.PageSize = 8
	config.Integers.Cells.SlotLimit = 8 // six source cells plus two charged dead descriptors
	config.Integers.Profile = SmallAllocatorProfile{PointerBits: 64}
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
	z := s.Manager.Z
	if z.PendingCell.Object == (ObjectRef{}) {
		t.Fatal("missing completed physical allocation")
	}
	if _, _, err := z.Allocator.RawLocation(z.PendingCell.Object); err != nil {
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
