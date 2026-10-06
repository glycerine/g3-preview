// Source: src/util/mpf.cpp:1984-2027; actual reached scoped NUM/division leaves.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestArbitraryFloatRoundColdCachedPowerAndNumericPrefixes(t *testing.T) {
	success := false
	refusal := false
	for ordinal := int64(0); ordinal < 100; ordinal++ {
		b := arena.NewBudget(1 << 28)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		digits := make([]uint32, 63)
		digits[0] = 1
		digits[56] = 1 << 8
		digits[62] = 1 << 18
		var sig Int
		serviceMust(t, m.integer.SetLimbs(&sig, digits))
		var x ArbitraryFloat
		serviceMust(t, m.SetComponents(&x, 8, 2000, false, -5000, &sig))
		b.FailAfter(ordinal)
		body := m.Round(ArbitraryFloatNearestEven, &x)
		s := &m.roundScope
		t.Log("reached prefix", ordinal, body, "pending", m.integer.pendingCell, "source", s.sourceCause, "tmp", m.integer.tmp)
		integerPhysicalAudit(t, m.integer, &sig, &x.significand, &s.sigma, &s.cap, &s.sticky)
		if b.Used() != m.integer.memory.Stats().TotalBytes+m.OwnedStorageBytes() {
			t.Fatal("before lift actual round cold graph", ordinal)
		}
		if body != nil && m.integer.pendingCell.object != (memory.ObjectRef{}) {
			captured := s.numeric.cell == m.integer.pendingCell
			for w := m.powerWrappers; w != nil; w = w.next {
				if w.roots.cell == m.integer.pendingCell {
					captured = true
				}
			}
			if !captured {
				t.Fatal("actual cache/direct descriptor child uncaptured", ordinal, body)
			}
		}

		pending, e := m.integer.allocator.PendingRoots()
		serviceMust(t, e)
		for i, p := range pending {
			if body != nil && p != (memory.Block{}) {
				captured := s.numeric.allocator[i] == p
				for w := m.powerWrappers; w != nil; w = w.next {
					if w.roots.allocator[i] == p {
						captured = true
					}
				}
				if !captured {
					t.Fatal("actual cache/direct raw child uncaptured", ordinal, body, i, p)
				}
			}
		}
		for i := range m.integer.natural.division {
			p := m.integer.natural.division[i].pending
			if body != nil && p != (memory.Block{}) && s.buffers[i].pending != p {
				t.Fatal("actual Natural division buffer prefix uncaptured", ordinal, body, i, p)
			}
		}
		b.FailAfter(-1)
		serviceMust(t, m.CleanupRound())
		serviceMust(t, m.Del(&x))
		serviceMust(t, m.integer.Del(&sig))
		if e := m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
			t.Fatal("actual class source stop", ordinal, e)
		}
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
		if body == nil {
			success = true
			break
		}
		refusal = true
	}
	if !success || !refusal {
		t.Fatal("missing actual round refusal/success", refusal, success)
	}
}
