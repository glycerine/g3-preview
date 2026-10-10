// Source: src/util/mpf.cpp:233-300; src/util/mpz.h:156-164.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatExponentRationalSetInlineZeroRetainsActualLeaf(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 40; ordinal++ {
		b := arena.NewBudget(1 << 28)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		q := NewRational()
		serviceMust(t, m.rational.SetFraction(&q, 3, 2))
		zero := NewRational()
		ex := NewInt(0)
		var out ArbitraryFloat
		serviceMust(t, m.MkZero(8, 24, false, &out))
		var r ArbitraryFloatRationalSetReceipt
		b.FailAfter(ordinal)
		body := m.SetExponentRationalWithReceipt(&out, 8, 4000, ArbitraryFloatNearestEven, &ex, &q, &r)
		s := &m.rationalSet
		h := s.history
		matched := body != nil && h != nil && (h.Numeric.Cell.Object != (umemory.ObjectRef{}) || h.Transferred.Block() != (umemory.Block{})) && !s.active
		if matched {
			reached = true
			used := b.Used()
			saved := *s
			ctx := s.allocationContext()
			wantBody, wantSource, wantOwn := m.ExponentRationalSetInvocationCauses(&r)
			if wantBody == nil || h.issued != r {
				t.Fatal("missing actual issued refusal history")
			}
			b.FailAfter(0)
			for i := 0; i < 3; i++ {
				var erased ArbitraryFloatRationalSetReceipt
				serviceMust(t, m.SetExponentRationalWithReceipt(&out, 8, 24, ArbitraryFloatNearestEven, &ex, &zero, &erased))
				if erased != (ArbitraryFloatRationalSetReceipt{}) || *s != saved || b.Used() != used {
					t.Fatal("erased route acquired/erased old frame")
				}
			}
			if ctx.roots(m.integer) != nil {
				t.Fatal("inactive saved context reacquired")
			}
			copied := *h
			forged := r
			forged.frame = &copied
			if e := m.CleanupExponentRationalSetWithReceipt(&forged, true); e != umemory.ErrRuntimeOwner {
				t.Fatal("copied actual leaf accepted", e)
			}
			b.FailAfter(-1)
			serviceMust(t, m.CleanupExponentRationalSetWithReceipt(&r, false))
			serviceMust(t, m.Del(&out))
			serviceMust(t, m.CleanupExponentRationalSetWithReceipt(&r, true))
			gotBody, gotSource, gotOwn := m.ExponentRationalSetInvocationCauses(&r)
			if gotBody != wantBody || gotSource != wantSource || gotOwn != wantOwn {
				t.Fatal("own ACK erased immutable history")
			}
			// A later finite body may reuse the embedded lexical frame, while
			// the earlier allocating call remains in its charged issued record.
			var next ArbitraryFloatRationalSetReceipt
			serviceMust(t, m.SetExponentRationalWithReceipt(&out, 8, 24, ArbitraryFloatNearestEven, &ex, &q, &next))
			serviceMust(t, m.CleanupExponentRationalSet())
			warmUsed := b.Used()
			b.FailAfter(0)
			for i := 0; i < 3; i++ {
				var inline ArbitraryFloatRationalSetReceipt
				serviceMust(t, m.SetExponentRationalWithReceipt(&out, 8, 24, ArbitraryFloatNearestEven, &ex, &q, &inline))
				if inline != (ArbitraryFloatRationalSetReceipt{}) || b.Used() != warmUsed {
					t.Fatal("warm finite SET acquired entry history")
				}
				serviceMust(t, m.CleanupExponentRationalSet())
			}
			b.FailAfter(-1)
			current, nowUsed := m.rationalSet, b.Used()
			if ctx.roots(m.integer) != nil {
				t.Fatal("saved old context retargeted to later finite invocation")
			}
			serviceMust(t, m.CleanupExponentRationalSetWithReceipt(&r, true))
			gotBody, gotSource, gotOwn = m.ExponentRationalSetInvocationCauses(&r)
			if m.rationalSet != current || b.Used() != nowUsed || gotBody != wantBody || gotSource != wantSource || gotOwn != wantOwn {
				t.Fatal("stale ACK changed later frame or old call history")
			}

		}
		b.FailAfter(-1)
		serviceMust(t, m.CleanupExponentRationalSet())
		serviceMust(t, m.CleanupRound())
		serviceMust(t, m.Del(&out))
		serviceMust(t, m.rational.Del(&q))
		if e = m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
			t.Fatal(e)
		}
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(ordinal, b.Used())
		}
		if reached {
			break
		}
	}
	if !reached {
		t.Fatal("actual source-completed direct refusal not reached")
	}
}
