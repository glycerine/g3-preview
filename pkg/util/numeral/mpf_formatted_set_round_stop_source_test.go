package numeral

// Source: src/util/mpf.cpp:335-374, src/util/mpf.cpp:1984-2027.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatFormattedSetRoundNUMStopProtectsOlderClass(t *testing.T) {
	found := false
	for offset := int64(-99000); offset >= -99980; offset -= 20 {
		if formattedSetRoundNUMStopRecipe(t, offset) {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("recipe did not reach actual youngest Round NUM DT")
	}
}
func formattedSetRoundNUMStopRecipe(t *testing.T, offset int64) bool {
	b := arena.NewBudget(1 << 28)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	digits := make([]uint32, 63)
	digits[0] = 1
	digits[62] = 1 << 14
	var z, after ArbitraryFloat
	var sig Int
	serviceMust(t, m.integer.SetLimbs(&sig, digits))
	var x, out ArbitraryFloat
	serviceMust(t, m.SetComponents(&x, 8, 2000, false, -5000, &sig))
	serviceMust(t, m.SetFormat(&out, 8, 2001, ArbitraryFloatNearestEven, &x))
	serviceMust(t, m.CleanupFormattedSet())
	// Warm same source arithmetic and restore the actual caller output BIG cell.
	serviceMust(t, m.SetComponents(&out, 8, 2000, false, -5000, &sig))
	serviceMust(t, m.integer.EnsureTmp(0, 80))
	serviceMust(t, m.integer.EnsureTmp(1, 80))
	serviceMust(t, m.MkZero(8, 24, true, &z))
	contextTLSNearMinimum(t, rt)
	lane, e := rt.Allocate(8)
	serviceMust(t, e)
	delta := offset - rt.State().LocalSize
	if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); e != arena.ErrBounds {
		t.Fatal(e)
	}
	var receipt ArbitraryFloatFormattedSetReceipt
	body := m.SetFormatWithReceipt(&out, 8, 2001, ArbitraryFloatNearestEven, &x, &receipt)
	s := &m.RoundScope
	t.Log("actual reached offset", offset, "child", body, "source", s.sourceCause, "flags", s.stickyConstructed, s.capConstructed, s.sigmaConstructed, "parent", m.formattedSet.sourceCause)
	matched := s.sourceCause == arena.ErrBounds && s.stickyConstructed && s.capConstructed && s.sigmaConstructed && m.formattedSet.sourceCause == s.sourceCause && errors.Is(body, s.sourceCause)
	integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{lane}, &sig, &x.significand, &out.significand, &s.sticky, &s.cap, &s.sigma)
	if b.Used() != rt.Stats().TotalBytes+m.OwnedStorageBytes() {
		t.Fatal("before recovery named cold graph")
	}
	if matched {
		used := b.Used()
		sourceEnded := m.sourceEnded
		scope := m.formattedSet
		round := m.RoundScope
		if e = m.SourceFinalize(); e != s.sourceCause || m.sourceEnded != sourceEnded || b.Used() != used || m.formattedSet != scope || m.RoundScope != round {
			t.Fatal("younger stop crossed older owner", e)
		}
	}
	recovery, e := rt.Allocate(200000)
	serviceMust(t, e)
	serviceMust(t, m.CleanupFormattedSetWithReceipt(&receipt, false))
	serviceMust(t, m.CleanupValue(&out))
	serviceMust(t, m.CleanupFormattedSetWithReceipt(&receipt, true))
	if matched {
		h, e := m.formattedSetHistory(&receipt)
		serviceMust(t, e)
		beforeBody, beforeSource, beforeOwn := h.bodyCause, h.sourceCause, h.ownCause
		oldCopy, oldRound, oldFrame, oldInvocation := h.copy, h.roundChild, h.roundFrame, h.roundInvocation
		b.FailAfter(0)
		serviceMust(t, m.SetFormat(&after, 8, 24, ArbitraryFloatNearestEven, &z))
		b.FailAfter(-1)
		current := m.formattedSet
		afterBefore := after
		used := b.Used()
		serviceMust(t, m.CleanupFormattedSetWithReceipt(&receipt, true))
		if h.bodyCause != beforeBody || h.sourceCause != beforeSource || h.ownCause != beforeOwn || h.copy != oldCopy || h.roundChild != oldRound || h.roundFrame != oldFrame || h.roundInvocation != oldInvocation || m.formattedSet != current || after != afterBefore || b.Used() != used {
			t.Fatal("old ACK/history touched later inline invocation")
		}
	}
	serviceMust(t, m.Del(&z))
	serviceMust(t, m.Del(&after))
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.integer.Del(&sig))
	serviceMust(t, m.Close())
	for _, p := range []*umemory.Block{&lane, &recovery} {
		contextTLSFinish(t, rt, func() error {
			if *p == (umemory.Block{}) {
				return nil
			}
			return rt.Deallocate(p)
		})
	}
	contextTLSFinish(t, rt, rt.Close)
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	return matched
}
