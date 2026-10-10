// Source: src/util/mpf.cpp:108-112; src/util/mpz.cpp:560-817.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func TestArbitraryFloatSignedFractionActualConversionTransport(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	var out ArbitraryFloat
	armed, calls := false, 0
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if armed {
			calls++
			s := &m.signedFraction
			if !s.executing || !s.qLive || !s.normalizationOwned || s.q.Num != NewInt(3) || s.q.Den != NewInt(2) || s.childFrame != &m.plainRationalSet || s.childInvocation != m.plainRationalSet.invocation {
				t.Fatal("actual conversion prefix")
			}
			before, old, used := *s, out, b.Used()
			for i, call := range []func() error{m.Close, m.SourceFinalize, m.CleanupSignedFraction, m.integer.CloseChecked, m.integer.SourceFinalize, m.rational.CloseChecked, m.rational.SourceFinalize, func() error { return m.SetFractionSigned(&out, 8, 4000, ArbitraryFloatNearestEven, 3, 2) }, func() error { return m.SetInt32(&out, 8, 24, 7) }, func() error { return m.integer.Del(&s.q.Num) }} {
				if e := call(); !errors.Is(e, ErrNumericPending) {
					t.Fatal("fraction transport guard", i, e)
				}
			}
			if s.q != before.q || s.qLive != before.qLive || s.qCursor != before.qCursor || s.invocation != before.invocation || s.child != before.child || s.childInvocation != before.childInvocation || s.numeric != before.numeric || s.gcd != before.gcd || s.division != before.division || out != old || b.Used() != used {
				t.Fatal("transport changed prefix")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	pad, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	armed = true
	body := m.SetFractionSigned(&out, 8, 4000, ArbitraryFloatNearestEven, 3, 2)
	armed = false
	if !errors.Is(body, umemory.Failure{Kind: umemory.FailureMemout}) || calls != 1 {
		t.Fatal("conversion transport missing", body, calls)
	}
	signedFractionPhysicalAudit(t, m, &out, []umemory.Block{pad})
	serviceMust(t, r.SetMaxSize(0))
	serviceMust(t, m.CleanupSignedFraction())
	serviceMust(t, m.CleanupValue(&out))
	serviceMust(t, m.Close())
	serviceMust(t, r.Deallocate(&pad))
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("transport refunds", b.Used())
	}
}

func TestArbitraryFloatSignedFractionRefusedOlderChild(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 150; ordinal++ {
		b := arena.NewBudget(1 << 28)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		q := NewRational()
		var older, out ArbitraryFloat
		serviceMust(t, m.rational.SetFractionSignedSource(&q, 3, 2))
		b.FailAfter(ordinal)
		body := m.SetRational(&older, 8, 4000, ArbitraryFloatNearestEven, &q)
		b.FailAfter(-1)
		if body != nil && !numericPhysicalReservationRefusal(body) {
			t.Fatal(body)
		}
		if m.rationalSet.active {
			reached = true
			child, used := m.rationalSet, b.Used()
			if e := m.SetFractionSigned(&out, 8, 24, ArbitraryFloatNearestEven, 3, 2); !errors.Is(e, ErrNumericPending) {
				t.Fatal("old exponent refusal", ordinal, e)
			}
			s := &m.signedFraction
			if s.childInvocation == 0 || s.exponentInvocation != 0 || s.child.frame != nil || m.rationalSet != child || b.Used() != used {
				t.Fatal("refused old child adopted", ordinal)
			}
			serviceMust(t, m.CleanupSignedFraction())
			if m.rationalSet != child || b.Used() != used {
				t.Fatal("parent cleanup touched refused exponent", ordinal)
			}
		}
		serviceMust(t, m.CleanupPlainRationalSet())
		serviceMust(t, m.CleanupValue(&older))
		serviceMust(t, m.CleanupValue(&out))
		serviceMust(t, m.rational.Del(&q))
		if e = m.Close(); e != nil {
			if !m.integer.sourceGuard {
				t.Fatal(e)
			}
			signedFractionScratchHole(t, m)
			serviceMust(t, m.Close())
		}
		if b.Used() != 0 {
			t.Fatal("older child refunds", ordinal, b.Used())
		}
		if reached {
			break
		}
		if body == nil {
			break
		}
	}
	if !reached {
		t.Fatal("no actual retained older exponent child")
	}
}

