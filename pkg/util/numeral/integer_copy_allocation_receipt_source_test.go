// Source: src/util/mpz.cpp:big_set; src/util/mpz.h:allocate;
// src/math/polynomial/polynomial.cpp:3318-3338.
package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestIntegerCopyAllocationReceiptUnchangedWithoutAllocation(t *testing.T) {
	m, _ := integerManager(t)
	out := NewInt(0)
	var receipt integerCopyAllocationReceipt
	serviceMust(t, m.CopyFromOwnerWithReceipt(&out, m, &m.Two64, &receipt))
	if receipt.invocation == 0 {
		t.Fatal("actual allocating COPY did not acquire invocation")
	}
	before := receipt
	used, state, stats := m.BudgetField.Used(), m.Memory.State(), m.Memory.Stats()
	serviceMust(t, m.CopyFromOwnerWithReceipt(&out, m, &m.Two64, &receipt))
	if receipt != before || m.BudgetField.Used() != used || m.Memory.State() != state || m.Memory.Stats() != stats {
		t.Fatal("capacity reuse acquired allocation authority")
	}
	serviceMust(t, m.CopyFromOwnerWithReceipt(&out, m, &out, &receipt))
	if receipt != before {
		t.Fatal("source self SET acquired allocation authority")
	}
	inline := NewInt(7)
	src := NewInt(11)
	serviceMust(t, m.CopyFromOwnerWithReceipt(&inline, m, &src, &receipt))
	if receipt != before || inline != src {
		t.Fatal("inline source SET acquired allocation authority")
	}
	serviceMust(t, m.Del(&out))
}

func TestIntegerCopyAllocationReceiptExactBackingAndForeignReplacement(t *testing.T) {
	for _, direct := range []bool{false, true} {
		t.Run(map[bool]string{false: "pooled", true: "direct"}[direct], func(t *testing.T) {
			b := arena.NewBudget(1 << 24)
			m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 8, SlotLimit: 8, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: direct}})
			serviceMust(t, e)
			out := NewInt(19)
			var receipt integerCopyAllocationReceipt
			e = m.CopyFromOwnerWithReceipt(&out, m, &m.Two64, &receipt)
			if !errors.Is(e, arena.ErrSlotLimit) || m.PendingCell.Object == (umemory.ObjectRef{}) || !m.MatchesCopyAllocationInvocation(&receipt) || receipt.Frame.Roots.Cell != m.PendingCell {
				t.Fatal("actual backing before descriptor refusal", e, m.PendingCell)
			}
			integerPhysicalAudit(t, m, &out)
			old := m.PendingCell
			serviceMust(t, m.CleanupPendingAllocation())
			serviceMust(t, m.Del(&out))
			out = NewInt(23)
			e = m.Copy(&out, &m.Two64)
			if !errors.Is(e, arena.ErrSlotLimit) || m.PendingCell.Object == (umemory.ObjectRef{}) || m.PendingCell.Object == old.Object {
				t.Fatal("real same-cardinality replacement", e)
			}
			foreign := m.PendingCell
			used, state, stats := b.Used(), m.Memory.State(), m.Memory.Stats()
			integerPhysicalAudit(t, m, &out)
			serviceMust(t, m.CleanupCopyAllocationInvocation(&receipt))
			if m.PendingCell != foreign || b.Used() != used || m.Memory.State() != state || m.Memory.Stats() != stats {
				t.Fatal("older receipt consumed foreign backing")
			}
			serviceMust(t, m.CleanupPendingAllocation())
			serviceMust(t, m.Del(&out))
			out = NewInt(29)
			var replacement integerCopyAllocationReceipt
			e = m.CopyFromOwnerWithReceipt(&out, m, &m.Two64, &replacement)
			if !errors.Is(e, arena.ErrSlotLimit) || replacement.Frame != receipt.Frame || replacement.invocation == receipt.invocation {
				t.Fatal("real reused frame unique invocation", e)
			}
			current := m.PendingCell
			used, state, stats = b.Used(), m.Memory.State(), m.Memory.Stats()
			serviceMust(t, m.CleanupCopyAllocationInvocation(&receipt))
			if m.PendingCell != current || b.Used() != used || m.Memory.State() != state || m.Memory.Stats() != stats {
				t.Fatal("stale receipt consumed current backing")
			}
			integerPhysicalAudit(t, m, &out)
			serviceMust(t, m.CleanupCopyAllocationInvocation(&replacement))
			serviceMust(t, m.Del(&out))
			serviceMust(t, m.CloseChecked())
			if b.Used() != 0 {
				t.Fatal("actual named owner retained charge", b.Used())
			}
		})
	}
}
