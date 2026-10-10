// Source: src/util/mpq.cpp:123-127; actual owned strings and integer display.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestRationalOwnedStringEveryColdReservation(t *testing.T) {
	for _, tc := range [][2]string{{"7", "1"}, {"0", "0"}, {"123456789012345678901234567890", "1"}, {"3", "98765432109876543210987654321"}, {"-123456789012345678901234567890", "98765432109876543210987654321"}} {
		complete := false
		for ordinal := int64(0); ordinal < 256; ordinal++ {
			m, r, b := mpfNumeralHelpersFixture(t)
			a := NewRational()
			serviceMust(t, m.integer.SetString(&a.Num, tc[0]))
			serviceMust(t, m.integer.SetString(&a.Den, tc[1]))
			before := a
			var out utext.OwnedString
			b.FailAfter(ordinal)
			body := m.rational.ToOwnedString(&a, &out)
			integerPhysicalAudit(t, m.integer, &a.Num, &a.Den)
			callerBytes := uint64(0)
			if body == nil {
				callerBytes = out.StorageBytes()
			}
			if a != before || b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes+callerBytes {
				t.Fatal("immediate rational string graph", tc, ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats(), callerBytes)
			}
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("unexpected formatting failure", tc, ordinal, body)
			}
			if body != nil {
				var later utext.OwnedString
				if e := m.rational.ToOwnedString(&a, &later); !errors.Is(e, ErrNumericPending) {
					t.Fatal("stopped formatter replay", e)
				}
			}
			b.FailAfter(-1)
			serviceMust(t, m.rational.CleanupOwnedString())
			out.OwnAbort()
			serviceMust(t, m.integer.CleanupRawNumeral(&a.Num))
			serviceMust(t, m.integer.CleanupRawNumeral(&a.Den))
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
				if ref != 0 || m.rational.stringScope.active {
					t.Fatal("unproved source scratch hole", ref)
				}
				serviceMust(t, m.Close())
			}
			serviceMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal("rational string refunds", b.Used())
			}
			if body == nil {
				complete = true
				break
			}
		}
		if !complete {
			t.Fatal("rational string reservation horizon", tc)
		}
	}
}

func TestRationalOwnedStringWarmInlineAndReturnOwnership(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	a := Rational{Num: NewInt(7), Den: NewInt(3)}
	var first utext.OwnedString
	serviceMust(t, m.rational.ToOwnedString(&a, &first))
	first.SourceDestroy()
	used, state := b.Used(), r.State()
	b.FailAfter(0)
	var out utext.OwnedString
	serviceMust(t, m.rational.ToOwnedString(&a, &out))
	if !out.Matches("7/3") || b.Used() != used || r.State() != state {
		t.Fatal("warm inline rational string allocated", b.Used(), used)
	}
	out.SourceDestroy()
	b.FailAfter(-1)
	serviceMust(t, m.integer.SetString(&a.Num, "123456789012345678901234567890"))
	serviceMust(t, m.integer.SetString(&a.Den, "98765432109876543210987654321"))
	out = utext.OwnedString{}
	serviceMust(t, m.rational.ToOwnedString(&a, &out))
	want := "123456789012345678901234567890/98765432109876543210987654321"
	serviceMust(t, m.rational.Del(&a))
	serviceMust(t, m.Close())
	serviceMust(t, r.Close())
	if !out.Matches(want) || b.Used() != out.StorageBytes() {
		t.Fatal("returned string was not independent of numeral owner", b.Used(), out.StorageBytes())
	}
	out.SourceDestroy()
	if b.Used() != 0 {
		t.Fatal("returned string refunds", b.Used())
	}
}
