// Source: src/util/mpf.cpp:1228-1253.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func TestArbitraryFloatToIEEEBVEveryColdReservationAndAliases(t *testing.T) {

	for _, sb := range []uint32{2, 24, 120} {
		for _, inf := range []bool{false, true} {
			for _, alias := range []bool{false, true} {
				complete := false
				for ordinal := int64(0); ordinal < 512; ordinal++ {
					m, r, b := mpfNumeralHelpersFixture(t)
					n := NewInt(1)
					serviceMust(t, m.integer.Mul2kInPlace(&n, sb-1))
					serviceMust(t, m.integer.Dec(&n))
					x := ArbitraryFloat{format: arbitraryFloatFormat(32, sb, true), exponent: 1, significand: n}
					if inf {
						serviceMust(t, m.integer.SetInt32(&x.significand, 0))
						x.assignEbits(17)
						x.exponent = 1 << 16
					}
					separate := NewInt(7)
					serviceMust(t, m.integer.Mul2kInPlace(&separate, 100))
					out := &separate
					if alias {
						out = &x.significand
					}
					oldX := x
					b.FailAfter(ordinal)
					body := m.ToIEEEBVMPZ(&x, out)
					s := &m.ieeeBV
					integerPhysicalAudit(t, m.integer, &x.significand, &separate, &s.biasedExp)
					if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes || !alias && x != oldX {
						t.Fatal("immediate IEEE BV graph", sb, inf, alias, ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats())
					}
					if body != nil && !numericPhysicalReservationRefusal(body) {
						t.Fatal("unexpected refusal", sb, inf, alias, ordinal, body, "scope", s.invocation, s.bodyCause, s.sourceCause, "out", *out, "live", s.biasedExpLive)
					}
					if s.active {
						if e := m.ToIEEEBVMPZ(&x, out); !errors.Is(e, ErrNumericPending) {
							t.Fatal("stopped scope replay", e)
						}
					}
					b.FailAfter(-1)
					serviceMust(t, m.CleanupToIEEEBVMPZ())
					serviceMust(t, m.integer.CleanupRawNumeral(&separate))
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
						t.Fatal("IEEE BV refunds", b.Used())
					}
					if body == nil {
						complete = true
						break
					}
				}
				if !complete {
					t.Fatal("IEEE BV reservation horizon", sb, inf, alias)
				}
			}
		}
	}
}
