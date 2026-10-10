// Source: src/util/mpq.cpp:156-185; actual RAW declarations and end-label cleanup.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"strings"
	"testing"
)

func TestRationalDecimalStreamEveryColdReservation(t *testing.T) {
	for _, tc := range [][2]string{{"-2147483648", "3"}, {"123456789012345678901234567890", "1"}, {"3", "98765432109876543210987654321"}, {"-123456789012345678901234567890", "98765432109876543210987654321"}} {
		complete := false
		for ordinal := int64(0); ordinal < 512; ordinal++ {
			m, r, b := mpfNumeralHelpersFixture(t)
			a := NewRational()
			serviceMust(t, m.integer.SetString(&a.Num, tc[0]))
			serviceMust(t, m.integer.SetString(&a.Den, tc[1]))
			before := a
			var out OwnedStringStream
			serviceMust(t, out.Construct(b))
			serviceMust(t, out.AppendCString("prefix:"))
			b.FailAfter(ordinal)
			body := m.rational.DisplayDecimalInto(&out, &a, 12, false)
			s := &m.rational.decimalStream
			integerPhysicalAudit(t, m.integer, &a.Num, &a.Den, &s.values[0], &s.values[1], &s.values[2], &s.values[3])
			streamBytes := out.Buffer.String.StorageBytes()
			if a != before || b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes+streamBytes {
				t.Fatal("immediate decimal stream graph", tc, ordinal, body, b.Used(), streamBytes)
			}
			got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
			serviceMust(t, e)
			if !strings.HasPrefix(got, "prefix:") {
				t.Fatal("borrowed prefix lost", got)
			}
			if body != nil {
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected decimal failure", tc, ordinal, body)
				}
				if s.bodySucceeded || s.sourceCursor != 0 {
					t.Fatal("failed RAW body executed source end label", ordinal)
				}
				if e := m.rational.DisplayDecimalInto(&out, &a, 12, false); !errors.Is(e, ErrNumericPending) {
					t.Fatal("stopped decimal replay", e)
				}
			}
			b.FailAfter(-1)
			serviceMust(t, m.rational.CleanupDecimalStream())
			after, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
			serviceMust(t, e)
			if after != got {
				t.Fatal("cleanup changed caller output")
			}
			out.Buffer.OwnAbort()
			serviceMust(t, m.rational.Del(&a))
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
					t.Fatal("unproved scratch hole", ref)
				}
				serviceMust(t, m.Close())
			}
			serviceMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal("decimal refunds", b.Used())
			}
			if body == nil {
				complete = true
				break
			}
		}
		if !complete {
			t.Fatal("decimal reservation horizon", tc)
		}
	}
}

func TestRationalDecimalStreamEveryInsertionException(t *testing.T) {
	prefixes := []string{"", "-", "-0", "-0.", "-0.1", "-0.12", "-0.125", "-0.1250"}
	for fail := 1; fail <= len(prefixes); fail++ {
		m, _, b := mpfNumeralHelpersFixture(t)
		a := Rational{Num: NewInt(-12501), Den: NewInt(100000)}
		var out OwnedStringStream
		serviceMust(t, out.Construct(b))
		boom := errors.New("actual decimal ostream exception")
		stream := smt2FailingStream{inner: &out, fail: fail, failure: boom}
		body := m.rational.DisplayDecimalInto(&stream, &a, 4, false)
		if !errors.Is(body, boom) {
			t.Fatal("selected decimal insertion missing", fail, body)
		}
		s := &m.rational.decimalStream
		expectedCount := uint8(4)
		if fail == 1 {
			expectedCount = 3
		}
		if s.sourceCursor != 0 || s.count != expectedCount || s.bodySucceeded {
			t.Fatal("unreached RAW end label", fail, s.count, s.sourceCursor)
		}
		got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
		serviceMust(t, e)
		if got != prefixes[fail-1] {
			t.Fatal("exact decimal prefix", fail, got)
		}
		serviceMust(t, m.rational.CleanupDecimalStream())
		after, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
		serviceMust(t, e)
		if after != got || s.sourceCursor != 0 || s.ownCursor != 4 {
			t.Fatal("owning abort replayed source or destroyed output", fail)
		}
		out.Buffer.OwnAbort()
	}
}

