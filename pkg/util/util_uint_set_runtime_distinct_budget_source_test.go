// Source: src/util/uint_set.h:42-44; src/util/vector.h:453-455.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-12-07.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Independent per-Budget equations: each actual provider owns its raw backing
// and directory; each wrapper's cold header remains with its original Budget.
func uintSetDistinctBudgetAudit(t *testing.T, providers []*Runtime, sets ...*RuntimeUintSet) {
	t.Helper()
	expected := map[*arena.Budget]uint64{}
	for _, s := range sets {
		if s == nil || s.closed {
			continue
		}
		expected[s.budget] += s.ownerBytes
	}
	for _, r := range providers {
		seen := map[Block]bool{}
		var physical, bitmap uint64
		for _, s := range sets {
			if s == nil || s.closed || s.words.Raw.Runtime != r {
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
					t.Fatal("duplicate actual block owner", h)
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
			t.Fatal("provider bijection", st, len(seen), physical, bitmap)
		}
		expected[r.Budget()] += st.TotalBytes
	}
	for b, n := range expected {
		if b.Used() != n {
			t.Fatal("independent physical/cold Budget", b.Used(), n)
		}
	}
}
func TestUintSetRuntimeSwapDistinctBudgetsEveryEmptyCombination(t *testing.T) {
	for mask := 0; mask < 4; mask++ {
		t.Run(string(rune('0'+mask)), func(t *testing.T) {
			ba, bc := arena.NewBudget(1<<24), arena.NewBudget(1<<24)
			ra, e := NewRuntime(RuntimeConfig{Budget: ba})
			heapMust(t, e)
			rc, e := NewRuntime(RuntimeConfig{Budget: bc})
			heapMust(t, e)
			a, e := NewRuntimeUintSet(ra, ba)
			heapMust(t, e)
			c, e := NewRuntimeUintSet(rc, bc)
			heapMust(t, e)
			if mask&1 != 0 {
				heapMust(t, a.Insert(1))
			}
			if mask&2 != 0 {
				heapMust(t, c.Insert(31))
			}
			ar, cr := a.words.Raw.Block, c.words.Raw.Block
			ah, ch := a.ownerBytes, c.ownerBytes
			ast, cst := ra.State(), rc.State()
			au, cu := ba.Used(), bc.Used()
			it, e := a.Begin()
			heapMust(t, e)
			ba.FailAfter(0)
			bc.FailAfter(0)
			heapMust(t, a.Swap(c))
			if a.budget != ba || c.budget != bc || a.ownerBytes != ah || c.ownerBytes != ch || a.words.Raw.Block != cr || c.words.Raw.Block != ar || a.words.Raw.Runtime != rc || c.words.Raw.Runtime != ra || ba.Used() != au || bc.Used() != cu || ra.State() != ast || rc.State() != cst {
				t.Fatal("pure source pointer/provenance exchange")
			}
			uintSetDistinctBudgetAudit(t, []*Runtime{ra, rc}, a, c)
			ba.FailAfter(-1)
			bc.FailAfter(-1)
			has, e := a.Contains(31)
			heapMust(t, e)
			if has != (mask&2 != 0) {
				t.Fatal(mask, has)
			}
			has, e = c.Contains(1)
			heapMust(t, e)
			if has != (mask&1 != 0) {
				t.Fatal(mask, has)
			}
			// A non-end captured iterator scans the current vector after the pointer move.
			if mask&1 != 0 {
				heapMust(t, it.Advance())
				v, e := it.Value()
				heapMust(t, e)
				want := uint32(32)
				if mask&2 != 0 {
					want = 31
				}
				if v != want {
					t.Fatal(v, want)
				}
			}
			heapMust(t, a.Insert(511))
			uintSetDistinctBudgetAudit(t, []*Runtime{ra, rc}, a, c)
			if e = rc.Close(); e != ErrRuntimeLive {
				t.Fatal("moved provider lifetime", e)
			}
			heapMust(t, a.Close())
			if rc.Stats().LiveBlocks != 0 {
				t.Fatal("wrong physical refund")
			}
			uintSetDistinctBudgetAudit(t, []*Runtime{ra, rc}, c)
			heapMust(t, c.Close())
			heapMust(t, ra.Close())
			heapMust(t, rc.Close())
			if ba.Used() != 0 || bc.Used() != 0 {
				t.Fatal(ba.Used(), bc.Used())
			}
		})
	}
}
func TestUintSetRuntimeAdoptedGrowthRefusesPhysicalBudgetOnly(t *testing.T) {
	failures, success := 0, 0
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		ba, bc := arena.NewBudget(1<<24), arena.NewBudget(1<<24)
		ra, e := NewRuntime(RuntimeConfig{Budget: ba})
		heapMust(t, e)
		rc, e := NewRuntime(RuntimeConfig{Budget: bc})
		heapMust(t, e)
		a, e := NewRuntimeUintSet(ra, ba)
		heapMust(t, e)
		c, e := NewRuntimeUintSet(rc, bc)
		heapMust(t, e)
		heapMust(t, c.Insert(1))
		heapMust(t, a.Swap(c))
		coldUsed := ba.Used()
		coldState := ra.State()
		ba.FailAfter(0)
		bc.FailAfter(ordinal)
		e = a.Insert(511)
		uintSetDistinctBudgetAudit(t, []*Runtime{ra, rc}, a, c)
		if ba.Used() != coldUsed || ra.State() != coldState {
			t.Fatal("growth charged cold Budget")
		}
		if e != nil {
			failures++
			intHeapActualRefusal(t, rc, e)
		} else {
			success++
			has, e := a.Contains(511)
			heapMust(t, e)
			if !has {
				t.Fatal("success endpoint")
			}
		}
		ba.FailAfter(-1)
		bc.FailAfter(-1)
		if stop := a.Close(); stop != nil {
			if !a.words.Raw.PendingReplaces {
				t.Fatal(stop)
			}
			heapMust(t, a.CleanupStopped())
		}
		heapMust(t, c.Close())
		heapMust(t, ra.Close())
		heapMust(t, rc.Close())
		if ba.Used() != 0 || bc.Used() != 0 {
			t.Fatal(ba.Used(), bc.Used())
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

func TestUintSetRuntimeAdoptedAssignAndCloneEveryActualRefusal(t *testing.T) {
	for route := 0; route < 2; route++ {
		failures, success := 0, 0
		for ordinal := int64(0); ordinal < 32; ordinal++ {
			ba, bc := arena.NewBudget(1<<24), arena.NewBudget(1<<24)
			ra, e := NewRuntime(RuntimeConfig{Budget: ba})
			heapMust(t, e)
			rc, e := NewRuntime(RuntimeConfig{Budget: bc})
			heapMust(t, e)
			a, e := NewRuntimeUintSet(ra, ba)
			heapMust(t, e)
			c, e := NewRuntimeUintSet(rc, bc)
			heapMust(t, e)
			heapMust(t, a.Insert(511))
			heapMust(t, c.Insert(1))
			heapMust(t, a.Swap(c))
			original := a.words.Raw.Block
			beforeCold := ba.Used()
			beforeColdState := ra.State()
			var copy *RuntimeUintSet
			ba.FailAfter(0)
			bc.FailAfter(ordinal)
			if route == 0 {
				e = a.Assign(c)
			} else {
				copy, e = c.Clone(rc, bc)
			}
			uintSetDistinctBudgetAudit(t, []*Runtime{ra, rc}, a, c, copy)
			if ba.Used() != beforeCold || ra.State() != beforeColdState {
				t.Fatal("adopted copy charged cold provider")
			}
			if e != nil {
				failures++
				intHeapActualRefusal(t, rc, e)
				if route == 0 {
					if _, probe := rc.Size(original); probe == nil {
						t.Fatal("source destination not physically destroyed")
					}
					if a.words.Raw.Block != original && !a.words.Raw.PendingReplaces {
						t.Fatal("unpublished source member changed")
					}
				}
			} else {
				success++
				dest := a
				if route == 1 {
					dest = copy
				}
				has, e := dest.Contains(511)
				heapMust(t, e)
				if !has {
					t.Fatal("exact copied source math")
				}
				if dest.words.Raw.Runtime != rc {
					t.Fatal("copy physical provider")
				}
				if route == 1 && dest.budget != bc {
					t.Fatal("clone explicit constructor cold owner")
				}
			}
			ba.FailAfter(-1)
			bc.FailAfter(-1)
			if copy != nil {
				if stop := copy.Close(); stop != nil {
					heapMust(t, copy.CleanupStopped())
				}
			}
			if stop := a.Close(); stop != nil {
				heapMust(t, a.CleanupStopped())
			}
			heapMust(t, c.Close())
			heapMust(t, ra.Close())
			heapMust(t, rc.Close())
			if ba.Used() != 0 || bc.Used() != 0 {
				t.Fatal(ba.Used(), bc.Used())
			}
			if success != 0 {
				t.Logf("route%d refusals%d firstsuccess%d", route, failures, ordinal)
				break
			}
		}
		if failures == 0 || success != 1 {
			t.Fatal(route, failures, success)
		}
	}
}
func TestUintSetRuntimeSwapDistinctBudgetsWarmAndEmptyLifetime(t *testing.T) {
	ba, bc := arena.NewBudget(1<<24), arena.NewBudget(1<<24)
	ra, e := NewRuntime(RuntimeConfig{Budget: ba})
	heapMust(t, e)
	rc, e := NewRuntime(RuntimeConfig{Budget: bc})
	heapMust(t, e)
	a, e := NewRuntimeUintSet(ra, ba)
	heapMust(t, e)
	c, e := NewRuntimeUintSet(rc, bc)
	heapMust(t, e)
	au, cu := ba.Used(), bc.Used()
	ast, cst := ra.State(), rc.State()
	ba.FailAfter(0)
	bc.FailAfter(0)
	allocs := testing.AllocsPerRun(100, func() {
		if e := a.Swap(c); e != nil {
			panic(e)
		}
		if e := a.Swap(c); e != nil {
			panic(e)
		}
	})
	if allocs != 0 || ba.Used() != au || bc.Used() != cu || ra.State() != ast || rc.State() != cst {
		t.Fatal("zero event provenance exchange", allocs)
	}
	heapMust(t, a.Swap(c))
	uintSetDistinctBudgetAudit(t, []*Runtime{ra, rc}, a, c)
	ba.FailAfter(-1)
	bc.FailAfter(-1)
	heapMust(t, ra.Close())
	if _, e = c.Contains(0); e != arena.ErrClosed {
		t.Fatal("actual current provider lifetime", e)
	}
	// Empty-source destruction invokes no allocator. Its original cold header
	// can be refunded after the expired physical provider without touching peers.
	heapMust(t, c.Close())
	heapMust(t, a.Close())
	heapMust(t, rc.Close())
	if ba.Used() != 0 || bc.Used() != 0 {
		t.Fatal(ba.Used(), bc.Used())
	}
}

func TestUintSetRuntimeIteratorCachedValueDoesNotReadDeadOwner(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	s, e := NewRuntimeUintSet(r, b)
	heapMust(t, e)
	heapMust(t, s.Insert(1))
	it, e := s.Begin()
	heapMust(t, e)
	heapMust(t, s.Close())
	heapMust(t, r.Close())
	value, e := it.Value()
	heapMust(t, e)
	if value != 1 {
		t.Fatal("source cached operator*", value)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestUintSetRuntimeIteratorIncrementAssertionBeforeDeadOwnerRead(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	s, e := NewRuntimeUintSet(r, b)
	heapMust(t, e)
	heapMust(t, s.Insert(1))
	it, e := s.End()
	heapMust(t, e)
	heapMust(t, s.Close())
	// ++end is a guarded source-invalid advance. Its index increment is real;
	// source DEBUG invariant precedes any read through the dead borrowed owner.
	e = it.Advance()
	if sourceDebug {
		if e != ErrRuntimeVectorAssertion {
			t.Fatal("source assertion precedence", e)
		}
	} else {
		if e != arena.ErrClosed {
			t.Fatal("actual reached owner read", e)
		}
	}
	value, probe := it.Value()
	heapMust(t, probe)
	if value != 33 {
		t.Fatal("increment completed prefix", value)
	}
	heapMust(t, r.Close())
}
