// Source: src/util/mpf.cpp:108-112,186-192,233-300.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func TestArbitraryFloatSignedFractionYoungerSourceStop(t *testing.T) {
	for _, childFirst := range []bool{false, true} {
		signedFractionYoungerSourceStop(t, childFirst)
	}
}
func signedFractionYoungerSourceStop(t *testing.T, childFirst bool) {
	reached := false
	for ordinal := int64(0); ordinal < 120; ordinal++ {
		b := arena.NewBudget(1 << 28)
		var m *ArbitraryFloatManager
		var r *Runtime
		var lane Block
		armed, target := false, false
		rt, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f Failure) error {
			if armed && m.rationalSet.executing && m.rationalSet.pLive && m.rationalSet.p.Cell != 0 && !m.integer.IsFailureRef(m.rationalSet.p.Cell) && m.integer.Cell(m.rationalSet.p.Cell).Size >= 125 && m.rationalSet.tLive {
				armed = false
				target = true
				b.FailAfter(-1)
				contextTLSNearMinimum(t, r)
				var e error
				lane, e = r.Allocate(8)
				serviceMust(t, e)
				delta := int64(-99950) - r.State().LocalSize
				if e = r.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
					t.Fatal("real TLS realloc", e)
				}
			}
			return f
		}})
		serviceMust(t, e)
		r = rt
		m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		var out ArbitraryFloat
		armed = true
		b.FailAfter(ordinal)
		body := m.SetFractionSigned(&out, 8, 4000, ArbitraryFloatNearestEven, 3, 2)
		armed = false
		b.FailAfter(-1)
		if target {
			s := &m.signedFraction
			c := &m.rationalSet
			if !errors.Is(body, arena.ErrBounds) || s.sourceCause == nil || !errors.Is(s.sourceCause, arena.ErrBounds) || !s.qLive || s.qCursor != 0 || s.q.Num != NewInt(3) || s.q.Den != NewInt(2) || c.sourceCause == nil || m.plainRationalSet.sourceCause == nil || !m.plainRationalSet.expLive {
				t.Fatal("younger source stop crossed rational", ordinal, body, s.sourceCause, c.sourceCause, s.qCursor)
			}
			signedFractionPhysicalAudit(t, m, &out, []Block{lane})
			before, child, plain, used, maps := *s, *c, m.plainRationalSet, b.Used(), m.completedMaps
			if e = m.SourceFinalize(); !errors.Is(e, arena.ErrBounds) || s.q != before.q || s.qLive != before.qLive || s.qCursor != before.qCursor || s.sourceCause != before.sourceCause || s.child != before.child || *c != child || m.plainRationalSet != plain || b.Used() != used || m.completedMaps != maps || m.sourceEnded {
				t.Fatal("source stop crossed older class", e)
			}
			recovery, e := r.Allocate(200000)
			serviceMust(t, e)
			if childFirst {
				serviceMust(t, m.CleanupPlainRationalSetWithReceipt(&s.child, true))
				serviceMust(t, m.CleanupValue(&out))
				zero := NewRational()
				serviceMust(t, m.SetRational(&out, 8, 24, ArbitraryFloatNearestEven, &zero))
				later, laterUsed := m.plainRationalSet, b.Used()
				serviceMust(t, m.CleanupSignedFraction())
				if m.plainRationalSet != later || b.Used() != laterUsed {
					t.Fatal("old parent adopted later plain child")
				}
			} else {
				serviceMust(t, m.CleanupSignedFraction())
			}
			serviceMust(t, m.CleanupValue(&out))
			if s.sourceCause != before.sourceCause || s.bodyCause != before.bodyCause || s.qLive {
				t.Fatal("owning cleanup changed source history")
			}
			serviceMust(t, m.SetFractionSigned(&out, 8, 24, ArbitraryFloatNearestEven, 0, 1))
			later, laterUsed := m.plainRationalSet, b.Used()
			serviceMust(t, m.CleanupSignedFraction())
			if m.plainRationalSet != later || b.Used() != laterUsed {
				t.Fatal("ended parent touched later zero child")
			}
			serviceMust(t, m.CleanupValue(&out))
			contextTLSFinish(t, r, m.Close)
			for _, block := range []*Block{&lane, &recovery} {
				contextTLSFinish(t, r, func() error {
					if *block == (Block{}) {
						return nil
					}
					return r.Deallocate(block)
				})
			}
			contextTLSFinish(t, r, r.Close)
			if b.Used() != 0 {
				t.Fatal("source stop refunds", b.Used())
			}
			reached = true
			break
		}
		serviceMust(t, m.CleanupSignedFraction())
		serviceMust(t, m.CleanupValue(&out))
		if e = m.Close(); e != nil {
			if !m.integer.sourceGuard {
				t.Fatal(e)
			}
			signedFractionScratchHole(t, m)
			serviceMust(t, m.Close())
		}
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("pretarget refunds", ordinal, b.Used())
		}
	}
	if !reached {
		t.Fatal("actual younger source release stop was not reached")
	}
}
