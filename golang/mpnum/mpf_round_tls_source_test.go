// Source: src/util/mpf.cpp:1984-2027; Runtime signed-host guard helper profile.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestArbitraryFloatRoundActualTLSAllocationReceipt(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
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
	pad, e := rt.Allocate(uint64(99990 - rt.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	body := m.Round(ArbitraryFloatNearestEven, &x)
	if body == nil {
		t.Fatal("actual reached allocation did not stop")
	}
	s := &m.roundScope
	integerPhysicalAuditWithRaw(t, m.integer, []memory.Block{pad}, &sig, &x.significand, &s.sigma, &s.cap, &s.sticky)
	if b.Used() != rt.Stats().TotalBytes+m.OwnedStorageBytes() {
		t.Fatal("before lift exact cold graph")
	}
	pending, e := m.integer.allocator.PendingRoots()
	serviceMust(t, e)
	retained := false
	if block := s.transferred.Block(); block != (memory.Block{}) {
		retained = true
		if s.history == nil || s.history.originalPending != block {
			t.Fatal("transferred allocation lost issued leaf token")
		}
	}
	if s.numeric.cell.object != (memory.ObjectRef{}) {
		retained = true
		if s.history == nil || s.history.originalCell != s.numeric.cell {
			t.Fatal("detached cell lost issued backing token")
		}
	}
	for i, p := range pending {
		if p != (memory.Block{}) {
			retained = true
			if s.numeric.allocator[i] != p {
				t.Fatal("actual reached raw allocation prefix lacks round child", body, p)
			}
		}
	}
	for i := range m.integer.natural.division {
		p := m.integer.natural.division[i].pending
		if p != (memory.Block{}) {
			retained = true
			if s.buffers[i].pending != p {
				t.Fatal("actual reached Natural buffer lacks round child", body, p)
			}
		}
	}
	if !retained {
		t.Fatal("recipe did not reach actual pending prefix", body)
	}
	serviceMust(t, rt.SetMaxSize(0))
	// Force only the later actual owning raw-release synchronization frontier.
	// The body failure and its issued raw token remain immutable diagnostics.
	contextTLSNearMinimum(t, rt)
	lane, e := rt.Allocate(8)
	serviceMust(t, e)
	delta := int64(-99950) - rt.State().LocalSize
	if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); e != arena.ErrBounds {
		t.Fatal(e)
	}
	originalBody := s.bodyCause
	if originalBody == nil || !errors.Is(body, originalBody) {
		t.Fatal("returned error lost original body cause")
	}
	firstOwn := m.CleanupRoundWithReceipt(nil, true)
	if firstOwn != arena.ErrBounds || s.ownCause != firstOwn {
		t.Fatal("actual own raw-release stop not captured", firstOwn, s.ownCause)
	}
	history := s.history
	if history == nil || history.ownCause != firstOwn || history.bodyCause != originalBody {
		t.Fatal("first own/body history lost")
	}
	serviceMust(t, m.CleanupRound())
	if history.ownCause != firstOwn || history.bodyCause != originalBody {
		t.Fatal("own retry erased first cause")
	}
	recovery, e := rt.Allocate(200000)
	serviceMust(t, e)

	serviceMust(t, m.CleanupRound())
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.integer.Del(&sig))
	if e := m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
		t.Fatal(e)
	}
	serviceMust(t, m.Close())
	contextTLSFinish(t, rt, func() error {
		if pad == (memory.Block{}) {
			return nil
		}
		return rt.Deallocate(&pad)
	})
	contextTLSFinish(t, rt, func() error {
		if lane == (memory.Block{}) {
			return nil
		}
		return rt.Deallocate(&lane)
	})
	contextTLSFinish(t, rt, func() error {
		if recovery == (memory.Block{}) {
			return nil
		}
		return rt.Deallocate(&recovery)
	})
	contextTLSFinish(t, rt, rt.Close)
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
