// Source: src/util/mpf.cpp:1635-1693; actual selected allocation prefixes.
package numeral

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	utext "github.com/glycerine/g3/pkg/util/text"
	"math/big"
	"testing"
)

func TestArbitraryFloatBinaryStringEveryColdReservation(t *testing.T) {
	huge := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 130), big.NewInt(37)).String()
	for _, tc := range []struct {
		eb, sb       uint32
		exp          int64
		sig          string
		upper, lower uint32
	}{
		{8, 120, 17, huge, 3, 2}, {8, 120, -127, "1", 0, 0}, {20, 120, 1 << 19, "1", 0, 0},
		{20, 120, 1 << 19, "0", 0, 0}, {20, 120, -((1 << 19) - 1), "0", 0, 0},
		{33, 24, 0, "5", 0, 0},
	} {
		complete := false
		for ordinal := int64(0); ordinal < 256; ordinal++ {
			m, r, b := mpfNumeralHelpersFixture(t)
			x := ArbitraryFloat{format: arbitraryFloatFormat(tc.eb, tc.sb, true), exponent: tc.exp}
			serviceMust(t, m.integer.SetString(&x.significand, tc.sig))
			before := x
			var out utext.OwnedString
			b.FailAfter(ordinal)
			body := m.ToStringBinary(&x, tc.upper, tc.lower, &out)
			s := &m.binaryString
			integerPhysicalAudit(t, m.integer, &x.significand, &s.tmp)
			callerBytes := uint64(0)
			if body == nil {
				callerBytes = out.StorageBytes()
			}
			if x != before || b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes+callerBytes {
				t.Fatal("binary immediate graph", tc, ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats(), callerBytes)
			}
			if body != nil {
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected cold binary error", tc, ordinal, body)
				}
				var later utext.OwnedString
				if e := m.ToStringBinary(&x, tc.upper, tc.lower, &later); !errors.Is(e, ErrNumericPending) {
					t.Fatal("failed binary replay", e)
				}
			}
			b.FailAfter(-1)
			serviceMust(t, m.CleanupBinaryString())
			out.OwnAbort()
			serviceMust(t, m.CleanupValue(&x))
			if e := m.Close(); e != nil {
				z := m.integer
				if !errors.Is(e, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
					t.Fatal("binary class stop", e)
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
				t.Fatal("binary refunds", tc, ordinal, b.Used())
			}
			if body == nil {
				complete = true
				break
			}
		}
		if !complete {
			t.Fatal("binary cold horizon", tc)
		}
	}
}

func TestArbitraryFloatBinaryStringWarmAndLaterCalls(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	x := ArbitraryFloat{format: arbitraryFloatFormat(2, 2, false), exponent: 0, significand: NewInt(1)}
	var first utext.OwnedString
	serviceMust(t, m.ToStringBinary(&x, 0, 0, &first))
	if !first.Matches("#b0 #b01 #b0.1") {
		t.Fatal("small binary", first)
	}
	first.SourceDestroy()
	used, stats := b.Used(), r.Stats()
	b.FailAfter(0)
	var second utext.OwnedString
	serviceMust(t, m.ToStringBinary(&x, 0, 0, &second))
	if b.Used() != used || r.Stats() != stats || !second.Matches("#b0 #b01 #b0.1") {
		t.Fatal("warm binary allocated")
	}
	b.FailAfter(-1)
	// An ended binary adapter must not adopt later COPY, division, or text.
	later := NewInt(0)
	serviceMust(t, m.integer.SetString(&later, "123456789012345678901234567890"))
	before := later
	used = b.Used()
	div := m.integer.DivisionRaw.Invocation
	serviceMust(t, m.CleanupBinaryString())
	if later != before || b.Used() != used || m.integer.DivisionRaw.Invocation != div || !second.Matches("#b0 #b01 #b0.1") {
		t.Fatal("ended binary adopted later call")
	}
	serviceMust(t, m.integer.Del(&later))
	second.SourceDestroy()
}

