// Source: src/util/mpf.cpp:1572-1576; actual scoped q and child invocations.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatRationalStringEveryColdReservation(t *testing.T) {
	reachedNumericFailure := false
	for _, sb := range []uint32{2, 24, 120} {
		for _, exp := range []int64{-160, 160} {
			complete := false
			for ordinal := int64(0); ordinal < 512; ordinal++ {
				m, r, b := mpfNumeralHelpersFixture(t)
				x := ArbitraryFloat{format: arbitraryFloatFormat(17, sb, true), exponent: exp, significand: NewInt(1)}
				serviceMust(t, m.integer.Mul2kInPlace(&x.significand, sb-1))
				serviceMust(t, m.integer.Dec(&x.significand))
				before := x
				var out OwnedString
				b.FailAfter(ordinal)
				body := m.ToRationalString(&x, &out)
				s, c := &m.rationalString, &m.rationalConversion
				integerPhysicalAudit(t, m.integer, &x.significand, &s.q.Num, &s.q.Den, &c.a.significand, &c.n, &c.d, &m.rational.NTmp)
				callerBytes := uint64(0)
				if body == nil {
					callerBytes = out.StorageBytes()
				}
				if x != before || b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes+callerBytes {
					t.Fatal("immediate rational string graph", sb, exp, ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats(), callerBytes)
				}
				if body != nil && !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected wrapper failure", sb, exp, ordinal, body)
				}
				if body != nil {
					reachedNumericFailure = true
					var later OwnedString
					if e := m.ToRationalString(&x, &later); !errors.Is(e, ErrNumericPending) {
						t.Fatal("stopped wrapper replay", e)
					}
				}
				b.FailAfter(-1)
				serviceMust(t, m.CleanupRationalString())
				out.OwnAbort()
				serviceMust(t, m.CleanupValue(&x))
				if e := m.Close(); e != nil {
					z := m.integer
					if !errors.Is(e, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
						t.Fatal("unexpected source stop", e)
					}
					index := int(z.SourceFinalizeStage-2) / 2
					ref := z.Tmp[index]
					if z.SourceFinalizeStage%2 == 1 {
						ref = z.Arg[index]
					}
					if ref != 0 || s.active {
						t.Fatal("unproved source scratch hole", ref)
					}
					serviceMust(t, m.Close())
				}
				serviceMust(t, r.Close())
				if b.Used() != 0 {
					t.Fatal("wrapper refunds", b.Used())
				}
				if body == nil {
					complete = true
					break
				}
			}
			if !complete {
				t.Fatal("wrapper reservation horizon", sb, exp)
			}
		}
	}
	if !reachedNumericFailure {
		t.Fatal("no actual numerical conversion failure exercised")
	}
}

func TestArbitraryFloatRationalStringWarmAndLaterChildren(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	var x ArbitraryFloat
	serviceMust(t, m.SetDouble(&x, 8, 24, 1.5))
	var first OwnedString
	serviceMust(t, m.ToRationalString(&x, &first))
	first.SourceDestroy()
	used, state := b.Used(), r.State()
	b.FailAfter(0)
	var out OwnedString
	serviceMust(t, m.ToRationalString(&x, &out))
	if !out.Matches("3/2") || b.Used() != used || r.State() != state {
		t.Fatal("warm inline wrapper allocated", b.Used(), used)
	}
	b.FailAfter(-1)
	q := NewRational()
	serviceMust(t, m.ToRational(&x, m.rational, &q))
	var later OwnedString
	serviceMust(t, m.rational.ToOwnedString(&q, &later))
	conv, format := m.rationalConversion.invocation, m.rational.stringScope.invocation
	serviceMust(t, m.CleanupRationalString())
	if m.rationalConversion.invocation != conv || m.rational.stringScope.invocation != format || !out.Matches("3/2") || !later.Matches("3/2") {
		t.Fatal("completed wrapper touched later children")
	}
	out.SourceDestroy()
	later.SourceDestroy()
	serviceMust(t, m.rational.Del(&q))
	serviceMust(t, m.CleanupRationalConversion())
	serviceMust(t, m.Del(&x))
}

func TestArbitraryFloatRationalStringRefusedChildKeepsOlderInvocation(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 60; ordinal++ {
		b := arena.NewBudget(1 << 28)
		m, e := NewArbitraryFloatManager(IntegerConfig{Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
		serviceMust(t, e)
		sig := NewInt(1)
		var x ArbitraryFloat
		serviceMust(t, m.SetComponents(&x, 8, 4000, false, 0, &sig))
		_, e = m.powerTwoWrapper(3999, false)
		serviceMust(t, e)
		q := NewRational()
		b.FailAfter(ordinal)
		body := m.ToRational(&x, m.rational, &q)
		c := &m.rationalConversion
		matched := body != nil && c.active && c.numericPhase == 2 && c.numeric.Cell.Object != (ObjectRef{})
		b.FailAfter(-1)
		if matched {
			reached = true
			invocation, numeric, transferred, a, n, d, used := c.invocation, c.numeric, c.transferred, c.a, c.n, c.d, b.Used()
			var out OwnedString
			if e := m.ToRationalString(&x, &out); !errors.Is(e, ErrNumericPending) {
				t.Fatal("older child refusal", e)
			}
			if m.rationalString.conversion != 0 || m.rationalString.format != 0 {
				t.Fatal("refused child was adopted")
			}
			serviceMust(t, m.CleanupRationalString())
			if !c.active || c.invocation != invocation || c.numeric != numeric || c.transferred != transferred || c.a != a || c.n != n || c.d != d || b.Used() != used || out.Self != nil {
				t.Fatal("wrapper cleanup changed unrelated older child")
			}
			integerPhysicalAudit(t, m.integer, &x.significand, &q.Num, &q.Den, &c.a.significand, &c.n, &c.d, &m.rational.NTmp)
		}
		serviceMust(t, m.CleanupRationalConversion())
		serviceMust(t, m.rational.Del(&q))
		serviceMust(t, m.CleanupValue(&x))
		if source := m.SourceFinalize(); source != nil {
			z := m.integer
			if !errors.Is(source, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
				t.Fatal(source)
			}
			index := int(z.SourceFinalizeStage-2) / 2
			ref := z.Tmp[index]
			if z.SourceFinalizeStage%2 == 1 {
				ref = z.Arg[index]
			}
			if ref != 0 {
				t.Fatal("unproved source scratch hole", ref)
			}
		}
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal("older invocation refunds", b.Used())
		}
		if matched {
			break
		}
	}
	if !reached {
		t.Fatal("no actual retained unpack allocation reached")
	}
}
