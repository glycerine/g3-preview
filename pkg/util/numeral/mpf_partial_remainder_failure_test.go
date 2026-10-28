package numeral

// Source: src/util/mpf.cpp:1272-1436.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatPartialRemainderEveryColdReservation(t *testing.T) {
	for _, sb := range []uint32{2, 7, 24, 80, 120} {
		for _, d := range []int64{-1, 0, 1, int64(sb - 1), int64(sb), int64(sb + 9)} {
			for _, same := range []bool{false, true} {
				if same && d != 0 {
					continue
				}
				complete := false
				for ordinal := int64(0); ordinal < 512; ordinal++ {
					m, r, b := mpfNumeralHelpersFixture(t)
					z := m.integer
					var x, y ArbitraryFloat
					serviceMust(t, z.SetInt32(&x.significand, 1))
					serviceMust(t, z.Mul2kInPlace(&x.significand, sb-1))
					serviceMust(t, z.Copy(&y.significand, &x.significand))
					serviceMust(t, z.Inc(&y.significand))
					x.format = arbitraryFloatFormat(11, sb, true)
					x.exponent = d - 10
					y.format = arbitraryFloatFormat(11, sb, false)
					y.exponent = -10
					other := &y
					if same {
						other = &x
					}
					before := y
					b.FailAfter(ordinal)
					body := m.PartialRemainder(&x, other, d, d >= int64(sb))
					s := &m.partialRemainder
					nums := []*Int{&x.significand, &y.significand}
					for i := range s.values {
						nums = append(nums, &s.values[i])
					}
					integerPhysicalAudit(t, z, nums...)
					if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
						t.Fatal("immediate actual remainder graph", sb, d, same, ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats())
					}
					if y != before {
						t.Fatal("borrowed divisor changed", sb, d, same, ordinal)
					}
					if body != nil && !numericPhysicalReservationRefusal(body) {
						t.Fatal("unexpected cold failure", sb, d, same, ordinal, body)
					}
					if s.active {
						if e := m.PartialRemainder(&x, other, d, d >= int64(sb)); !errors.Is(e, ErrNumericPending) {
							t.Fatal("stopped caller replay", e)
						}
					}
					b.FailAfter(-1)
					serviceMust(t, m.CleanupPartialRemainder())
					serviceMust(t, m.CleanupValue(&y))
					serviceMust(t, m.CleanupValue(&x))
					if source := m.SourceFinalize(); source != nil {
						if !errors.Is(source, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
							t.Fatal("unexpected source destructor stop", sb, d, same, ordinal, source, z.SourceFinalizeStage)
						}
						index := int(z.SourceFinalizeStage-2) / 2
						ref := z.Tmp[index]
						if z.SourceFinalizeStage%2 == 1 {
							ref = z.Arg[index]
						}
						if ref != 0 {
							t.Fatal("unproved actual scratch hole", ref)
						}
					}
					serviceMust(t, m.Close())
					serviceMust(t, r.Close())
					if b.Used() != 0 {
						t.Fatal("remainder scope refund", sb, d, same, ordinal, b.Used())
					}
					if body == nil {
						complete = true
						break
					}
				}
				if !complete {
					t.Fatal("remainder reservation horizon", sb, d, same)
				}
			}
		}
	}
}
