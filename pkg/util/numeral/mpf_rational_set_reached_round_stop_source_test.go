package numeral

// Source: src/util/mpf.cpp:233-300,1984-2027; src/util/mpz.h:156-164.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// The returning allocation transport lifts that allocation fault and arms the
// signed-host TLS release domain after real sticky publication. It is a helper
// profile, not mathematical OOM or an allocation-before-lift audit.
func TestArbitraryFloatExponentRationalSetReachedRoundNUMStop(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 90; ordinal++ {
		b := arena.NewBudget(1 << 28)
		var m *ArbitraryFloatManager
		var rt *umemory.Runtime
		var lane umemory.Block
		armed, target := false, false
		rt0, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
			if armed && m.RoundScope.executing && m.RoundScope.stickyConstructed && m.RoundScope.sticky.Cell != 0 {
				armed = false
				target = true
				b.FailAfter(-1)
				contextTLSNearMinimum(t, rt)
				var e error
				lane, e = rt.Allocate(8)
				serviceMust(t, e)
				delta := int64(-99980) - rt.State().LocalSize
				if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); e != arena.ErrBounds {
					t.Fatal(e)
				}
			}
			return f
		}})
		serviceMust(t, e)
		rt = rt0
		m, e = NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		q := NewRational()
		ex := NewInt(-5000)
		var out ArbitraryFloat
		serviceMust(t, m.rational.SetFraction(&q, 13, 8))
		serviceMust(t, m.SetExponentRational(&out, 8, 2001, ArbitraryFloatNearestEven, &ex, &q))
		serviceMust(t, m.CleanupExponentRationalSet())
		serviceMust(t, m.CleanupRound())
		var receipt ArbitraryFloatRationalSetReceipt
		armed = true
		b.FailAfter(ordinal)
		body := m.SetExponentRationalWithReceipt(&out, 8, 2001, ArbitraryFloatNearestEven, &ex, &q, &receipt)
		armed = false
		s := &m.rationalSet
		r := &m.RoundScope
		if target {
			if r.sourceCause != arena.ErrBounds || !r.stickyConstructed || !r.capConstructed || !r.sigmaConstructed || s.sourceCause != r.sourceCause || !s.sqLive || !s.tLive || !s.pLive || !s.expLive || !s.sigLive || !errors.Is(body, r.sourceCause) {
				t.Fatalf("target reached but suffix differs ordinal%d body%v child%v flags%v/%v/%v parent%v", ordinal, body, r.sourceCause, r.stickyConstructed, r.capConstructed, r.sigmaConstructed, s.sourceCause)
			}
			integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{lane}, &q.Num, &q.Den, &out.significand, &r.sigma, &r.cap, &r.sticky, &s.sig.Num, &s.sig.Den, &s.exp, &s.p, &s.p2, &s.prev, &s.t.Num, &s.t.Den, &s.sq.Num, &s.sq.Den, &m.rational.NTmp, &m.rational.AddTmp1, &m.rational.AddTmp2, &m.rational.LtTmp1.Num, &m.rational.LtTmp1.Den, &m.rational.LtTmp2.Num, &m.rational.LtTmp2.Den)
			if b.Used() != rt.Stats().TotalBytes+m.OwnedStorageBytes() {
				t.Fatal("before TLS recovery named physical/cold graph")
			}
			before := *s
			used := b.Used()
			if e = m.SourceFinalize(); e != r.sourceCause || *s != before || b.Used() != used || m.sourceEnded {
				t.Fatal("young child crossed older caller/class suffix", e)
			}
			recovery, e := rt.Allocate(200000)
			serviceMust(t, e)
			serviceMust(t, m.CleanupExponentRationalSetWithReceipt(&receipt, false))
			serviceMust(t, m.CleanupValue(&out))
			serviceMust(t, m.CleanupExponentRationalSetWithReceipt(&receipt, true))
			serviceMust(t, m.rational.Del(&q))
			serviceMust(t, m.Close())
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
			reached = true
			t.Log("actual sticky publication allocation ordinal", ordinal)
			break
		}
		b.FailAfter(-1)
		serviceMust(t, m.CleanupExponentRationalSet())
		serviceMust(t, m.CleanupRound())
		serviceMust(t, m.CleanupValue(&out))
		serviceMust(t, m.rational.Del(&q))
		if e = m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
			t.Fatal(e)
		}
		serviceMust(t, m.Close())
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal(ordinal, b.Used())
		}
	}
	if !reached {
		t.Fatal("actual sticky published allocation refusal not reached")
	}
}
