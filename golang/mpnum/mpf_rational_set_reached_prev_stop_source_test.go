// Source: src/util/mpf.cpp:233-300; src/util/mpq.cpp:78-86,312-315; src/util/mpz.h:156-164.
// Licensed under the MIT license. Source authors: Christoph Wintersteiger; Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

// The returning allocation transport lifts that allocation fault and arms the
// signed-host TLS release domain after real Floor quotient publication. It is a helper
// profile, not mathematical OOM or an allocation-before-lift audit.
func TestArbitraryFloatExponentRationalSetReachedPrevNUMStop(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 90; ordinal++ {
		b := arena.NewBudget(1 << 28)
		var m *ArbitraryFloatManager
		var rt *memory.Runtime
		var lane memory.Block
		armed, target := false, false
		rt0, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f memory.Failure) error {
			if armed && m.rationalSet.executing && m.rationalSet.prevLive && m.rationalSet.prev.cell != 0 && !m.rationalSet.p2Live && !m.rationalSet.pLive {
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
		m, e = NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		q := NewRational()
		ex := NewInt(0)
		var out ArbitraryFloat
		d := make([]uint32, 129)
		d[0], d[128] = 3, 2
		serviceMust(t, m.integer.SetInt64(&q.num, 2))
		serviceMust(t, m.integer.SetLimbs(&q.den, d))
		var receipt ArbitraryFloatRationalSetReceipt
		armed = true
		b.FailAfter(ordinal)
		body := m.SetExponentRationalWithReceipt(&out, 8, 24, ArbitraryFloatNearestEven, &ex, &q, &receipt)
		armed = false
		s := &m.rationalSet
		r := &m.roundScope
		if target {
			if s.prevSourceCause != arena.ErrBounds || s.sourceCause != s.prevSourceCause || !s.prevLive || !s.expLive || !s.sigLive || s.p2Live || s.pLive || s.tLive || s.sqLive || !errors.Is(body, s.prevSourceCause) {
				t.Fatalf("reached Floor quotient but source suffix differs ordinal%d body%v prev%v flags prev/exp/sig %v/%v/%v p2/p/t/sq %v/%v/%v/%v", ordinal, body, s.prevSourceCause, s.prevLive, s.expLive, s.sigLive, s.p2Live, s.pLive, s.tLive, s.sqLive)
			}
			// Independent literal branch oracle: below-one input was inverted
			// before Floor; later Round cannot supply this source state.
			cmp, e := m.integer.Compare(&s.sig.num, &q.den)
			serviceMust(t, e)
			if cmp != 0 || s.sig.den != q.num || s.exp != ex {
				t.Fatal("below-one Inv/Prev source branch was not reached")
			}
			integerPhysicalAuditWithRaw(t, m.integer, []memory.Block{lane}, &q.num, &q.den, &out.significand, &r.sigma, &r.cap, &r.sticky, &s.sig.num, &s.sig.den, &s.exp, &s.p, &s.p2, &s.prev, &s.t.num, &s.t.den, &s.sq.num, &s.sq.den, &m.rational.nTmp, &m.rational.addTmp1, &m.rational.addTmp2, &m.rational.ltTmp1.num, &m.rational.ltTmp1.den, &m.rational.ltTmp2.num, &m.rational.ltTmp2.den)
			if b.Used() != rt.Stats().TotalBytes+m.OwnedStorageBytes() {
				t.Fatal("before TLS recovery named physical/cold graph")
			}
			before := *s
			used := b.Used()
			maps := m.completedMaps
			if e = m.SourceFinalize(); e != s.prevSourceCause || *s != before || b.Used() != used || m.completedMaps != maps || m.sourceEnded {
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
			reached = true
			t.Log("actual Floor quotient publication allocation ordinal", ordinal)
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
		t.Fatal("actual Floor quotient allocation refusal not reached")
	}
}
