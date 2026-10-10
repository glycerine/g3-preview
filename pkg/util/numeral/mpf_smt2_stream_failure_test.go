// Source: src/util/mpf.cpp:1584-1588; actual scoped q and child invocations.
package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestArbitraryFloatSMT2StreamEveryColdReservation(t *testing.T) {
	reachedFormatFailure := false
	for _, sb := range []uint32{2, 24, 120} {
		for _, exp := range []int64{-160, 160} {
			complete := false
			for ordinal := int64(0); ordinal < 512; ordinal++ {
				m, r, b := mpfNumeralHelpersFixture(t)
				x := ArbitraryFloat{format: arbitraryFloatFormat(17, sb, true), exponent: exp, significand: NewInt(1)}
				serviceMust(t, m.integer.Mul2kInPlace(&x.significand, sb-1))
				serviceMust(t, m.integer.Dec(&x.significand))
				before := x
				var out utext.OwnedStringStream
				serviceMust(t, out.Construct(b))
				b.FailAfter(ordinal)
				body := m.DisplaySMT2Into(&out, &x, true)
				s, c := &m.smt2Stream, &m.rationalConversion
				integerPhysicalAudit(t, m.integer, &x.significand, &s.q.Num, &s.q.Den, &c.a.significand, &c.n, &c.d, &m.rational.NTmp, &m.integer.smt2Stream.tmp)
				callerBytes := out.Buffer.String.StorageBytes()
				if x != before || b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes+callerBytes {
					t.Fatal("immediate SMT2 wrapper graph", sb, exp, ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats(), callerBytes)
				}
				if body != nil && !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected wrapper failure", sb, exp, ordinal, body)
				}
				if body != nil {
					reachedFormatFailure = reachedFormatFailure || s.format != 0
					var later utext.OwnedStringStream
					serviceMust(t, later.Construct(b))
					if e := m.DisplaySMT2Into(&later, &x, true); !errors.Is(e, ErrNumericPending) {
						t.Fatal("stopped wrapper replay", e)
					}
				}
				b.FailAfter(-1)
				serviceMust(t, m.CleanupSMT2Stream())
				out.Buffer.OwnAbort()
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
	if !reachedFormatFailure {
		t.Fatal("no actual rational formatter failure exercised")
	}
}

func TestArbitraryFloatSMT2StreamRefusedChildKeepsOlderInvocation(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 60; ordinal++ {
		b := arena.NewBudget(1 << 28)
		m, e := NewArbitraryFloatManager(IntegerConfig{Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
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
		matched := body != nil && c.active && c.numericPhase == 2 && c.numeric.Cell.Object != (umemory.ObjectRef{})
		b.FailAfter(-1)
		if matched {
			reached = true
			invocation, numeric, transferred, a, n, d, used := c.invocation, c.numeric, c.transferred, c.a, c.n, c.d, b.Used()
			var out utext.OwnedStringStream
			serviceMust(t, out.Construct(b))
			if e := m.DisplaySMT2Into(&out, &x, true); !errors.Is(e, ErrNumericPending) {
				t.Fatal("older child refusal", e)
			}
			if m.smt2Stream.conversion != 0 || m.smt2Stream.format != 0 {
				t.Fatal("refused child was adopted")
			}
			serviceMust(t, m.CleanupSMT2Stream())
			if !c.active || c.invocation != invocation || c.numeric != numeric || c.transferred != transferred || c.a != a || c.n != n || c.d != d || b.Used() != used || out.Buffer.Position != 0 {
				t.Fatal("wrapper cleanup changed unrelated older child")
			}
			integerPhysicalAudit(t, m.integer, &x.significand, &q.Num, &q.Den, &c.a.significand, &c.n, &c.d, &m.rational.NTmp, &m.integer.smt2Stream.tmp)
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

func TestArbitraryFloatSMT2StreamRefusedFormatterKeepsOlderCopy(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 60; ordinal++ {
		b := arena.NewBudget(1 << 28)
		m, e := NewArbitraryFloatManager(IntegerConfig{Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
		serviceMust(t, e)
		var x ArbitraryFloat
		serviceMust(t, m.SetDouble(&x, 8, 24, 1.5))
		var warm utext.OwnedStringStream
		serviceMust(t, warm.Construct(b))
		serviceMust(t, m.DisplaySMT2Into(&warm, &x, false))
		warm.Buffer.OwnAbort()
		a := Rational{Num: NewInt(-1), Den: NewInt(3)}
		serviceMust(t, m.integer.Mul2kInPlace(&a.Num, 4000))
		var older utext.OwnedStringStream
		serviceMust(t, older.Construct(b))
		b.FailAfter(ordinal)
		body := m.rational.DisplaySMT2Into(&older, &a, false)
		matched := body != nil && m.rational.smt2Stream.active && m.integer.CopyAllocation.Roots.Cell.Object != (umemory.ObjectRef{})
		b.FailAfter(-1)
		if matched {
			reached = true
			oldFormat, oldCopy, roots, tmp, used := m.rational.smt2Stream.invocation, m.integer.smt2Stream.invocation, m.integer.CopyAllocation.Roots, m.integer.smt2Stream.tmp, b.Used()
			var out utext.OwnedStringStream
			serviceMust(t, out.Construct(b))
			if e := m.DisplaySMT2Into(&out, &x, false); !errors.Is(e, ErrNumericPending) {
				t.Fatal("older formatter refusal", e)
			}
			s := &m.smt2Stream
			if s.format != 0 || !s.conversionOwned || s.qLive {
				t.Fatal("refused formatter was adopted or conversion did not finish")
			}
			serviceMust(t, m.CleanupSMT2Stream())
			if m.rational.smt2Stream.invocation != oldFormat || !m.rational.smt2Stream.active || m.integer.smt2Stream.invocation != oldCopy || m.integer.CopyAllocation.Roots != roots || m.integer.smt2Stream.tmp != tmp || b.Used() != used || out.Buffer.Position != 0 {
				t.Fatal("wrapper joined unrelated older formatter")
			}
			out.Buffer.OwnAbort()
			integerPhysicalAudit(t, m.integer, &a.Num, &a.Den, &x.significand, &m.integer.smt2Stream.tmp, &m.rational.NTmp)
		}
		serviceMust(t, m.rational.CleanupSMT2Stream())
		older.Buffer.OwnAbort()
		serviceMust(t, m.rational.Del(&a))
		serviceMust(t, m.Del(&x))
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
			t.Fatal("older formatter refunds", b.Used())
		}
		if matched {
			break
		}
	}
	if !reached {
		t.Fatal("no retained formatter COPY descriptor reached")
	}
}
