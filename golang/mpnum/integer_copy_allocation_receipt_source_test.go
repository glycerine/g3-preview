// Source: src/util/mpz.cpp:big_set; src/util/mpz.h:allocate;
// src/math/polynomial/polynomial.cpp:3318-3338.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestIntegerCopyAllocationReceiptUnchangedWithoutAllocation(t *testing.T) {
	m, _ := integerManager(t)
	out := NewInt(0)
	var receipt integerCopyAllocationReceipt
	serviceMust(t, m.copyFromOwnerWithReceipt(&out, m, &m.two64, &receipt))
	if receipt.invocation == 0 {
		t.Fatal("actual allocating COPY did not acquire invocation")
	}
	before := receipt
	used, state, stats := m.budget.Used(), m.memory.State(), m.memory.Stats()
	serviceMust(t, m.copyFromOwnerWithReceipt(&out, m, &m.two64, &receipt))
	if receipt != before || m.budget.Used() != used || m.memory.State() != state || m.memory.Stats() != stats {
		t.Fatal("capacity reuse acquired allocation authority")
	}
	serviceMust(t, m.copyFromOwnerWithReceipt(&out, m, &out, &receipt))
	if receipt != before {
		t.Fatal("source self SET acquired allocation authority")
	}
	inline := NewInt(7)
	src := NewInt(11)
	serviceMust(t, m.copyFromOwnerWithReceipt(&inline, m, &src, &receipt))
	if receipt != before || inline != src {
		t.Fatal("inline source SET acquired allocation authority")
	}
	serviceMust(t, m.Del(&out))
}

func TestIntegerCopyAllocationReceiptExactBackingAndForeignReplacement(t *testing.T) {
	for _, direct := range []bool{false, true} {
		t.Run(map[bool]string{false: "pooled", true: "direct"}[direct], func(t *testing.T) {
			b := arena.NewBudget(1 << 24)
			m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 8, SlotLimit: 8, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: direct}})
			serviceMust(t, e)
			out := NewInt(19)
			var receipt integerCopyAllocationReceipt
			e = m.copyFromOwnerWithReceipt(&out, m, &m.two64, &receipt)
			if !errors.Is(e, arena.ErrSlotLimit) || m.pendingCell.object == (memory.ObjectRef{}) || !m.matchesCopyAllocationInvocation(&receipt) || receipt.frame.roots.cell != m.pendingCell {
				t.Fatal("actual backing before descriptor refusal", e, m.pendingCell)
			}
			integerPhysicalAudit(t, m, &out)
			old := m.pendingCell
			serviceMust(t, m.CleanupPendingAllocation())
			serviceMust(t, m.Del(&out))
			out = NewInt(23)
			e = m.Copy(&out, &m.two64)
			if !errors.Is(e, arena.ErrSlotLimit) || m.pendingCell.object == (memory.ObjectRef{}) || m.pendingCell.object == old.object {
				t.Fatal("real same-cardinality replacement", e)
			}
			foreign := m.pendingCell
			used, state, stats := b.Used(), m.memory.State(), m.memory.Stats()
			integerPhysicalAudit(t, m, &out)
			serviceMust(t, m.cleanupCopyAllocationInvocation(&receipt))
			if m.pendingCell != foreign || b.Used() != used || m.memory.State() != state || m.memory.Stats() != stats {
				t.Fatal("older receipt consumed foreign backing")
			}
			serviceMust(t, m.CleanupPendingAllocation())
			serviceMust(t, m.Del(&out))
			out = NewInt(29)
			var replacement integerCopyAllocationReceipt
			e = m.copyFromOwnerWithReceipt(&out, m, &m.two64, &replacement)
			if !errors.Is(e, arena.ErrSlotLimit) || replacement.frame != receipt.frame || replacement.invocation == receipt.invocation {
				t.Fatal("real reused frame unique invocation", e)
			}
			current := m.pendingCell
			used, state, stats = b.Used(), m.memory.State(), m.memory.Stats()
			serviceMust(t, m.cleanupCopyAllocationInvocation(&receipt))
			if m.pendingCell != current || b.Used() != used || m.memory.State() != state || m.memory.Stats() != stats {
				t.Fatal("stale receipt consumed current backing")
			}
			integerPhysicalAudit(t, m, &out)
			serviceMust(t, m.cleanupCopyAllocationInvocation(&replacement))
			serviceMust(t, m.Del(&out))
			serviceMust(t, m.CloseChecked())
			if b.Used() != 0 {
				t.Fatal("actual named owner retained charge", b.Used())
			}
		})
	}
}
