// Source: src/util/mpf.cpp:1984-2027; actual reached scoped NUM/division leaves.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatRoundReachedNumericLeafAuthority(t *testing.T) {
	success := false
	refusal := false
	for ordinal := int64(0); ordinal < 100; ordinal++ {
		b := arena.NewBudget(1 << 28)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		// Warm only the literal member caches used by the tiny-denormal branch.
		for _, n := range []uint32{2004, 2002, 2000, 1999} {
			_, e = m.roundPower(n, false)
			serviceMust(t, e)
		}
		_, e = m.roundBound(8, false)
		serviceMust(t, e)
		_, e = m.roundBound(8, true)
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
		s := &m.RoundScope
		t.Log("reached prefix", ordinal, body, "pending", m.integer.PendingCell, "source", s.sourceCause, "tmp", m.integer.Tmp)
		integerPhysicalAudit(t, m.integer, &sig, &x.significand, &s.sigma, &s.cap, &s.sticky)
		if b.Used() != m.integer.Memory.Stats().TotalBytes+m.OwnedStorageBytes() {
			t.Fatal("before lift actual round cold graph", ordinal)
		}
		if body != nil && m.integer.PendingCell.Object != (ObjectRef{}) && s.Numeric.Cell != m.integer.PendingCell {
			t.Fatal("actual reached rounding NUM leaf lost direct frame authority", ordinal, body, m.integer.PendingCell)
		}

		pending, e := m.integer.Allocator.PendingRoots()
		serviceMust(t, e)
		for i, p := range pending {
			if body != nil && p != (Block{}) && s.Numeric.Allocator[i] != p {
				t.Fatal("actual allocator returned failure prefix uncaptured", ordinal, body, i, p)
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
