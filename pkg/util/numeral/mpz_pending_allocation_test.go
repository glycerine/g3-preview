// Source: src/util/mpz.h, src/util/mpz.cpp, src/util/mpq.h.
package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestIntegerPendingAllocationExplicitCleanup(t *testing.T) {
	for _, direct := range []bool{false, true} {
		b := arena.NewBudget(1 << 24)
		m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 8, SlotLimit: 8, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: direct}})
		if e != nil {
			t.Fatal(e)
		}
		old := NewInt(17)
		out := NewInt(19)
		before := m.Cells.Stats()
		e = m.Copy(&out, &m.Two64)
		if !errors.Is(e, arena.ErrSlotLimit) || m.PendingCell.Object == (umemory.ObjectRef{}) {
			t.Fatal("physical allocation before descriptor refusal", e, m.PendingCell)
		}
		integerPhysicalAudit(t, m, &old, &out)
		pending := m.PendingCell.Object
		state := m.Memory.State()
		if e = m.CleanupPendingAllocation(); e != nil {
			t.Fatal(e)
		}
		if m.PendingCell.Object != (umemory.ObjectRef{}) || m.Cells.Stats() != before || old != NewInt(17) || out.Cell != m.FailedNullRef {
			t.Fatal("cleanup changed source numeral publication", old, out, m.Cells.Stats(), before)
		}
		if _, e = m.Allocator.ObjectExtent(pending); e == nil {
			t.Fatal("pending object remains owned")
		}
		if m.Memory.State().GlobalCount != state.GlobalCount || m.Memory.State().OutOfMemory != state.OutOfMemory {
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
