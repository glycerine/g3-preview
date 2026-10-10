// Source: src/util/mpf.cpp:1462-1481.
package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatRemEveryColdGeneralReservation(t *testing.T) {
	for _, sb := range []uint32{2, 24, 120} {
		for _, gap := range []int64{-2, -1, 0, 3*int64(sb) + 5} {
			for alias := 0; alias < 3; alias++ {
				complete := false
				for ordinal := int64(0); ordinal < 1024; ordinal++ {
					m, r, b := mpfNumeralHelpersFixture(t)
					n := NewInt(1)
					serviceMust(t, m.integer.Mul2kInPlace(&n, sb-1))
					serviceMust(t, m.integer.Dec(&n))
					var x, y, separate ArbitraryFloat
					serviceMust(t, m.SetComponents(&x, 17, sb, true, gap, &n))
					serviceMust(t, m.integer.MachineDiv2kInPlace(&n, 1))
					serviceMust(t, m.SetComponents(&y, 17, sb, false, 0, &n))
					serviceMust(t, m.integer.Del(&n))
					beforeX, beforeY := x, y
					out := []*ArbitraryFloat{&separate, &x, &y}[alias]
					b.FailAfter(ordinal)
					body := m.Rem(&x, &y, out)
					s, p := &m.rem, &m.partialRemainder
					nums := []*Int{&x.significand, &y.significand, &separate.significand, &s.a.significand, &s.b.significand, &m.RoundScope.sigma, &m.RoundScope.cap, &m.RoundScope.sticky}
					for i := range p.values {
						nums = append(nums, &p.values[i])
					}
					integerPhysicalAudit(t, m.integer, nums...)
					if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
						t.Fatal("immediate remainder graph", sb, gap, alias, ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats())
					}
					if alias != 1 && x != beforeX || alias != 2 && y != beforeY {
						t.Fatal("stopped const input changed", sb, gap, alias, ordinal)
					}
					if body != nil && !numericPhysicalReservationRefusal(body) {
						t.Fatal("unexpected cold refusal", sb, gap, alias, ordinal, body)
					}
					if s.active {
						if e := m.Rem(&x, &y, out); !errors.Is(e, ErrNumericPending) {
							t.Fatal("stopped parent replay", e)
						}
					}
					b.FailAfter(-1)
					if e := m.CleanupRem(); e != nil {
						t.Fatal("cleanup", sb, gap, alias, ordinal, body, e)
					}
					serviceMust(t, m.CleanupValue(&separate))
					serviceMust(t, m.CleanupValue(&x))
					serviceMust(t, m.CleanupValue(&y))
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
						t.Fatal("remainder refund", b.Used())
					}
					if body == nil {
						complete = true
						break
					}
				}
				if !complete {
					t.Fatal("remainder reservation horizon", sb, gap, alias)
				}
			}
		}
	}
}
