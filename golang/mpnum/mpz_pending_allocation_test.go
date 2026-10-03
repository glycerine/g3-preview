package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestIntegerPendingAllocationExplicitCleanup(t *testing.T) {
	for _, direct := range []bool{false, true} {
		b := arena.NewBudget(1 << 24)
		m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 8, SlotLimit: 7, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: direct}})
		if e != nil {
			t.Fatal(e)
		}
		old := NewInt(17)
		out := NewInt(19)
		before := m.cells.Stats()
		e = m.Copy(&out, &m.two64)
		if !errors.Is(e, arena.ErrSlotLimit) || m.pendingCell.object == (memory.ObjectRef{}) {
			t.Fatal("physical allocation before descriptor refusal", e, m.pendingCell)
		}
		integerPhysicalAudit(t, m, &old, &out)
		pending := m.pendingCell.object
		state := m.memory.State()
		if e = m.CleanupPendingAllocation(); e != nil {
			t.Fatal(e)
		}
		if m.pendingCell.object != (memory.ObjectRef{}) || m.cells.Stats() != before || old != NewInt(17) || out.cell != m.failedRef {
			t.Fatal("cleanup changed source numeral publication", old, out, m.cells.Stats(), before)
		}
		if _, e = m.allocator.ObjectExtent(pending); e == nil {
			t.Fatal("pending object remains owned")
		}
		if m.memory.State().GlobalCount != state.GlobalCount || m.memory.State().OutOfMemory != state.OutOfMemory {
			t.Fatal("cleanup fabricated or cleared source allocation state")
		}
		after := b.Used()
		if e = m.CleanupPendingAllocation(); e != nil || b.Used() != after {
			t.Fatal("cleanup retry repeated release", e)
		}
		if _, e = m.Int64(&out); !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatal("failed numeral repaired", e)
		}
		if e = m.Del(&out); e != nil {
			t.Fatal(e)
		}
		integerPhysicalAudit(t, m, &old, &out)
		if e = m.CloseChecked(); e != nil || b.Used() != 0 {
			t.Fatal("owning cleanup", e, b.Used())
		}
		if e = m.CleanupPendingAllocation(); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("closed owner guard", e)
		}
	}
}
