// Source: src/util/mpf.cpp:2100-2131; src/util/mpz.h:156-164.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatRoundSqrtDetachedDescriptorAcrossOlderScratchStop(t *testing.T) {
	for _, direct := range []bool{false, true} {
		found := false
		for ordinal := int64(0); ordinal < 20; ordinal++ {
			b := arena.NewBudget(1 << 27)
			m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: direct}})
			serviceMust(t, e)
			_, e = m.roundPower(32, false)
			serviceMust(t, e)
			sig := NewInt(1 << 26)
			var holders [32]Int
			for i := range holders {
				serviceMust(t, m.integer.SetLimbs(&holders[i], []uint32{uint32(i + 1), 1}))
			}
			var x ArbitraryFloat
			serviceMust(t, m.SetComponents(&x, 8, 33, false, 0, &sig))
			var receipt ArbitraryFloatRoundReceipt
			b.FailAfter(ordinal)
			returned := m.RoundSqrtWithReceipt(ArbitraryFloatNearestEven, &x, &receipt)
			s := &m.RoundSqrtScope
			t.Log("prefix", direct, ordinal, returned, s.Numeric.Cell, s.Transferred.Block())
			named := []*Int{&sig, &x.significand}
			for i := range holders {
				named = append(named, &holders[i])
			}
			integerPhysicalAudit(t, m.integer, named...)
			if b.Used() != m.memory.Stats().TotalBytes+m.OwnedStorageBytes() {
				t.Fatal("before lift exact graph", ordinal)
			}
			selected := returned != nil && s.history != nil && s.Numeric.Cell.Object != (ObjectRef{}) && (s.Numeric.Cell.Offset == 0) == direct
			b.FailAfter(-1)
			serviceMust(t, m.CleanupRoundSqrtWithReceipt(&receipt, false))
			serviceMust(t, m.Del(&x))
			serviceMust(t, m.integer.Del(&sig))
			for i := range holders {
				serviceMust(t, m.integer.Del(&holders[i]))
			}
			if selected {
				h := s.history
				original := h.originalCell
				body := h.bodyCause
				t.Log("selected descriptor", direct, ordinal, "scratch", m.integer.Tmp)
				if e = m.SourceFinalize(); e != arena.ErrInvalidSlot || !m.integer.sourceGuard {
					t.Fatal(e)
				}
				integerPhysicalAudit(t, m.integer)
				if s.Numeric.Cell != original || s.cellRetired {
					t.Fatal("older source scratch stop consumed younger captured cell")
				}
				t.Log("source state", m.integer.sourceGuard, s.cellRetired)
				if h.originalCell != original || h.bodyCause != body {
					t.Fatal("source erased issued token/history")
				}
				serviceMust(t, m.CleanupRoundSqrtWithReceipt(&receipt, true))
				if h.originalCell != original || h.bodyCause != body || !s.cellRetired {
					t.Fatal("own replayed detached descriptor or erased history")
				}
				found = true
			} else {
				serviceMust(t, m.CleanupRoundSqrtWithReceipt(&receipt, true))
			}
			if e = m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
				t.Fatal(e)
			}
			serviceMust(t, m.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
			if found {
				break
			}
		}
		if !found {
			t.Fatal("defined raw helper descriptor/scratch-stop prefix missing", direct)
		}
	}
}