func TestArbitraryFloatSignedFractionRefusedNormalizationKeepsOlderGCD(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 100; ordinal++ {
		b := arena.NewBudget(1 << 28)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		z := m.integer
		a, d, g := NewInt(1), NewInt(1), NewInt(0)
		serviceMust(t, z.Mul2kInPlace(&a, 4000))
		serviceMust(t, z.Mul2kInPlace(&d, 2000))
		serviceMust(t, z.Inc(&d))
		b.FailAfter(ordinal)
		body := z.GCD(&a, &d, &g)
		b.FailAfter(-1)
		old := &z.GcdRaw
		if body != nil && old.Active && NumericAllocationPending(&old.Numeric) {
			reached = true
			roots, values, inv, used := old.Numeric, old.Values, old.Invocation, b.Used()
			var out ArbitraryFloat
			if e := m.SetFractionSigned(&out, 8, 24, ArbitraryFloatNearestEven, -2147483648, 3); !errors.Is(e, ErrNumericPending) {
				t.Fatal("normalization refusal", ordinal, e)
			}
			if m.signedFraction.gcd.Invocation != 0 || m.signedFraction.childInvocation != 0 {
				t.Fatal("refused GCD adopted")
			}
			serviceMust(t, m.CleanupSignedFraction())
			if !old.Active || old.Numeric != roots || old.Values != values || old.Invocation != inv || b.Used() != used {
				t.Fatal("wrapper changed older GCD roots", ordinal)
			}
			signedFractionPhysicalAudit(t, m, &out, nil, &a, &d, &g)
		}
		serviceMust(t, z.CleanupGCDRaw())
		for _, v := range []*Int{&a, &d, &g} {
			serviceMust(t, z.CleanupRawNumeral(v))
		}
		if e = m.Close(); e != nil {
			signedFractionScratchHole(t, m)
			serviceMust(t, m.Close())
		}
		if b.Used() != 0 {
			t.Fatal("older GCD refunds", ordinal, b.Used())
		}
		if reached {
			break
		}
		if body == nil {
			break
		}
	}
	if !reached {
		t.Fatal("actual retained older GCD roots not reached")
	}
}

func TestArbitraryFloatSignedFractionInlineChildKeepsOlderHistory(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 80; ordinal++ {
		b := arena.NewBudget(1 << 28)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		q := NewRational()
		var older, out ArbitraryFloat
		ex := NewInt(0)
		var receipt ArbitraryFloatRationalSetReceipt
		serviceMust(t, m.rational.SetFractionSignedSource(&q, 3, 2))
		serviceMust(t, m.MkZero(8, 4, false, &out))
		b.FailAfter(ordinal)
		body := m.SetExponentRationalWithReceipt(&older, 8, 4000, ArbitraryFloatNearestEven, &ex, &q, &receipt)
		b.FailAfter(-1)
		h := m.rationalSet.history
		if body != nil && h != nil && !m.rationalSet.active && (h.Numeric.Cell.Object != (umemory.ObjectRef{}) || h.Transferred.Block() != (umemory.Block{})) {
			reached = true
			before := *h
			serviceMust(t, m.SetFractionSigned(&out, 8, 4, ArbitraryFloatNearestEven, 3, 2))
			if m.signedFraction.child.frame != nil || m.signedFraction.exponentInvocation == 0 || *h != before || h.owned {
				t.Fatal("inline wrapper swept unrelated history", ordinal)
			}
			serviceMust(t, m.CleanupSignedFraction())
			if *h != before {
				t.Fatal("ended wrapper changed older history", ordinal)
			}
			serviceMust(t, m.CleanupExponentRationalSetWithReceipt(&receipt, true))
		}
		serviceMust(t, m.CleanupExponentRationalSet())
		serviceMust(t, m.CleanupValue(&older))
		serviceMust(t, m.CleanupValue(&out))
		serviceMust(t, m.rational.Del(&q))
		if e = m.Close(); e != nil {
			signedFractionScratchHole(t, m)
			serviceMust(t, m.Close())
		}
		if b.Used() != 0 {
			t.Fatal("inline history refunds", ordinal, b.Used())
		}
		if reached {
			break
		}
		if body == nil {
			break
		}
	}
	if !reached {
		t.Fatal("actual source-completed retained history not reached")
	}
}
