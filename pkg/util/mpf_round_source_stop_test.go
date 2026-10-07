// Source: src/util/mpf.cpp:2016-2027,1963-2098; real signed-host TLS guard profile.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatRoundYoungerStickyNUMStopsOlderClass(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	rt, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	digits := make([]uint32, 63)
	digits[0] = 1
	digits[56] = 1 << 8
	digits[62] = 1 << 18
	var sig Int
	serviceMust(t, m.integer.SetLimbs(&sig, digits))
	var x ArbitraryFloat
	serviceMust(t, m.SetComponents(&x, 8, 2000, false, -5000, &sig))
	serviceMust(t, m.Round(ArbitraryFloatNearestEven, &x))
	serviceMust(t, m.CleanupRound())
	serviceMust(t, m.SetComponents(&x, 8, 2000, false, -5000, &sig))
	serviceMust(t, m.integer.EnsureTmp(1, 200))
	contextTLSNearMinimum(t, rt)
	lane, e := rt.Allocate(8)
	serviceMust(t, e)
	delta := int64(-99500) - rt.State().LocalSize
	if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	var receipt ArbitraryFloatRoundReceipt
	returned := m.RoundWithReceipt(ArbitraryFloatNearestEven, &x, &receipt)
	s := &m.RoundScope
	if !errors.Is(returned, arena.ErrBounds) || s.sourceCause == nil || !s.capConstructed || !s.sigmaConstructed {
		t.Fatal("recipe missed younger source NUM stop", returned, s.sourceCause, s.stickyConstructed, s.capConstructed, s.sigmaConstructed, rt.State())
	}
	integerPhysicalAuditWithRaw(t, m.integer, []Block{lane}, &sig, &x.significand, &s.sticky, &s.cap, &s.sigma)
	if b.Used() != rt.Stats().TotalBytes+m.OwnedStorageBytes() {
		t.Fatal("before lift exact physical/cold graph")
	}
	beforeBytes, beforeState, beforeReady := b.Used(), rt.State(), m.rationalReady
	cause := s.sourceCause
	if e = m.SourceFinalize(); e != cause {
		t.Fatal("class source ignored younger round NUM", e, cause)
	}
	if b.Used() != beforeBytes || rt.State() != beforeState || m.rationalReady != beforeReady {
		t.Fatal("younger source stop mutated older class")
	}
	// Recovery is a distinct real allocator action; owning ACK follows completed
	// child suffix, older NUM retirement, and then captured unjoined abort.
	recovery, e := rt.Allocate(200000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return m.CleanupRoundWithReceipt(&receipt, false) })
	contextTLSFinish(t, rt, func() error { return m.CleanupRoundWithReceipt(&receipt, true) })
	body, source, _ := m.RoundInvocationCauses(&receipt)
	if body == nil || source != cause {
		t.Fatal("ACK erased original causes")
	}

	originalHistory, _ := m.roundHistory(&receipt)
	serviceMust(t, m.SetComponents(&x, 8, 2000, false, -5000, &sig))
	var later ArbitraryFloatRoundReceipt
	serviceMust(t, m.RoundWithReceipt(ArbitraryFloatNearestEven, &x, &later))
	used, state, value, invocation := b.Used(), rt.State(), x, m.RoundScope.Invocation
	serviceMust(t, m.CleanupRoundWithReceipt(&receipt, true))
	if b.Used() != used || rt.State() != state || x != value || m.RoundScope.Invocation != invocation || originalHistory.sourceCause != cause {
		t.Fatal("old ACK changed later round or lost immutable history")
	}
	b2, s2, _ := m.RoundInvocationCauses(&receipt)
	if b2 != body || s2 != source {
		t.Fatal("later invocation lost issued source/body tuple")
	}
	contextTLSFinish(t, rt, func() error { return m.Del(&x) })
	contextTLSFinish(t, rt, func() error { return m.integer.Del(&sig) })
	contextTLSFinish(t, rt, m.Close)
	contextTLSFinish(t, rt, func() error {
		if recovery == (Block{}) {
			return nil
		}
		return rt.Deallocate(&recovery)
	})
	contextTLSFinish(t, rt, func() error {
		if lane == (Block{}) {
			return nil
		}
		return rt.Deallocate(&lane)
	})
	serviceMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
