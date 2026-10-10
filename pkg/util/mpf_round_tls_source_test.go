// Source: src/util/mpf.cpp:1984-2027; Runtime signed-host guard helper profile.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatRoundActualTLSAllocationReceipt(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
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
	s := &m.RoundScope
	integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{pad}, &sig, &x.significand, &s.sigma, &s.cap, &s.sticky)
	if b.Used() != rt.Stats().TotalBytes+m.OwnedStorageBytes() {
		t.Fatal("before lift exact cold graph")
	}
	pending, e := m.integer.Allocator.PendingRoots()
	serviceMust(t, e)
	retained := false
	if block := s.Transferred.Block(); block != (umemory.Block{}) {
		retained = true
		if s.history == nil || s.history.originalPending != block {
			t.Fatal("transferred allocation lost issued leaf token")
		}
	}
	if s.Numeric.Cell.Object != (umemory.ObjectRef{}) {
		retained = true
		if s.history == nil || s.history.originalCell != s.Numeric.Cell {
			t.Fatal("detached cell lost issued backing token")
		}
	}
	for i, p := range pending {
		if p != (umemory.Block{}) {
			retained = true
			if s.Numeric.Allocator[i] != p {
				t.Fatal("actual reached raw allocation prefix lacks round child", body, p)
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
		if pad == (umemory.Block{}) {
			return nil
		}
		return rt.Deallocate(&pad)
	})
	contextTLSFinish(t, rt, func() error {
		if lane == (umemory.Block{}) {
			return nil
		}
		return rt.Deallocate(&lane)
	})
	contextTLSFinish(t, rt, func() error {
		if recovery == (umemory.Block{}) {
			return nil
		}
		return rt.Deallocate(&recovery)
	})
	contextTLSFinish(t, rt, rt.Close)
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