func TestArbitraryFloatBinaryStringEntryAuthority(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	x := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, false), significand: NewInt(1)}
	if e := m.ToStringBinary(&x, 0, 0, nil); !errors.Is(e, utext.ErrOwnedStringOwner) {
		t.Fatal("nil return", e)
	}
	var out utext.OwnedString
	serviceMust(t, out.Construct(m.budget, "existing"))
	if e := m.ToStringBinary(&x, 0, 0, &out); !errors.Is(e, utext.ErrOwnedStringOwner) || !out.Matches("existing") {
		t.Fatal("live return", e)
	}
	out.SourceDestroy()
	copyManager := *m
	var later utext.OwnedString
	if e := copyManager.ToStringBinary(&x, 0, 0, &later); !errors.Is(e, umemory.ErrRuntimeOwner) {
		t.Fatal("copied manager", e)
	}
	m.binaryString.invocation = ^uint64(0)
	if e := m.ToStringBinary(&x, 0, 0, &later); !errors.Is(e, ErrNumericPending) || later.Self != nil {
		t.Fatal("counter wrap", e)
	}
	m.binaryString.invocation = 0
	if modularSourceDebug {
		x.significand = NewInt(-1)
		if e := m.ToStringBinary(&x, 0, 0, &later); !errors.Is(e, ErrIntegerAssertion) {
			t.Fatal("source nonnegative assertion", e)
		}
		serviceMust(t, m.CleanupBinaryString())
	}
}

func TestArbitraryFloatBinaryStringRefusedDivisionKeepsOlderRoots(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 80; ordinal++ {
		budget := arena.NewBudget(1 << 28)
		m, e := NewArbitraryFloatManager(IntegerConfig{Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: budget, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
		serviceMust(t, e)
		z := m.integer
		x := ArbitraryFloat{format: arbitraryFloatFormat(8, 3, false), exponent: -160, significand: NewInt(3)}
		var warm utext.OwnedString
		serviceMust(t, m.ToStringBinary(&x, 0, 0, &warm))
		warm.SourceDestroy()
		a, b, q := NewInt(-1), NewInt(1), NewInt(0)
		serviceMust(t, z.Mul2kInPlace(&a, 4000))
		serviceMust(t, z.Mul2kInPlace(&b, 2000))
		serviceMust(t, z.Inc(&b))
		budget.FailAfter(ordinal)
		body := z.Div(&a, &b, &q)
		old := &z.DivisionRaw
		matched := body != nil && old.Active && NumericAllocationPending(&old.Numeric)
		budget.FailAfter(-1)
		if matched {
			reached = true
			roots, values, inv, used := old.Numeric, old.Values, old.Invocation, budget.Used()
			var out utext.OwnedString
			if e := m.ToStringBinary(&x, 0, 0, &out); !errors.Is(e, ErrNumericPending) {
				t.Fatal("older division refusal", ordinal, e)
			}
			if m.binaryString.division.Invocation != 0 {
				t.Fatal("refused division adopted")
			}
			serviceMust(t, m.CleanupBinaryString())
			if !old.Active || old.Numeric != roots || old.Values != values || old.Invocation != inv || budget.Used() != used {
				t.Fatal("binary cleanup touched older division")
			}
			integerPhysicalAudit(t, z, &a, &b, &q, &x.significand, &m.binaryString.tmp)
		}
		serviceMust(t, z.CleanupDivision())
		for _, v := range []*Int{&a, &b, &q} {
			serviceMust(t, z.CleanupRawNumeral(v))
		}
		if e := m.SourceFinalize(); e != nil {
			if !errors.Is(e, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
				t.Fatal("older division class stop", e)
			}
			i := int(z.SourceFinalizeStage-2) / 2
			ref := z.Tmp[i]
			if z.SourceFinalizeStage%2 == 1 {
				ref = z.Arg[i]
			}
			if ref != 0 {
				t.Fatal("unproved old division scratch hole", ref)
			}
		}
		serviceMust(t, m.Close())
		if budget.Used() != 0 {
			t.Fatal("old division refunds", budget.Used())
		}
		if matched {
			break
		}
	}
	if !reached {
		t.Fatal("no actual retained RAW division allocation reached")
	}
}
