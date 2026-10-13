// Source: src/util/mpf.cpp:194-231,233-300; signed-host TLS source release.
package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatFormattedLiteralYoungerSourceStop(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 160; ordinal++ {
		b := arena.NewBudget(1 << 28)
		var m *ArbitraryFloatManager
		var rt *umemory.Runtime
		var lane umemory.Block
		armed, target := false, false
		owner, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
			c := &m.rationalSet
			if armed && c.executing && c.pLive && c.p.Cell != 0 && !m.integer.IsFailureRef(c.p.Cell) && m.integer.Cell(c.p.Cell).Size >= 125 && c.tLive {
				armed = false
				target = true
				b.FailAfter(-1)
				contextTLSNearMinimum(t, rt)
				var e error
				lane, e = rt.Allocate(8)
				serviceMust(t, e)
				delta := int64(-99950) - rt.State().LocalSize
				if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
					t.Fatal("actual TLS release domain", e)
				}
			}
			return f
		}})
		serviceMust(t, e)
		rt = owner
		m, e = NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		var out ArbitraryFloat
		armed = true
		b.FailAfter(ordinal)
		body := m.SetFormattedLiteral(&out, 8, 4000, ArbitraryFloatNearestEven, "1.5p0")
		armed = false
		b.FailAfter(-1)
		if target {
			s := &m.formattedLiteral
			c := &m.rationalSet
			if !errors.Is(body, arena.ErrBounds) || !errors.Is(s.sourceCause, arena.ErrBounds) || !s.qLive || !s.exLive || s.qCursor != 0 || s.q.Num != NewInt(3) || s.q.Den != NewInt(2) || s.ex != NewInt(0) || c.sourceCause == nil {
				t.Fatal("younger stop crossed formatted ex/q", ordinal, body, s.sourceCause, s.q, s.ex, c.sourceCause)
			}
			round := &m.RoundScope
			integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{lane}, &out.significand, &s.ex, &s.q.Num, &s.q.Den, &c.sig.Num, &c.sig.Den, &c.exp, &c.p, &c.p2, &c.prev, &c.t.Num, &c.t.Den, &c.sq.Num, &c.sq.Den, &round.sigma, &round.cap, &round.sticky, &m.rational.NTmp, &m.rational.AddTmp1, &m.rational.AddTmp2, &m.rational.LtTmp1.Num, &m.rational.LtTmp1.Den, &m.rational.LtTmp2.Num, &m.rational.LtTmp2.Den)
			if b.Used() != rt.Stats().TotalBytes+m.OwnedStorageBytes() {
				t.Fatal("formatted stopped source owner graph", ordinal, b.Used(), rt.Stats(), m.OwnedStorageBytes())
			}
			before, child, used := *s, *c, b.Used()
			if e := m.SourceFinalize(); !errors.Is(e, arena.ErrBounds) || *s != before || *c != child || b.Used() != used || m.sourceEnded {
				t.Fatal("source stop crossed class fields", e)
			}
			recovery, e := rt.Allocate(200000)
			serviceMust(t, e)
			serviceMust(t, m.CleanupFormattedLiteral())
			serviceMust(t, m.CleanupValue(&out))
			if s.sourceCause != before.sourceCause || s.bodyCause != before.bodyCause || s.qLive || s.exLive {
				t.Fatal("owning cleanup changed source history")
			}
			contextTLSFinish(t, rt, m.Close)
			for _, block := range []*umemory.Block{&lane, &recovery} {
				contextTLSFinish(t, rt, func() error {
					if *block == (umemory.Block{}) {
						return nil
					}
					return rt.Deallocate(block)
				})
			}
			contextTLSFinish(t, rt, rt.Close)
			if b.Used() != 0 {
				t.Fatal("formatted stop owner refund", b.Used())
			}
			reached = true
			break
		}
		serviceMust(t, m.CleanupFormattedLiteral())
		serviceMust(t, m.CleanupValue(&out))
		if e := m.Close(); e != nil {
			if !m.integer.sourceGuard {
				t.Fatal("pretarget formatted close", ordinal, e)
			}
			serviceMust(t, m.Close())
		}
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("pretarget formatted refund", ordinal, b.Used())
		}
	}
	if !reached {
		t.Fatal("formatted younger source release stop was not reached")
	}
}
