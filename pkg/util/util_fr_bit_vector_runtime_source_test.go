// Source: src/util/bit_vector.h:226-252; src/util/bit_vector.cpp:237-247; src/util/vector.h:102-129,159-167,191-202.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-10-03.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func frBitVectorRuntimeAudit(t *testing.T, r *Runtime, fs ...*RuntimeFastResetBitVector) {
	t.Helper()
	seen := map[Block]bool{}
	var physical, bitmap, cold uint64
	add := func(h Block) {
		if h == (Block{}) {
			return
		}
		if seen[h] {
			t.Fatal("duplicate named owning block", h)
		}
		seen[h] = true
		n, e := r.Size(h)
		heapMust(t, e)
		physical += n + uint64(r.SizeTBits()/8)
		bitmap += ((n + uint64(r.SizeTBits()/8) + 63) / 64) * 8
	}
	for _, f := range fs {
		if f == nil || f.closed {
			continue
		}
		cold += f.ownerBytes
		if !f.base.destroyed && !f.base.pendingReplaces {
			add(f.base.block)
		}
		add(f.base.pending)
		if !f.oneDestroyed && !f.ones.Raw.PendingReplaces {
			add(f.ones.Raw.Block)
		}
		add(f.ones.Raw.Pending)
	}
	st := r.Stats()
	if st.LiveBlocks != uint32(len(seen)) || st.PhysicalBytes != physical || st.InitializationBytes != bitmap || r.Budget().Used() != st.TotalBytes+cold {
		t.Fatal("named derived/base graph", st, len(seen), physical, bitmap, r.Budget().Used(), cold)
	}
}
func TestFastResetBitVectorRuntimeCopyAssignmentAndEveryRefusal(t *testing.T) {
	for route := 0; route < 3; route++ {
		success, failures := 0, 0
		for ordinal := int64(0); ordinal < 96; ordinal++ {
			b := arena.NewBudget(1 << 24)
			r, e := NewRuntime(RuntimeConfig{Budget: b})
			heapMust(t, e)
			a, e := NewRuntimeFastResetBitVector(r, b)
			heapMust(t, e)
			c, e := NewRuntimeFastResetBitVector(r, b)
			heapMust(t, e)
			heapMust(t, a.Push(true))
			heapMust(t, a.Push(true))
			for i := uint32(0); i < 97; i++ {
				heapMust(t, c.Push(i%3 == 0))
			}
			b.FailAfter(ordinal)
			var out *RuntimeFastResetBitVector
			switch route {
			case 0:
				e = a.Push(true)
			case 1:
				out, e = c.Clone(r, b)
			case 2:
				e = a.Assign(c)
			}
			frBitVectorRuntimeAudit(t, r, a, c, out)
			if e != nil {
				failures++
				intHeapActualRefusal(t, r, e)
			} else {
				success++
				if route == 1 {
					heapMust(t, out.Reset())
					if out.base.numBits != 97 {
						t.Fatal("copied base size")
					}
				}
			}
			b.FailAfter(-1)
			for _, f := range []*RuntimeFastResetBitVector{out, a, c} {
				if f != nil {
					if f.stop != nil || f.constructorFailed || f.oneDestroyed || f.ones.Raw.Pending != (Block{}) || f.base.pending != (Block{}) {
						heapMust(t, f.CleanupStopped())
					} else {
						heapMust(t, f.Close())
					}
				}
			}
			heapMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(route, ordinal, b.Used())
			}
			if e == nil {
				t.Logf("FRroute%d firstsuccess%d", route, ordinal)
				break
			}
		}
		if route > 0 && (failures == 0 || success != 1) {
			t.Fatal(route, failures, success)
		}
	}
}

func TestFastResetBitVectorRuntimeSelfAssignmentSourceMemberGuard(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	f, e := NewRuntimeFastResetBitVector(r, b)
	heapMust(t, e)
	heapMust(t, f.Push(true))
	heapMust(t, f.Push(false))
	old := f.ones.Raw.Block
	heapMust(t, f.Assign(f))
	if f.ones.Raw.Block != old {
		t.Fatal("member self assignment changed pointer")
	}
	frBitVectorRuntimeAudit(t, r, f)
	heapMust(t, f.Reset())
	heapMust(t, f.Close())
	heapMust(t, r.Close())
}

func TestFastResetBitVectorRuntimeReverseMemberDestructorStop(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true})
	heapMust(t, e)
	f, e := NewRuntimeFastResetBitVector(r, b)
	heapMust(t, e)
	heapMust(t, f.base.Resize(4096, true))
	// Real member allocations prepare a destructor-order protocol fixture; no
	// failure callback or Runtime counters are modified.
	for {
		c, e := f.ones.Capacity()
		heapMust(t, e)
		if c >= 50000 {
			break
		}
		heapMust(t, f.ones.Raw.ExpandStride(4))
	}
	base := f.base.block
	contextTLSNearMinimum(t, r)
	e = f.SourceFinalize()
	if !errors.Is(e, arena.ErrBounds) || f.ones.Raw.Block != (Block{}) || f.base.block != base || f.sourceStage != 0 {
		t.Fatal("indices destructor before base", e, f.sourceStage)
	}
	if e = f.Push(true); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ordinary mutation after source destructor stop", e)
	}
	frBitVectorRuntimeAudit(t, r, f)
	cause := f.destructorStop
	if e = f.Close(); e != cause {
		t.Fatal(e, cause)
	}
	if e = f.CleanupStopped(); e != nil {
		if !errors.Is(e, arena.ErrBounds) || f.base.block != (Block{}) || f.closed {
			t.Fatal("second actual physical stop", e, f.base.block, f.closed, r.State())
		}
		frBitVectorRuntimeAudit(t, r, f)
		heapMust(t, f.CleanupStopped())
	}

	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestFastResetBitVectorRuntimeFailedAssignmentStillReachesSourceDestructor(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	a, e := NewRuntimeFastResetBitVector(r, b)
	heapMust(t, e)
	c, e := NewRuntimeFastResetBitVector(r, b)
	heapMust(t, e)
	// Equal base capacities make the first refusal the member copy allocation,
	// after source destroys the old one-index vector.
	heapMust(t, a.base.Resize(97, false))
	heapMust(t, a.SetBit(0))
	for i := uint32(0); i < 97; i++ {
		heapMust(t, c.Push(i%3 == 0))
	}
	b.FailAfter(0)
	body := a.Assign(c)
	intHeapActualRefusal(t, r, body)
	if !a.oneDestroyed {
		t.Fatal("member destructive frontier not reached")
	}
	frBitVectorRuntimeAudit(t, r, a, c)
	e = a.SourceFinalize()
	if !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal("actual member destructor guard, not body preflight", e, body)
	}
	if a.stop != body {
		t.Fatal("lost body history")
	}
	frBitVectorRuntimeAudit(t, r, a, c)
	b.FailAfter(-1)
	heapMust(t, a.CleanupStopped())
	heapMust(t, c.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
