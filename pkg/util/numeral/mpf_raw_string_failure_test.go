// Source: src/util/mpf.cpp:1590-1606; src/util/mpz.cpp:1432-1435.
package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestArbitraryFloatRawStringEveryColdReservation(t *testing.T) {
	for _, tc := range []struct {
		sig string
		exp int64
	}{{"0", -127}, {"7", 128}, {"12345678901234567890123456789012345678901234567890", math.MinInt64}} {
		complete := false
		for ordinal := int64(0); ordinal < 256; ordinal++ {
			m, r, b := mpfNumeralHelpersFixture(t)
			x := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, true), exponent: tc.exp}
			serviceMust(t, m.integer.SetString(&x.significand, tc.sig))
			before := x
			var out utext.OwnedString
			b.FailAfter(ordinal)
			body := m.ToStringRaw(&x, &out)
			integerPhysicalAudit(t, m.integer, &x.significand)
			callerBytes := uint64(0)
			if body == nil {
				callerBytes = out.StorageBytes()
			}
			if x != before || b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes+callerBytes {
				t.Fatal("immediate raw string graph", tc, ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats(), callerBytes)
			}
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("unexpected raw string failure", tc, ordinal, body)
			}
			if body != nil {
				var later utext.OwnedString
				if e := m.ToStringRaw(&x, &later); !errors.Is(e, ErrNumericPending) {
					t.Fatal("stopped raw string replay", e)
				}
			}
			b.FailAfter(-1)
			serviceMust(t, m.CleanupRawString())
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
				if ref != 0 || m.rawString.active {
					t.Fatal("unproved source scratch hole", ref)
				}
				serviceMust(t, m.Close())
			}
			serviceMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal("raw string refunds", b.Used())
			}
			if body == nil {
				complete = true
				break
			}
		}
		if !complete {
			t.Fatal("raw string reservation horizon", tc)
		}
	}
}

func TestArbitraryFloatRawStringWarmStreamReservation(t *testing.T) {
	m, _, b := mpfNumeralHelpersFixture(t)
	x := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, false), exponent: math.MinInt64, significand: NewInt(1)}
	var first utext.OwnedString
	serviceMust(t, m.ToStringRaw(&x, &first))
	first.SourceDestroy()
	b.FailAfter(0)
	var second utext.OwnedString
	e := m.ToStringRaw(&x, &second)
	// Both text buffering and returned storage are Go-managed.
	serviceMust(t, e)
	if m.rawString.exponentStream.Buffer.Position != 20 || !second.Matches("[+ 1 -9223372036854775808 N]") {
		t.Fatal("Go output must finish independently of numeric arena failures", second.External)
	}
	second.SourceDestroy()
	b.FailAfter(-1)
	serviceMust(t, m.CleanupRawString())
}
