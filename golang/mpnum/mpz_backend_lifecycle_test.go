// Licensed under the MIT license.
// Source: src/util/mpz.h, src/util/mpz.cpp, src/util/mpq.h.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestIntegerBackendScratchHeaderAndSpareDigit(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64}})
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	object, e := m.cellObject(m.tmp[1])
	if e != nil {
		t.Fatal(e)
	}
	raw, off, e := m.allocator.RawLocation(object)
	if e != nil {
		t.Fatal(e)
	}
	if e = m.prepareTmpSpan(1, 4); e != nil {
		t.Fatal(e)
	}
	if _, e = m.memory.ReadUint32(raw, off); !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("scratch capacity borrow invented size", e)
	}
	x := NewInt(0)
	if e = m.SetUint64(&x, 0x80000000); e != nil {
		t.Fatal(e)
	}
	c := m.cell(x.cell)
	if v, e := c.backing.readWord(m.allocator, 12); e != nil || v != 0 {
		t.Fatal("typed64 high word", v, e)
	}
	if v, e := c.backing.readWord(m.allocator, 0); e != nil || v != 1 {
		t.Fatal("size publication", v, e)
	}
	if c.payload != (arena.VectorRef{}) || m.words.Stats().Live != 0 {
		t.Fatal("duplicate numeric payload")
	}
	if e = m.Del(&x); e != nil {
		t.Fatal(e)
	}
}

func TestIntegerBackendConstructorReservationSweep(t *testing.T) {
	success := false
	for stop := int64(0); stop < 80; stop++ {
		b := arena.NewBudget(1 << 24)
		b.FailAfter(stop)
		m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64}})
		if e == nil {
			m.Close()
			success = true
		} else {
			var owner *IntegerManager
			if errors.As(e, &owner) {
				b.FailAfter(-1)
				if cleanup := owner.Cleanup(); cleanup != nil {
					t.Fatal(stop, cleanup)
				}
			}
		}
		if b.Used() != 0 {
			t.Fatalf("constructor stop%d retained unowned physical/header charges %d: %v", stop, b.Used(), e)
		}
		if success {
			break
		}
	}
	if !success {
		t.Fatal("no complete constructor")
	}
}

func TestIntegerBackendPooledWarmReuseNoAllocations(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64}})
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	x := NewInt(0)
	if e = m.SetUint64(&x, 1<<40); e != nil {
		t.Fatal(e)
	}
	if e = m.Del(&x); e != nil {
		t.Fatal(e)
	}
	physical := m.memory.Stats()
	used := b.Used()
	count, e := m.memory.AllocationCount()
	if e != nil {
		t.Fatal(e)
	}
	b.FailAfter(0)
	var problem error
	allocations := testing.AllocsPerRun(100, func() {
		if problem != nil {
			return
		}
		if problem = m.SetUint64(&x, 1<<40); problem == nil {
			problem = m.Del(&x)
		}
	})
	if problem != nil || allocations != 0 {
		t.Fatal("warmed allocation", allocations, problem)
	}
	afterCount, e := m.memory.AllocationCount()
	if e != nil {
		t.Fatal(e)
	}
	if m.memory.Stats() != physical || b.Used() != used || afterCount != count {
		t.Fatal("real physical plateau changed")
	}
	b.FailAfter(-1)
}

func TestIntegerBackendPhysicalPrefixBeforeDescriptorLimit(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 8, SlotLimit: 8, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64}})
	if e != nil {
		t.Fatal(e)
	}
	before := m.allocator.Stats()
	out := NewInt(7)
	if e = m.SetUint64(&out, 1<<40); !errors.Is(e, arena.ErrSlotLimit) {
		t.Fatal("expected checked descriptor exhaustion after actual source allocation", e)
	}
	if out.cell != 0 || out.val != 7 {
		t.Fatal("source capacity preparation failed before pointer publication", out)
	}
	root := m.pendingCell
	if root.object == (memory.ObjectRef{}) {
		t.Fatal("actual unpublished physical object lost")
	}
	if v, e := root.readWord(m.allocator, 4); e != nil || v != 6 {
		t.Fatal("capacity-only source publication", v, e)
	}
	if _, e = root.readWord(m.allocator, 0); !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("unpublished size synthesized", e)
	}
	if m.cells.Stats().Live != 6 || m.allocator.Stats().Chunks != before.Chunks {
		t.Fatal("wrong prefix roots")
	}
	if e = m.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("pending actual object not cleaned", b.Used())
	}
}
