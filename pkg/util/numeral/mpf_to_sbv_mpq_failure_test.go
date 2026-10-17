// Source: src/util/mpf.cpp:1186-1223; src/util/mpq.h:735-738.
package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryFloatToSBVMPQEveryColdReservationAndOutputPrefix(t *testing.T) {
	reachedOutputFailure := false
	for _, sb := range []uint32{2, 24, 120} {
		for _, exp := range []int64{-1, 160} {
			for _, mode := range mpfIntegralModes {
				complete := false
				for ordinal := int64(0); ordinal < 512; ordinal++ {
					m, r, b := mpfNumeralHelpersFixture(t)
					n := NewInt(1)
					serviceMust(t, m.integer.Mul2kInPlace(&n, sb-1))
					serviceMust(t, m.integer.Dec(&n))
					x := ArbitraryFloat{format: arbitraryFloatFormat(17, sb, true), exponent: exp, significand: n}
					out := Rational{Num: NewInt(7), Den: NewInt(1)}
					serviceMust(t, m.integer.Mul2kInPlace(&out.Den, 100))
					oldX, oldDen := x, out.Den
					b.FailAfter(ordinal)
					body := m.ToSBVMPQ(mode, &x, &out)
					s := &m.sbv
					integerPhysicalAudit(t, m.integer, &x.significand, &out.Num, &out.Den, &s.a.significand, &s.z)
					if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes || x != oldX {
						t.Fatal("immediate SBV graph", sb, exp, mode, ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats())
					}
					if body != nil && !numericPhysicalReservationRefusal(body) {
						t.Fatal("unexpected refusal", sb, exp, mode, ordinal, body)
					}
					if body != nil && s.outputCopy.Frame != nil {
						reachedOutputFailure = true
						if out.Den != oldDen {
							t.Fatal("denominator reset before refused numerator COPY", sb, exp, mode, ordinal)
						}
					}
					if s.active {
						if e := m.ToSBVMPQ(mode, &x, &out); !errors.Is(e, ErrNumericPending) {
							t.Fatal("stopped scope replay", e)
						}
					}
					b.FailAfter(-1)
					serviceMust(t, m.CleanupToSBVMPQ())
					serviceMust(t, m.integer.CleanupRawNumeral(&out.Num))
					serviceMust(t, m.integer.CleanupRawNumeral(&out.Den))
					serviceMust(t, m.CleanupValue(&x))
					z := m.integer
					if source := m.SourceFinalize(); source != nil {
						if !errors.Is(source, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
							t.Fatal("unexpected source stop", source, z.SourceFinalizeStage)
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
					serviceMust(t, r.Close())
					if b.Used() != 0 {
						t.Fatal("SBV refunds", b.Used())
					}
					if body == nil {
						complete = true
						break
					}
				}
				if !complete {
					t.Fatal("SBV reservation horizon", sb, exp, mode)
				}
			}
		}
	}
	if !reachedOutputFailure {
		t.Fatal("output numerator COPY failure was not exercised")
	}
}

func TestArbitraryFloatToSBVMPQWarmInlineAndModeBranchAuthority(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	var x ArbitraryFloat
	serviceMust(t, m.SetDouble(&x, 8, 3, 1.5))
	out := NewRational()
	serviceMust(t, m.ToSBVMPQ(ArbitraryFloatNearestEven, &x, &out))
	used, state := b.Used(), r.State()
	b.FailAfter(0)
	serviceMust(t, m.ToSBVMPQ(ArbitraryFloatNearestAway, &x, &out))
	if out.Num != NewInt(2) || out.Den != NewInt(1) || b.Used() != used || r.State() != state {
		t.Fatal("warm inline rounding allocated", out)
	}
	before := out
	copied := m.sbv
	copied.hooks.Policy = &copied
	if e := copied.run(ArbitraryFloatNearestEven, &x, &out); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("copied context authority", e)
	}
	if e := m.sbv.run(ArbitraryFloatNearestEven, &x, &out); !errors.Is(e, ErrNumericOwner) || out != before {
		t.Fatal("ended context authority", e, out)
	}
	b.FailAfter(-1)
	// Source switches on mode only for a negative effective exponent.
	if !modularSourceDebug {
		if e := m.ToSBVMPQ(ArbitraryFloatRoundingMode(255), &x, &out); !errors.Is(e, diagnostic.ErrDebugNativeUnreachable) {
			t.Fatal("negative exponent invalid mode", e)
		}
		serviceMust(t, m.CleanupToSBVMPQ())
	}
	x.exponent = 4
	serviceMust(t, m.ToSBVMPQ(ArbitraryFloatRoundingMode(255), &x, &out))
	if out.Num != NewInt(24) {
		t.Fatal("nonnegative exponent must bypass mode switch", out)
	}
	serviceMust(t, m.rational.Del(&out))
	serviceMust(t, m.Del(&x))
}

func TestArbitraryFloatToSBVMPQReachedExponentOverflow(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	x := ArbitraryFloat{format: arbitraryFloatFormat(11, 120, false), exponent: math.MinInt64, significand: NewInt(3)}
	serviceMust(t, m.integer.Mul2kInPlace(&x.significand, 100))
	before := x
	out := Rational{Num: NewInt(7), Den: NewInt(9)}
	oldOut := out
	if e := m.ToSBVMPQ(ArbitraryFloatNearestEven, &x, &out); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("reached signed subtraction", e)
	}
	if x != before || out != oldOut || m.sbv.copyZ.Frame == nil || m.sbv.aLive || m.sbv.zLive || !m.sbv.active {
		t.Fatal("overflow must follow local integer COPY and precede output SET", x, out, m.sbv)
	}
	serviceMust(t, m.CleanupToSBVMPQ())
	serviceMust(t, m.rational.Del(&out))
	serviceMust(t, m.Del(&x))
}
