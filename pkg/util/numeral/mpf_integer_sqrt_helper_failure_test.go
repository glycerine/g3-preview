package numeral

// Source: src/util/mpf.cpp:951-1003.

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func TestMyMPZSqrtEveryColdReservation(t *testing.T) {
	for _, sb := range []uint32{2, 24, 80} {
		for _, alias := range []bool{false, true} {
			for _, odd := range []bool{false, true} {
				complete := false
				for ordinal := int64(0); ordinal < 2048; ordinal++ {
					m, r, b := mpfNumeralHelpersFixture(t)
					z := m.integer
					var input, separate Int
					serviceMust(t, z.SetInt32(&input, 1))
					shift := 2*sb + 6
					if odd {
						shift = 2*sb + 5
					}
					serviceMust(t, z.Mul2k(&input, shift, &input))
					serviceMust(t, z.Inc(&input))
					old := input
					out := &separate
					if alias {
						out = &input
					}
					b.FailAfter(ordinal)
					body := MyMPZSqrt(z, sb, odd, &input, out)
					s := &z.mySqrt
					integerPhysicalAudit(t, z, &input, &separate, &s.values[0], &s.values[1], &s.values[2], &s.values[3], &s.values[4])
					if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
						t.Fatal("immediate helper scope physical graph", sb, alias, odd, ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats())
					}
					if !alias && input != old {
						t.Fatal("stopped const input changed", ordinal)
					}
					if body != nil && !numericPhysicalReservationRefusal(body) {
						t.Fatal("unexpected cold failure", ordinal, body)
					}
					if s.active {
						if e := MyMPZSqrt(z, sb, odd, &input, out); !errors.Is(e, ErrNumericPending) {
							t.Fatal("stopped scope replay", e)
						}
					}
					b.FailAfter(-1)
					if e := z.CleanupMyMPZSqrt(); e != nil {
						t.Fatalf("cleanup sb=%d alias=%v odd=%v ordinal=%d body=%v scope=%+v: %v", sb, alias, odd, ordinal, body, *s, e)
					}
					serviceMust(t, z.CleanupRawNumeral(&separate))
					serviceMust(t, z.CleanupRawNumeral(&input))
					if source := m.SourceFinalize(); source != nil {
						z := m.integer
						if !errors.Is(source, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
							t.Fatal("unexpected source destructor stop", sb, alias, odd, ordinal, source, z.SourceFinalizeStage)
						}
						index := int(z.SourceFinalizeStage-2) / 2
						ref := z.Tmp[index]
						if z.SourceFinalizeStage%2 == 1 {
							ref = z.Arg[index]
						}
						if ref != 0 {
							t.Fatal("unproved actual source scratch hole", ordinal, ref)
						}
					}
					serviceMust(t, m.Close())
					serviceMust(t, r.Close())
					if b.Used() != 0 {
						t.Fatal("helper scope refund", ordinal, b.Used())
					}
					if body == nil {
						complete = true
						break
					}
				}
				if !complete {
					t.Fatal("helper reservation horizon", sb, alias, odd)
				}
			}
		}
	}
}
