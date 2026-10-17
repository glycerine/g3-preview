package polynomial

import (
	"strconv"
	"testing"
)

func TestAlgebraicManagerEveryConstructorReservationAndCleanup(t *testing.T) {
	succeeded := false
	for ordinal := int64(0); ordinal < 256 && !succeeded; ordinal++ {
		passed := t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			cfg, _, b := algebraicManagerFixture(t)
			// The borrowed Runtime retains its block-directory storage. Warm
			// that owner once; every tested manager/private allocator is new.
			before := cfg.Rationals.Manager.Integers().RuntimeOwner().Stats()
			coldBudget := b.Used()
			warm, e := NewAlgebraicManager(cfg)
			serviceMust(t, e)
			serviceMust(t, warm.Close())
			after := cfg.Rationals.Manager.Integers().RuntimeOwner().Stats()
			if after.LiveBlocks != before.LiveBlocks || after.PhysicalBytes != before.PhysicalBytes || b.Used()-coldBudget != after.DirectoryBytes-before.DirectoryBytes {
				t.Fatal("cold borrowed Runtime storage was not solely its directory", before, after, b.Used(), coldBudget)
			}
			baseline := b.Used()
			b.FailAfter(ordinal)
			m, body := NewAlgebraicManager(cfg)
			b.FailAfter(-1)
			succeeded = body == nil
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("unexpected constructor error", body)
			}
			if m != nil {
				if body != nil && m.completed {
					t.Fatal("failed constructor published an ordinary manager")
				}
				serviceMust(t, m.Close())
				if !m.closed || m.header != 0 {
					t.Fatal("constructor cleanup retained owner")
				}
			}
			if b.Used() != baseline {
				t.Fatal("constructor cleanup leaked actual owner bytes", ordinal, b.Used(), baseline)
			}
			serviceMust(t, cfg.Rationals.Manager.CheckOwnerReady())
			retry, e := NewAlgebraicManager(cfg)
			serviceMust(t, e)
			serviceMust(t, retry.Close())
			if b.Used() != baseline {
				t.Fatal("retry owner bytes", b.Used(), baseline)
			}
		})
		if !passed {
			return
		}
	}
	if !succeeded {
		t.Fatal("constructor reservation horizon did not reach success")
	}
}