func TestRationalDecimalStreamWarmAndLaterChild(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	a := Rational{Num: NewInt(-1), Den: NewInt(3)}
	var first OwnedStringStream
	serviceMust(t, first.Construct(b))
	serviceMust(t, m.rational.DisplayDecimalInto(&first, &a, 3, false))
	first.Buffer.OwnAbort()
	used, state := b.Used(), r.State()
	var out OwnedStringStream
	serviceMust(t, out.Construct(b))
	b.FailAfter(0)
	serviceMust(t, m.rational.DisplayDecimalInto(&out, &a, 3, false))
	got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
	serviceMust(t, e)
	if got != "-0.333?" || b.Used() != used || r.State() != state {
		t.Fatal("warm decimal allocated", got, b.Used(), used)
	}
	b.FailAfter(-1)
	var later OwnedStringStream
	serviceMust(t, later.Construct(b))
	serviceMust(t, m.rational.DisplaySMT2Into(&later, &a, false))
	invocation := m.rational.smt2Stream.invocation
	serviceMust(t, m.rational.CleanupDecimalStream())
	if m.rational.smt2Stream.invocation != invocation {
		t.Fatal("ended decimal touched later child")
	}
	out.Buffer.OwnAbort()
	later.Buffer.OwnAbort()
}

func TestRationalDecimalStreamReachedZeroDenominatorAndAuthority(t *testing.T) {
	m, _, b := mpfNumeralHelpersFixture(t)
	a := Rational{Num: NewInt(-7), Den: NewInt(0)}
	var out OwnedStringStream
	serviceMust(t, out.Construct(b))
	if e := m.rational.DisplayDecimalInto(nil, &a, 1, false); !errors.Is(e, ErrOwnedStringOwner) || m.rational.decimalStream.invocation != 0 {
		t.Fatal("nil stream began invocation", e)
	}
	body := m.rational.DisplayDecimalInto(&out, &a, 1, false)
	s := &m.rational.decimalStream
	if !errors.Is(body, ErrDivideByZero) || s.count != 4 || s.sourceCursor != 0 || s.values[1].Val != 7 || out.Buffer.Position != 1 {
		t.Fatal("zero denominator failed before source copies/sign/ten", body, s.count)
	}
	copy := *s
	if copy.valid() || copy.NumericRoots(m.integer, nil) != nil || copy.NumericExecuting() {
		t.Fatal("copied scope gained authority")
	}
	serviceMust(t, m.rational.CleanupDecimalStream())
	out.Buffer.OwnAbort()
	original := s.invocation
	s.invocation = ^uint64(0)
	if e := m.rational.DisplayDecimalInto(&out, &a, 1, false); !errors.Is(e, ErrNumericPending) || s.active {
		t.Fatal("decimal counter wrapped", e)
	}
	s.invocation = original
}

func TestRationalDecimalStreamInlineCallKeepsOlderCopyAllocation(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 60; ordinal++ {
		b := arena.NewBudget(1 << 28)
		m, e := NewArbitraryFloatManager(IntegerConfig{Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
		serviceMust(t, e)
		source := NewInt(1)
		serviceMust(t, m.integer.Mul2kInPlace(&source, 4000))
		var target Int
		var receipt integerCopyAllocationReceipt
		b.FailAfter(ordinal)
		body := m.integer.CopyFromOwnerWithReceipt(&target, m.integer, &source, &receipt)
		matched := body != nil && m.integer.MatchesCopyAllocationInvocation(&receipt) && m.integer.CopyAllocation.Roots.Cell.Object != (umemory.ObjectRef{})
		b.FailAfter(-1)
		if matched {
			reached = true
			roots, invocation := m.integer.CopyAllocation.Roots, m.integer.CopyAllocation.invocation
			var out OwnedStringStream
			serviceMust(t, out.Construct(b))
			a := Rational{Num: NewInt(1), Den: NewInt(3)}
			serviceMust(t, m.rational.DisplayDecimalInto(&out, &a, 2, false))
			used := b.Used()
			serviceMust(t, m.rational.CleanupDecimalStream())
			if m.integer.CopyAllocation.Roots != roots || m.integer.CopyAllocation.invocation != invocation || !m.integer.MatchesCopyAllocationInvocation(&receipt) || b.Used() != used {
				t.Fatal("inline decimal adopted unrelated older COPY")
			}
			got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
			serviceMust(t, e)
			if got != "0.33?" {
				t.Fatal("inline decimal with older pending owner", got)
			}
			integerPhysicalAudit(t, m.integer, &source, &target)
			out.Buffer.OwnAbort()
		}
		serviceMust(t, m.integer.CleanupRawNumeral(&target))
		serviceMust(t, m.integer.CleanupCopyAllocationInvocation(&receipt))
		serviceMust(t, m.integer.Del(&source))
		if e := m.SourceFinalize(); e != nil {
			z := m.integer
			if !errors.Is(e, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
				t.Fatal(e)
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
			t.Fatal("older COPY refunds", b.Used())
		}
		if matched {
			break
		}
	}
	if !reached {
		t.Fatal("no retained older COPY descriptor reached")
	}
}
