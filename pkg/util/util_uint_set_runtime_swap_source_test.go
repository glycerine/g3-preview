// Source: src/util/uint_set.h:42-44; src/util/vector.h:453-455.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-12-07.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func uintSetProvidersAudit(t *testing.T, providers []*Runtime, sets ...*RuntimeUintSet) {
	t.Helper()
	cold := uint64(0)
	for _, s := range sets {
		if !s.closed {
			cold += s.ownerBytes
		}
	}
	total := cold
	for _, r := range providers {
		seen := map[Block]bool{}
		var physical, bitmap uint64
		for _, s := range sets {
			if s.closed || s.words.Raw.Runtime != r {
				continue
			}
			refs := []Block{s.words.Raw.Block, s.words.Raw.Pending}
			if s.destroyed || s.words.Raw.PendingReplaces {
				refs[0] = Block{}
			}
			for _, h := range refs {
				if h == (Block{}) {
					continue
				}
				if seen[h] {
					t.Fatal("duplicate provider owner", h)
				}
				seen[h] = true
				n, e := r.Size(h)
				heapMust(t, e)
				bytes := n + uint64(r.SizeTBits()/8)
				physical += bytes
				bitmap += ((bytes + 63) / 64) * 8
			}
		}
		st := r.Stats()
		if st.LiveBlocks != uint32(len(seen)) || st.PhysicalBytes != physical || st.InitializationBytes != bitmap {
			t.Fatal("provider roots", st, len(seen), physical, bitmap)
		}
		total += st.TotalBytes
	}
	if providers[0].Budget().Used() != total {
		t.Fatal("whole provider graph", providers[0].Budget().Used(), total)
	}
}
func TestUintSetRuntimeSwapActualProviderProvenance(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	ra, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	rb, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	a, e := NewRuntimeUintSet(ra, b)
	heapMust(t, e)
	c, e := NewRuntimeUintSet(rb, b)
	heapMust(t, e)
	heapMust(t, a.Insert(1))
	heapMust(t, c.Insert(63))
	it, e := a.Begin()
	heapMust(t, e)
	ar, cr := a.words.Raw.Block, c.words.Raw.Block
	ast, cst := ra.State(), rb.State()
	bytes := b.Used()
	aself, cself := a.self, c.self
	b.FailAfter(0)
	heapMust(t, a.Swap(c))
	if a.words.Raw.Block != cr || c.words.Raw.Block != ar || a.words.Raw.Runtime != rb || c.words.Raw.Runtime != ra || a.self != aself || c.self != cself || bytes != b.Used() || ra.State() != ast || rb.State() != cst {
		t.Fatal("source pointer swap/provenance only")
	}
	uintSetProvidersAudit(t, []*Runtime{ra, rb}, a, c)
	b.FailAfter(-1)
	// Captured endpoint remains32: current-member reads find no bit before it.
	heapMust(t, it.Advance())
	v, e := it.Value()
	heapMust(t, e)
	if v != 32 {
		t.Fatal(v)
	}
	heapMust(t, a.Insert(511))
	if ra.Stats().LiveBlocks != 1 || rb.Stats().LiveBlocks != 1 {
		t.Fatal("growth physical provider")
	}
	uintSetProvidersAudit(t, []*Runtime{ra, rb}, a, c)
	if e = ra.Close(); e != ErrRuntimeLive {
		t.Fatal("actual imported block lifetime", e)
	}
	if e = rb.Close(); e != ErrRuntimeLive {
		t.Fatal(e)
	}
	heapMust(t, a.Close())
	if rb.Stats().LiveBlocks != 0 || ra.Stats().LiveBlocks != 1 {
		t.Fatal("physical finalization provider")
	}
	heapMust(t, c.Close())
	heapMust(t, ra.Close())
	heapMust(t, rb.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestUintSetRuntimeSwapEmptyProviderLifetime(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	ra, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	rb, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	a, e := NewRuntimeUintSet(ra, b)
	heapMust(t, e)
	c, e := NewRuntimeUintSet(rb, b)
	heapMust(t, e)
	heapMust(t, a.Swap(c))
	uintSetProvidersAudit(t, []*Runtime{ra, rb}, a, c)
	heapMust(t, ra.Close())
	if _, e = c.Contains(0); e != arena.ErrClosed {
		t.Fatal("current borrowed provider guard", e)
	}
	heapMust(t, a.Insert(2))
	heapMust(t, c.Close())
	heapMust(t, a.Close())
	heapMust(t, rb.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestUintSetRuntimeSwappedGrowthEveryActualRefusal(t *testing.T) {
	failures, success := 0, 0
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		b := arena.NewBudget(1 << 24)
		ra, e := NewRuntime(RuntimeConfig{Budget: b})
		heapMust(t, e)
		rb, e := NewRuntime(RuntimeConfig{Budget: b})
		heapMust(t, e)
		a, e := NewRuntimeUintSet(ra, b)
		heapMust(t, e)
		c, e := NewRuntimeUintSet(rb, b)
		heapMust(t, e)
		heapMust(t, c.Insert(1))
		heapMust(t, a.Swap(c))
		ast := ra.Stats()
		b.FailAfter(ordinal)
		e = a.Insert(511)
		uintSetProvidersAudit(t, []*Runtime{ra, rb}, a, c)
		if ra.Stats() != ast {
			t.Fatal("growth touched nonphysical provider")
		}
		if e != nil {
			failures++
			intHeapActualRefusal(t, rb, e)
		} else {
			success++
			yes, e := a.Contains(511)
			heapMust(t, e)
			if !yes {
				t.Fatal("success endpoint")
			}
		}
		b.FailAfter(-1)
		if stop := a.Close(); stop != nil {
			if !a.words.Raw.PendingReplaces {
				t.Fatal(stop)
			}
			heapMust(t, a.CleanupStopped())
		}
		heapMust(t, c.Close())
		heapMust(t, ra.Close())
		heapMust(t, rb.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
		if success != 0 {
			t.Logf("refusals%d firstsuccess%d", failures, ordinal)
			break
		}
	}
	if failures == 0 || success != 1 {
		t.Fatal(failures, success)
	}
}
func TestUintSetRuntimeSwapWarmNoEvents(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	ra, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	rb, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	a, e := NewRuntimeUintSet(ra, b)
	heapMust(t, e)
	c, e := NewRuntimeUintSet(rb, b)
	heapMust(t, e)
	heapMust(t, a.Insert(1))
	heapMust(t, c.Insert(63))
	ast, cst := ra.State(), rb.State()
	used := b.Used()
	b.FailAfter(0)
	allocs := testing.AllocsPerRun(100, func() {
		if e := a.Swap(c); e != nil {
			panic(e)
		}
		if e := a.Swap(c); e != nil {
			panic(e)
		}
	})
	if allocs != 0 || b.Used() != used || ra.State() != ast || rb.State() != cst {
		t.Fatal("zero event swap", allocs)
	}
	uintSetProvidersAudit(t, []*Runtime{ra, rb}, a, c)
	b.FailAfter(-1)
	heapMust(t, a.Close())
	heapMust(t, c.Close())
	heapMust(t, ra.Close())
	heapMust(t, rb.Close())
}
