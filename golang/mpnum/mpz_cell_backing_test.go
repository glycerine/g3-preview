// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestIntegerSourceCellBackingInitializesCapacityOnly(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	a, e := memory.NewSmallAllocator(memory.SmallAllocatorConfig{Runtime: r})
	if e != nil {
		t.Fatal(e)
	}
	c, e := allocateSourceCell(a, 6)
	if e != nil {
		t.Fatal(e)
	}
	if c.requested != 32 || c.extent != 32 || c.object == (memory.ObjectRef{}) || r.Stats().PhysicalBytes != 8200 {
		t.Fatal("actual source cell backing topology")
	}
	if v, e := c.readWord(a, 4); e != nil || v != 6 {
		t.Fatal("actual capacity store", v, e)
	}
	for _, off := range []uint64{0, 8, 12, 28} {
		if _, e := c.readWord(a, off); !errors.Is(e, memory.ErrRuntimeUninitialized) {
			t.Fatalf("invented cell initialization at%d: %v", off, e)
		}
	}
	if e = c.writeWord(a, 8, 99); e != nil {
		t.Fatal(e)
	}
	if v, e := c.readWord(a, 8); e != nil || v != 99 {
		t.Fatal("actual digit backing", v, e)
	}
	if _, e := c.readWord(a, 32); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("cell access escaped rounded extent", e)
	}
	if e = a.Deallocate(c.requested, c.object); e != nil {
		t.Fatal(e)
	}
	if _, e := c.readWord(a, 8); e == nil {
		t.Fatal("freed generation readable through raw chunk")
	}
	if e = a.Close(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("source cell owner leak")
	}
}

func TestIntegerSourceCellWrappedHeaderGuardUsesActualRoundedExtent(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		b := arena.NewBudget(1 << 24)
		r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: bits})
		if e != nil {
			t.Fatal(e)
		}
		a, e := memory.NewSmallAllocator(memory.SmallAllocatorConfig{Runtime: r})
		if e != nil {
			t.Fatal(e)
		}
		// cell_size wraps to4 even on LP64. LP64 pool rounds to8; ILP32
		// pool gives4, so the capacity-header write is the actual endangered action.
		c, e := allocateSourceCell(a, 0x3fffffff)
		if bits == 32 {
			if !errors.Is(e, arena.ErrBounds) {
				t.Fatalf("ILP32 capacity action %v", e)
			}
		} else {
			if e != nil {
				t.Fatal(e)
			}
			if v, e := c.readWord(a, 4); e != nil || v != 0x3fffffff {
				t.Fatal("LP64 rounded header write", v, e)
			}
		}
		if c.object == (memory.ObjectRef{}) || c.requested != 4 || c.extent != uint64(bits/8) || r.Stats().LiveBlocks != 1 {
			t.Fatal("wrapped header guard lost allocated object")
		}
		if e = c.writeWord(a, 8, 1); !errors.Is(e, arena.ErrBounds) {
			t.Fatal("digit escaped actual object", e)
		}
		if e = a.Deallocate(c.requested, c.object); e != nil {
			t.Fatal(e)
		}
		if e = a.Close(); e != nil {
			t.Fatal(e)
		}
		if e = r.Close(); e != nil {
			t.Fatal(e)
		}
		if b.Used() != 0 {
			t.Fatal("wrapped cell cleanup leak")
		}
	}
}
